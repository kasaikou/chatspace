package chatspace

import (
	"fmt"
	"io"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/streamwest-1629/chatspace/app/chatspace/lib"
	"github.com/streamwest-1629/chatspace/service/discord"
	"github.com/streamwest-1629/chatspace/util"
	"go.uber.org/zap"
)

type app struct {
	lock             sync.Mutex
	closed           bool
	closer           chan<- *sync.WaitGroup
	logger           *zap.Logger
	scheduler        *lib.Scheduler
	launchedStates   map[string]*launchState
	voiceChannelName string
}

func New(baseLogger *zap.Logger, discordToken, voiceChannelName string) (io.Closer, error) {

	sess, err := discord.New(discordToken)
	if err != nil {
		return nil, err
	}
	messageCreate, messageCreateCloser, err := discord.ListenMessageCreate(sess, true)
	if err != nil {
		return nil, err
	}
	voiceStateUpdate, voiceStateUpdateCloser, err := discord.ListenVoiceStateUpdate(sess, true)
	if err != nil {
		return nil, err
	}

	closer := make(chan *sync.WaitGroup, 1)

	app := &app{
		closed:         false,
		closer:         make(chan<- *sync.WaitGroup, 1),
		logger:         baseLogger.With(zap.String("app", "chatspace")),
		scheduler:      lib.NewScheduler(),
		launchedStates: make(map[string]*launchState),
	}

	go func() {
		logger := app.logger.With(zap.String("service", "discord-event-listener"))
		for {
			select {
			case wg := <-closer:
				logger.Debug("try to guracefully shutdown")
				defer wg.Done()
				sess.Close()
				messageCreateCloser.Close()
				voiceStateUpdateCloser.Close()
				return

			case event := <-messageCreate:
				logger.Debug("listen messageCreate event")
				app.messageCreate(event)

			case event := <-voiceStateUpdate:
				logger.Debug("listen voiceStateUpdate event")
				app.voiceStateUpdate(event)
			}
		}
	}()

	if err := sess.Open(); err != nil {
		return nil, fmt.Errorf("cannot open discord session: %w", err)
	}
	return app, nil
}

func (a *app) Close() error {
	a.logger.Info("close application")
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.closed {
		return fmt.Errorf("application %w", util.ErrAlreadyClosed)
	}

	a.scheduler.Close()
	wg := sync.WaitGroup{}
	wg.Add(1)
	a.closer <- &wg
	wg.Wait()

	a.closed = true
	a.logger.Info("gracefully close application")
	return nil
}

func (a *app) messageCreate(event discord.Event[discordgo.MessageCreate]) {
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.closed {
		a.logger.Warn("event recieved after application closed")
		return
	}

	// FUTURE FEATURE: Read aloud in joined voice channel
}

func (a *app) voiceStateUpdate(event discord.Event[discordgo.VoiceStateUpdate]) {
	a.lock.Lock()
	defer a.lock.Unlock()

	logger := a.logger.With(zap.String("GuildID", event.Content.GuildID))
	if a.closed {
		logger.Warn("event recieved after application closed")
		return
	}

	sess := event.Content.Session
	currentChannelID, prevChannelID := event.Event.ChannelID, event.Event.BeforeUpdate.ChannelID
	if currentChannelID == prevChannelID {
		// If currentChannelID and prevChannelID are equal, it maybe mean change mute.
		logger.Debug("currentChannelID and prevChannelID are equal", zap.String("ChannelID", currentChannelID))
		return
	}

	move := 0
	var channel *discordgo.Channel

	if currentChannelID != "" {
		currentChannel, err := sess.Channel(currentChannelID)
		if err != nil {
			logger.Error("cannot get discord channel information", zap.Error(err), zap.String("ChannelID", currentChannelID))
		} else if currentChannel.Name == a.voiceChannelName {
			channel = currentChannel
			move++
		}
	}

	if prevChannelID != "" {
		prevChannel, err := sess.Channel(prevChannelID)
		if err != nil {
			logger.Error("cannot get dicord channel information", zap.Error(err), zap.String("ChannelID", prevChannelID))
		} else if prevChannel.Name == a.voiceChannelName {
			channel = prevChannel
			move--
		}
	}

	switch move {
	case 1:
		launchedStatus, exist := a.launchedStates[event.Content.GuildID]
		if !exist {
			logger.Debug("detected to join closed workspace voice channel")
			// TODO: create workspace
			launchedStatus, err := newLaunchState(sess, a.logger, event.Content.GuildID, channel, channel)
			if err != nil {
				logger.Error("launch workspace voice channel", zap.Error(err))
				return
			}
			a.launchedStates[event.Content.GuildID] = launchedStatus
		}
		// TODO: increase join voice member
		launchedStatus.joinVoiceChannel(sess, event.Content.Actor)

	case -1:
		launchedStatus, exist := a.launchedStates[event.Content.GuildID]
		if !exist {
			logger.Warn("detected to leave workspace voice channel, but stopped voice channel")
			return
		} else {
			launchedStatus.leaveVoiceChannel(sess, event.Content.Actor)
		}
	}
}
