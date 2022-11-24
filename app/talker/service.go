package talker

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/streamwest-1629/chatspace/app/voicevox"
	"go.uber.org/zap"
)

type ServiceController struct {
	logger  *zap.Logger
	quit    chan<- *sync.WaitGroup
	discord *discordgo.Session
	app     *discordgo.Application
}

func NewService(baseLogger *zap.Logger, discordToken string, voicevoxApp *voicevox.VoiceVox) (*ServiceController, error) {

	baseLogger = baseLogger.With(zap.String("package", "talker"))

	baseLogger.Info("initialize application")
	sess, err := discordgo.New(fmt.Sprintf("Bot %s", discordToken))
	if err != nil {
		return nil, fmt.Errorf("failed create a new discord session: %w", err)
	}

	app, err := sess.Application("@me")
	baseLogger.Info("application status", zap.String("applicationID", app.ID))
	if err != nil {
		return nil, fmt.Errorf("failed get application status: %w", err)
	}

	messageCreateListener := make(chan discordgo.MessageCreate)
	voiceStateUpdateListener := make(chan discordgo.VoiceStateUpdate)
	quit := make(chan *sync.WaitGroup)

	// Add discord session handler
	sess.AddHandler(func(_ *discordgo.Session, arg *discordgo.MessageCreate) {
		if arg.Author.ID != app.ID {
			messageCreateListener <- *arg
		}
	})
	sess.AddHandler(func(_ *discordgo.Session, arg *discordgo.VoiceStateUpdate) {
		if arg.Member.User.ID != app.ID {
			voiceStateUpdateListener <- *arg
		}
	})

	sc := &ServiceController{
		logger:  baseLogger.With(zap.String("feature", "controller")),
		quit:    quit,
		discord: sess,
		app:     app,
	}

	go func() {
		logger := baseLogger.With(zap.String("feature", "eventListener"))
		memberJoinVCs := map[string]map[string]string{}
		serverStatuses := map[string]*joinedServerStatus{}

		for {
			select {
			case wg := <-quit:
				defer wg.Done()
				return
			case event := <-messageCreateListener:

				currentChannelId := ""
				guildMemberJoinVCs, joined := memberJoinVCs[app.GuildID]
				if joined {
					currentChannelId = guildMemberJoinVCs[event.Author.ID]
				}

				if sc.IsMentioned(event) {
					if currentChannelId == "" {
						SendMessage(
							sess, logger, event.ID, event.ChannelID,
							strings.Join([]string{"😑", "ボイスチャンネルに入室しているときのみ利用可能です"}, " "),
							nil,
						)
						break
					}

					serverStatus, exist := serverStatuses[event.GuildID]
					if exist {
						if serverStatus.voiceConn.ChannelID != currentChannelId {
							SendMessage(
								sess, logger, event.ID, event.ChannelID,
								strings.Join([]string{"💔", "他のボイスチャンネルにいるので動けません"}, " "),
								nil,
							)
						}
					} else {
						if ss, err := newJoinedServerStatus(baseLogger, sess, voicevoxApp, &event, currentChannelId); err != nil {
							logger.Error("failed start server", zap.Error(err))
						} else {
							serverStatus = ss
							serverStatuses[event.GuildID] = serverStatus
						}
					}

					content := event.Content
					for _, mention := range event.Mentions {
						content = strings.ReplaceAll(content, mention.Mention(), " ")
					}
					for _, mention := range event.MentionChannels {
						content = strings.ReplaceAll(content, mention.Mention(), " ")
					}
					for _, mention := range event.MentionRoles {
						content = strings.ReplaceAll(content, mention, " ")
					}

					switch {
					case strings.Contains(content, "--help"):
						SendMessage(
							sess, logger, event.ID, event.ChannelID,
							strings.Join([]string{"😶", "ヘルプ"}, " "),
							&discordgo.MessageEmbed{
								Description: strings.Join([]string{
									"<このボットへのメンション> <コマンド> （その他）",
									"`--list-voice`: 使用できるボイスの一覧を表示",
									"`--set-voice`: ボイスを設定",
									"```",
									"--set-voice <ボイスを設定するメンバーへのメンション>(...) <設定するボイスの名前>",
									"```",
									"`--leave`: Bot退出",
									"`--help`: ヘルプ表示",
								}, "\n"),
							},
						)
					case strings.Contains(content, "--leave"):
						serverStatus.Close()
						delete(serverStatuses, event.GuildID)
					case strings.Contains(content, "--list-voice"):
						speakers, err := voicevoxApp.GetSpeakers("", true)
						if err != nil {
							logger.Error("failed get voicevox speakers", zap.Error(err))
							break
						}

						speakerNames := []string{}
						for _, speaker := range speakers {
							speakerNames = append(speakerNames, fmt.Sprintf("- %s", speaker.Name))
						}

						SendMessage(
							sess, logger, event.ID, event.ChannelID,
							strings.Join([]string{"🥳", "担当可能な声の一覧"}, " "),
							&discordgo.MessageEmbed{
								Description: strings.Join(speakerNames, "\n"),
								Footer: &discordgo.MessageEmbedFooter{
									Text: "ボイスを設定するときは: --set-voice <声を設定するメンバーへのメンション>(...) <設定する声の名前>",
								},
							},
						)

					case strings.Contains(content, "--set-voice"):

						speakers, err := voicevoxApp.GetSpeakers("", false)
						if err != nil {
							logger.Error("failed get voicevox speakers", zap.Error(err))
							break
						}
						mentionedUser := []*discordgo.User{}
						for _, mention := range event.Mentions {
							if !mention.Bot {
								mentionedUser = append(mentionedUser, mention)
							}
						}

						if len(mentionedUser) == 0 {
							SendMessage(
								sess, logger, event.ID, event.ChannelID,
								strings.Join([]string{"🤔", "声を設定するメンバーを指定してください"}, " "),
								&discordgo.MessageEmbed{
									Description: strings.Join([]string{
										"声を設定するには，<Botへのメンション> --set-voice <声を設定するメンバーへのメンション>(...) <設定する声の名前>を指定していください",
										strings.Join([]string{"例:", "--set-voice", event.Author.Mention(), speakers[rand.Intn(len(speakers))].Name}, " "),
									}, "\n"),
								},
							)
							break
						}

						searchName := func() string {
							if matched := regexp.MustCompile(`\s+([^\s]+?)\s*$`).FindStringSubmatch(content); len(matched) == 2 {
								return matched[1]
							} else {
								return "**Unknown**"
							}
						}()

						speakers, err = voicevoxApp.GetSpeakers(searchName, false)
						if err != nil {
							logger.Error("failed get voicevox speakers", zap.Error(err))
							break
						}

						if len(speakers) == 0 {
							SendMessage(
								sess, logger, event.ID, event.ChannelID,
								strings.Join([]string{"🤯", "当てはまる声がみつかりませんでした「" + searchName + "」"}, " "),
								nil,
							)
							break
						}

						for _, mention := range event.Mentions {
							if !mention.Bot {
								serverStatus.SetVoiceSpeaker(&event, mention.ID, speakers[0])
							}
						}
					}

				} else {
					if !joined {
						break
					}

					if serverStatus, exist := serverStatuses[event.GuildID]; !exist {
						break
					} else if serverStatus.voiceConn.ChannelID == currentChannelId {
						serverStatus.Speak(&event)
					}
				}

			case event := <-voiceStateUpdateListener:
				guildMemberJoinVCs, exist := memberJoinVCs[event.GuildID]
				if !exist {
					guildMemberJoinVCs = map[string]string{}
					memberJoinVCs[event.GuildID] = guildMemberJoinVCs
				}

				if event.BeforeUpdate == nil {
					event.BeforeUpdate = &discordgo.VoiceState{}
				}

				if event.ChannelID == "" {
					delete(guildMemberJoinVCs, event.Member.User.ID)
				} else {
					guildMemberJoinVCs[event.Member.User.ID] = event.Member.User.ID
				}

				if ss, exist := serverStatuses[event.GuildID]; exist {
					currentChannelIdCount := 0
					for _, channelId := range guildMemberJoinVCs {
						if ss.voiceConn.ChannelID == channelId {
							currentChannelIdCount++
						}
					}

					if currentChannelIdCount == 0 {
						logger.Info("close empty voice channel server")
						ss.Close()
						delete(serverStatuses, event.GuildID)
					}
				}

				if len(guildMemberJoinVCs) == 0 {
					delete(memberJoinVCs, event.GuildID)
				} else {
					memberJoinVCs[event.GuildID] = guildMemberJoinVCs
				}
			}
		}
	}()

	sess.Open()
	return sc, nil
}

func (sc *ServiceController) IsMentioned(mc discordgo.MessageCreate) bool {
	for _, mentioned := range mc.Mentions {
		if mentioned.ID == sc.app.ID {
			return true
		}
	}
	return false
}

func (sc *ServiceController) Close() error {
	wg := sync.WaitGroup{}
	wg.Add(1)
	sc.quit <- &wg
	wg.Wait()
	return nil
}
