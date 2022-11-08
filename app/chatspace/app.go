package chatspace

import (
	"errors"
	"fmt"
	"sync"

	"github.com/streamwest-1629/chatspace/service/discord"
	"go.uber.org/zap"
)

var ErrAlreadyClosed = errors.New("application have been already closed")

type AppController struct {
	closed bool
	closer chan<- *sync.WaitGroup
}

func New(baseLogger *zap.Logger, discordToken, voiceChannelName string) (*AppController, error) {

	sess, err := discord.New(discordToken)
	if err != nil {
		return nil, err
	}
	messageCreate, err := discord.ListenMessageCreate(sess, true)
	if err != nil {
		return nil, err
	}
	voiceStateUpdate, err := discord.ListenVoiceStateUpdate(sess, true)
	if err != nil {
		return nil, err
	}

	closer := make(chan *sync.WaitGroup, 1)
	controller := &AppController{
		closed: false,
		closer: closer,
	}

	go func() {
		logger := baseLogger.With(zap.String("app", "chatspace"))

		for {
			select {
			case wg := <-closer:
				defer wg.Done()
				controller.closed = true
				sess.Close()
				return

			case message := <-messageCreate:
				sess := message.Content.Session

			case state := <-voiceStateUpdate:
				sess := state.Content.Session
				ch, err := sess.Channel(state.Event.ChannelID)

				if err != nil {
					logger.Error("cannot get channel", zap.Error(err))
					break
				} else if ch.Name != voiceChannelName {
					break
				}
			}
		}
	}()

	if err := sess.Open(); err != nil {
		return nil, fmt.Errorf("cannot open discord session: %w", err)
	}
	return controller, nil
}
