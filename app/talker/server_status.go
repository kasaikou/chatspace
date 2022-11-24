package talker

import (
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/streamwest-1629/chatspace/app/voicevox"
	"github.com/streamwest-1629/chatspace/util"
	"go.uber.org/zap"
)

type joinedServerStatus struct {
	lock           sync.Mutex
	logger         *zap.Logger
	sess           *discordgo.Session
	voiceConn      *voicevox.ManagedDiscordVoiceConnection
	guildID        string
	memberIds      map[string]struct{}
	memberSpeakers map[string]voicevox.VoiceSpeaker
	prevChannelID  string
}

func newJoinedServerStatus(baseLogger *zap.Logger, sess *discordgo.Session, voicevoxApp *voicevox.VoiceVox, event *discordgo.MessageCreate, voiceChannelId string) (*joinedServerStatus, error) {

	vc, err := voicevox.StartManagedDiscordVoiceConnection(
		baseLogger.With(zap.String("feature", "voicevoxRequest")),
		sess, event.GuildID, voiceChannelId, voicevoxApp,
		util.ReplaceMsgFunc(sess),
	)

	if err != nil {
		SendMessage(
			sess, baseLogger, event.ID, event.ChannelID,
			strings.Join([]string{"🤯", "ボイスチャットに入ることができませんでした．"}, " "),
			nil,
		)
		return nil, err
	}

	baseLogger = baseLogger.With(
		zap.Time("launchAt", time.Now().UTC()),
		zap.String("guildID", event.GuildID),
	)

	ss := &joinedServerStatus{
		logger:         baseLogger.With(zap.String("feature", "serverStatus")),
		sess:           sess,
		voiceConn:      vc,
		guildID:        event.GuildID,
		prevChannelID:  event.ChannelID,
		memberIds:      make(map[string]struct{}),
		memberSpeakers: make(map[string]voicevox.VoiceSpeaker),
	}

	return ss, nil
}

func (ss *joinedServerStatus) SetVoiceSpeaker(event *discordgo.MessageCreate, memberId string, speaker voicevox.VoiceSpeaker) {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	ss.memberSpeakers[memberId] = speaker

	member, err := ss.sess.GuildMember(ss.guildID, memberId)
	if err != nil {
		ss.logger.Error("failed get discord server member status", zap.Error(err))
		return
	}

	memberName := member.Nick
	if memberName == "" {
		memberName = member.User.Username
	}

	expression := voicevox.CharacterExpression(speaker.Character)
	intro := strings.Join([]string{memberName, expression.Hello()}, "、")

	SendMessage(
		ss.sess, ss.logger, event.ID, event.ChannelID,
		strings.Join([]string{"💕", intro}, " "),
		&discordgo.MessageEmbed{
			Description: "あなたの声はこれから「" + speaker.Name + "」が担当させていただきます。",
		},
	)
}

func (ss *joinedServerStatus) Speak(event *discordgo.MessageCreate) {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	speaker, exist := ss.memberSpeakers[event.Author.ID]
	if !exist {
		speakers, err := ss.voiceConn.GetSpeakers("", true)
		if err != nil {
			ss.logger.Error("cannot get speakers status", zap.Error(err))
			return
		}

		speaker = speakers[rand.Intn(len(speakers))]
		ss.memberSpeakers[event.Author.ID] = speaker
	}

	contents := util.WordSpliter(event.ContentWithMentionsReplaced())
	for _, content := range contents {
		ss.voiceConn.Speak(speaker.Id, false, content)
	}
	ss.prevChannelID = event.ChannelID
}

func (ss *joinedServerStatus) Close() error {
	ss.lock.Lock()
	defer ss.lock.Unlock()
	SendMessage(ss.sess, ss.logger, "", ss.prevChannelID, "🤗 退出します．", nil)
	return ss.voiceConn.Close()
}

func SendMessage(sess *discordgo.Session, logger *zap.Logger, replyMessageID, channelID, mainContent string, embed *discordgo.MessageEmbed) {
	if replyMessageID == "" {
		if embed == nil {
			if _, err := sess.ChannelMessageSend(channelID, mainContent); err != nil {
				logger.Error("failed send message", zap.Error(err), zap.String("channelID", channelID))
			}
		} else {
			if embed.Title == "" {
				embed.Title = mainContent
			} else if embed.Description == "" {
				embed.Description = mainContent
			} else if embed.Footer == nil {
				embed.Footer = &discordgo.MessageEmbedFooter{
					Text: mainContent,
				}
			}

			if _, err := sess.ChannelMessageSendEmbed(channelID, embed); err != nil {
				logger.Error("failed send message", zap.Error(err), zap.String("channelID", channelID))
			}
		}
	} else {
		replyReference := discordgo.MessageReference{MessageID: replyMessageID}

		if embed == nil {
			if _, err := sess.ChannelMessageSendReply(channelID, mainContent, &replyReference); err != nil {
				logger.Error("failed send message", zap.Error(err), zap.String("channelID", channelID), zap.String("replyMsgID", replyMessageID))
			}
		} else {
			if embed.Title == "" {
				embed.Title = mainContent
			} else if embed.Description == "" {
				embed.Description = mainContent
			} else if embed.Footer == nil {
				embed.Footer = &discordgo.MessageEmbedFooter{
					Text: mainContent,
				}
			}

			if _, err := sess.ChannelMessageSendEmbedReply(channelID, embed, &replyReference); err != nil {
				logger.Error("failed send message", zap.Error(err), zap.String("channelID", channelID), zap.String("replyMsgID", replyMessageID))
			}
		}
	}
}
