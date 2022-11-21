package chatspace

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

// Controll chatspace application service.
// Internal members contains external service sessions.
type ServiceController struct {
	logger   *zap.Logger
	chCloser chan<- *sync.WaitGroup
	discord  *discordgo.Session
}

var ManagedChannelName = "もくもく"

// Make a new ServiceController instance.
func NewService(baseLogger *zap.Logger, discordToken string) (*ServiceController, error) {

	// Initialize discord service
	baseLogger = baseLogger.With(zap.String("package", "chatspace"))
	sess, err := discordgo.New(fmt.Sprintf("Bot %s", discordToken))
	if err != nil {
		return nil, fmt.Errorf("failed create a new discord session: %w", err)
	}

	app, err := sess.Application("@me")
	if err != nil {
		return nil, fmt.Errorf("failed get application status: %w", err)
	}

	messageCreateListener := make(chan discordgo.MessageCreate)
	voiceStateUpdateListener := make(chan discordgo.VoiceStateUpdate)
	chCloser := make(chan *sync.WaitGroup)

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

	// Application Event handler
	go func() {

		logger := baseLogger.With(zap.String("feature", "eventListener"))
		schedules := NewScheduleQueue()
		ticker := time.NewTicker(timeStep / 4)
		serverStatuses := map[string]*ServerStatus{}

		for {

		Select:
			select {
			case wg := <-chCloser:
				defer wg.Done()

				// finalize eventListener
				for _, ss := range serverStatuses {
					if err := ss.Close(); err != nil {
						logger.Error("cannot close chatspace instance", zap.Error(err))
					}
				}

				logger.Info("started finalize application event listener")
				if err := sess.Close(); err != nil {
					logger.Error("failed discord session's closing", zap.Error(err))
				}
				close(messageCreateListener)
				close(voiceStateUpdateListener)
				ticker.Stop()
				return

			case event := <-messageCreateListener:
				logger.Debug("triggered messageCreate event")
				serverStatus, exist := serverStatuses[event.GuildID]
				if exist {
					serverStatus.onMessageCreate(sess, &event)
				}

			case event := <-voiceStateUpdateListener:

				logger.Debug("triggered voiceStateUpdate event")

				_, exist := serverStatuses[event.GuildID]
				if !exist {
					logger.Debug("check join and start chatspace server")
					if event.ChannelID != "" {
						ch, err := sess.Channel(event.ChannelID)
						if err != nil {
							logger.Error("failed get discord channel status", zap.Error(err))
							break Select
						}

						if ch.Name == ManagedChannelName {
							logger.Debug("request new chatspace server instance")
							serverStatus, err := NewServerStatus(baseLogger, sess, schedules, event.GuildID, event.ChannelID)
							if err != nil {
								logger.Error("failed new chatspace server instance", zap.Error(err))
							} else {
								serverStatuses[event.GuildID] = serverStatus
							}
						}
					}
				}

				serverStatus, exist := serverStatuses[event.GuildID]
				if exist {
					if isClose := serverStatus.onVoiceChangeUpdate(sess, &event); isClose {
						if err := serverStatus.Close(); err != nil {
							logger.Error("cannot close chatspace instance", zap.Error(err))
						}
						delete(serverStatuses, event.GuildID)
					}
				}

			case <-ticker.C:

				// Update schedule
				logger.Debug("update event schedule")
				if schedules.Len() > 2 {
					if !sort.IsSorted(schedules) {
						sort.Sort(schedules)
					}
				}

				currentUnix := time.Now().Unix()
				for schedules.Len() > 0 {
					if schedule := schedules.Front(); schedule.Unix < currentUnix {
						schedule.Func()
						schedules.PopFront()
					} else {
						logger.Debug("checked event schedules", zap.Time("nextEvent", time.Unix(schedule.Unix, 0)))
						break
					}
				}
			}
		}
	}()

	sess.Open()
	sc := &ServiceController{
		logger:   baseLogger.With(zap.String("feature", "controller")),
		chCloser: chCloser,
		discord:  sess,
	}

	return sc, nil
}

// Close the chatspace aplication service.
func (sc *ServiceController) Close() error {
	wg := sync.WaitGroup{}
	wg.Add(1)
	sc.chCloser <- &wg
	wg.Wait()
	return nil
}
