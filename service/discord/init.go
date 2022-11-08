package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type ManagedEvent interface {
	discordgo.MessageCreate |
		discordgo.VoiceStateUpdate
}

type EventContent struct {
	GuildID string
	Actor   *discordgo.User
	Session *discordgo.Session
}

type Event[EventType ManagedEvent] struct {
	Content EventContent
	Event   EventType
}

func New(discordToken string) (*discordgo.Session, error) {
	return discordgo.New(fmt.Sprintf("Bot %s", discordToken))
}

func appOwnID(sess *discordgo.Session) (string, error) {
	if app, err := sess.Application("@me"); err != nil {
		return "", err
	} else {
		return app.ID, nil
	}
}

// Set discord message create event listener.
// Listened event will be send to channel, it is return value.
//
// If ignoreSelf is true, listener will be skip to send event created by application's own acts.
func ListenMessageCreate(sess *discordgo.Session, ignoreSelf bool) (<-chan Event[discordgo.MessageCreate], error) {

	appID, err := appOwnID(sess)
	if err != nil {
		return nil, err
	}
	listener := make(chan Event[discordgo.MessageCreate])

	sess.AddHandler(func(sess *discordgo.Session, content *discordgo.MessageCreate) {
		if content.Author.ID == appID && ignoreSelf {
			return
		}
		listener <- Event[discordgo.MessageCreate]{
			Content: EventContent{
				Actor:   content.Author,
				GuildID: content.GuildID,
				Session: sess,
			},
			Event: *content,
		}
	})

	return listener, nil
}

// Set discord message create event listener.
// Listened event will be send to channel, it is return value.
//
// If ignoreSelf is true, listener will be skip to send event created by application's own acts.
func ListenVoiceStateUpdate(sess *discordgo.Session, ignoreSelf bool) (<-chan Event[discordgo.VoiceStateUpdate], error) {

	appID, err := appOwnID(sess)
	if err != nil {
		return nil, err
	}
	listener := make(chan Event[discordgo.VoiceStateUpdate])

	sess.AddHandler(func(sess *discordgo.Session, content *discordgo.VoiceStateUpdate) {
		if content.Member.User.ID == appID && ignoreSelf {
			return
		}
		listener <- Event[discordgo.VoiceStateUpdate]{
			Content: EventContent{
				Actor:   content.Member.User,
				GuildID: content.GuildID,
				Session: sess,
			},
			Event: *content,
		}
	})
}
