package chatspace

import (
	"os"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/streamwest-1629/chatspace/app/chatspace/lib"
	"go.uber.org/zap"
)

type status int

const (
	statusClosed status = iota
	statusWork
	statusChat
)

var (
	TimeDurationWork time.Duration = 45
	TimeDurationChat time.Duration = 15
)

func init() {
	if _, exist := os.LookupEnv("DEBUG"); exist {
		TimeDurationWork *= time.Second
		TimeDurationChat *= time.Second
	} else {
		TimeDurationWork *= time.Minute
		TimeDurationChat *= time.Minute
	}
}

type launchState struct {
	lock           sync.Mutex
	logger         *zap.Logger
	status         status
	guildID        string
	voiceChannelID string
	textChannelID  string
	vcConn         *discordgo.VoiceConnection
	workMemberIDs  map[string]interface{}
	chatChannelIDs map[string]interface{}
}

func newLaunchState(sess *discordgo.Session, scheduler *lib.Scheduler, baseLogger *zap.Logger, guildID string, voiceChannel, textChannel *discordgo.Channel) (*launchState, error) {
	logger := baseLogger.With(zap.String("GuildID", guildID))

	vcConn, err := sess.ChannelVoiceJoin(guildID, voiceChannel.ID, false, true)
	if err != nil {
		return nil, err
	}

	return &launchState{
		logger:         logger,
		status:         statusChat,
		guildID:        guildID,
		vcConn:         vcConn,
		voiceChannelID: voiceChannel.ID,
		textChannelID:  textChannel.ID,
		workMemberIDs:  make(map[string]interface{}),
		chatChannelIDs: make(map[string]interface{}),
	}, nil
}

func (state *launchState) joinVoiceChannel(sess *discordgo.Session, actor *discordgo.User) {
	state.lock.Lock()
	defer state.lock.Unlock()

	state.workMemberIDs[actor.ID] = nil

	switch state.status {
	case statusClosed:
		state.logger.Warn("recieve event after workspace closed")
		return
	case statusChat:
	case statusWork:
	}
}

func (state *launchState) leaveVoiceChannel(sess *discordgo.Session, actor *discordgo.User) {
	state.lock.Lock()
	defer state.lock.Unlock()

	delete(state.workMemberIDs, actor.ID)

	switch state.status {
	case statusClosed:
		state.logger.Warn("recieve event after workspace closed")
		return
	case statusChat:
	case statusWork:
	}
}

func (state *launchState) Work2Chat(sess *discordgo.Session, scheduler *lib.Scheduler) {
	state.lock.Lock()
	defer state.lock.Unlock()
	state.logger.Debug("change from work-mode to chat-mode")
	if state.status == statusClosed {
		return
	}

	// TODO: create chatting voice channels and move member

	// TODO: set schedule for changing chat-mode to work-mode
}

func (state *launchState) Chat2Work(sess *discordgo.Session, scheduler *lib.Scheduler) {
	state.lock.Lock()
	defer state.lock.Unlock()
	if state.status == statusClosed {
		return
	}

	state.logger.Debug("change from chat-mode to work-mode")

	// TODO: correct from chatting voice channels' member and delete them.

	// TODO: set schedule for changing work-mode to chat-mode
}

func (state *launchState) Close() error {
	state.lock.Lock()
	defer state.lock.Unlock()

	state.vcConn.Disconnect()

	// TODO: delete chatting voice channels

	state.status = statusClosed
	return nil
}
