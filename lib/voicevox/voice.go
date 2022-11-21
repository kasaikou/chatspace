package voicevox

import "github.com/google/uuid"

type VoiceSpeaker struct {
	Name   string       `json:"name"`
	UUID   uuid.UUID    `json:"speaker_uuid"`
	Styles []VoiceStyle `json:"styles"`
}

type VoiceStyle struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}
