package chatspace

import (
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

type serverStatusMode int

const (
	serverStatusModeChat serverStatusMode = iota
	serverStatusModeWork
)

const (
	timeStep  = time.Second
	chatTimes = 15 * timeStep
	workTimes = 45 * timeStep
)

type ServerStatus struct {
	lock              sync.Mutex
	logger            *zap.Logger
	isClosed          bool
	sess              *discordgo.Session
	voiceConn         *discordgo.VoiceConnection
	guildID           string
	channelID         string
	mode              serverStatusMode
	scheduler         *ScheduleQueue
	memberIDs         map[string]struct{}
	managedChannelIDs map[string]struct{}
}

func NewServerStatus(baseLogger *zap.Logger, sess *discordgo.Session, scheduler *ScheduleQueue, guildID, channelID string) (*ServerStatus, error) {

	vc, err := sess.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		return nil, err
	}

	ss := &ServerStatus{
		logger: baseLogger.With(
			zap.String("feature", "serverStatus"),
			zap.Time("launchAt", time.Now().UTC()),
			zap.String("guildID", guildID),
		),
		sess:              sess,
		voiceConn:         vc,
		guildID:           guildID,
		channelID:         channelID,
		scheduler:         scheduler,
		memberIDs:         make(map[string]struct{}),
		managedChannelIDs: make(map[string]struct{}),
	}

	if msg, err := ss.sess.ChannelMessageSendEmbed(ss.channelID, &discordgo.MessageEmbed{
		Title:       "💕よろしくおねがいします！",
		Description: "この度は来てくださりありがとうございます。しっかり作業部屋を運営してまいりますのでよろしくお願いします。",
	}); err != nil {
		ss.logger.Error("failed send message", zap.String("channelID", msg.ChannelID), zap.Error(err))
	}

	ss.Switch2Chat()

	return ss, err
}

func (ss *ServerStatus) onMessageCreate(sess *discordgo.Session, event *discordgo.MessageCreate) {
	ss.lock.Lock()
	defer ss.lock.Unlock()
}

func (ss *ServerStatus) onVoiceChangeUpdate(sess *discordgo.Session, event *discordgo.VoiceStateUpdate) (isClose bool) {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	isClose = false
	userId := event.Member.User.ID
	if event.BeforeUpdate == nil {
		event.BeforeUpdate = &discordgo.VoiceState{}
	}
	if event.ChannelID == event.BeforeUpdate.ChannelID {
		return false
	}

	if event.ChannelID == ss.channelID {
		if _, exist := ss.memberIDs[userId]; !exist {

			ss.logger.Debug("Joined into chatspace", zap.String("userID", userId))
			ss.memberIDs[userId] = struct{}{}

			switch ss.mode {
			case serverStatusModeWork:
				if err := sess.GuildMemberMute(ss.guildID, userId, true); err != nil {
					ss.logger.Error("cannot change mute", zap.String("userID", userId), zap.String("changeTo", "mute"), zap.Error(err))
				}
			case serverStatusModeChat:
				if err := sess.GuildMemberMute(ss.guildID, userId, false); err != nil {
					ss.logger.Error("cannot change mute", zap.String("userID", userId), zap.String("changeTo", "unmute"), zap.Error(err))
				}
			}
		}

	} else if event.BeforeUpdate.ChannelID == ss.channelID {

		if _, exist := ss.memberIDs[userId]; exist {
			ss.logger.Debug("Left from chatspace", zap.String("userID", userId))
			delete(ss.memberIDs, userId)

			if len(ss.memberIDs) == 0 {
				isClose = true
			}
		}
	}

	if event.ChannelID != ss.channelID && event.ChannelID != "" {
		if err := sess.GuildMemberMute(ss.guildID, userId, false); err != nil {
			ss.logger.Error("cannot change mute", zap.String("userID", userId), zap.String("changeTo", "unmute"), zap.Error(err))
		}
	}
	return isClose
}

func (ss *ServerStatus) Switch2Work() {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	if ss.isClosed {
		return
	}

	ss.logger.Info("switch mode chat to work")
	ss.mode = serverStatusModeWork
	for memberId := range ss.memberIDs {
		if err := ss.sess.GuildMemberMute(ss.guildID, memberId, true); err != nil {
			ss.logger.Error("cannot change mute", zap.String("userID", memberId), zap.String("changeTo", "mute"), zap.Error(err))
		}
	}

	if msg, err := ss.sess.ChannelMessageSendEmbed(ss.channelID, &discordgo.MessageEmbed{
		Title:       "🚀作業時間です！",
		Description: "作業は45分間です。作業中はミュートを行うので次の休憩までに作業を進めましょう。",
		Footer: &discordgo.MessageEmbedFooter{
			Text: "休憩時間は" + time.Now().Add(9*time.Hour).Format(time.Kitchen) + "ごろからです",
		},
	}); err != nil {
		ss.logger.Error("failed send message", zap.String("channelID", msg.ChannelID), zap.Error(err))
	}

	ss.scheduler.PushBack(ScheduleEvent{
		Unix: time.Now().Add(workTimes).Unix(),
		Func: ss.Switch2Chat,
	})
}

func (ss *ServerStatus) Switch2Chat() {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	if ss.isClosed {
		return
	}

	ss.logger.Info("switch mode work to chat")
	ss.mode = serverStatusModeChat
	for memberId := range ss.memberIDs {
		if err := ss.sess.GuildMemberMute(ss.guildID, memberId, false); err != nil {
			ss.logger.Error("cannot change mute", zap.String("userID", memberId), zap.String("changeTo", "unmute"), zap.Error(err))
		}
	}

	if msg, err := ss.sess.ChannelMessageSendEmbed(ss.channelID, &discordgo.MessageEmbed{
		Title:       "🌿休憩時間です！",
		Description: "休憩は15分間です。休憩中はミュートを外すので好きに話してください。",
		Footer: &discordgo.MessageEmbedFooter{
			Text: "作業時間は" + time.Now().Add(9*time.Hour).Format(time.Kitchen) + "ごろからです",
		},
	}); err != nil {
		ss.logger.Error("failed send message", zap.String("channelID", msg.ChannelID), zap.Error(err))
	}

	ss.scheduler.PushBack(ScheduleEvent{
		Unix: time.Now().Add(chatTimes).Unix(),
		Func: ss.Switch2Work,
	})
}

func (ss *ServerStatus) Close() error {
	ss.isClosed = true

	if msg, err := ss.sess.ChannelMessageSendEmbed(ss.channelID, &discordgo.MessageEmbed{
		Title:       "🤗またお越しください！",
		Description: "私はすぐに駆け付けます。ボイスチャンネルにまた来てください。",
	}); err != nil {
		ss.logger.Error("failed send message", zap.String("channelID", msg.ChannelID), zap.Error(err))
	}
	return ss.voiceConn.Disconnect()
}
