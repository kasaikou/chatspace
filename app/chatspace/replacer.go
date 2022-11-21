package chatspace

import (
	"regexp"

	"github.com/bwmarrin/discordgo"
)

func replaceMsgFunc(sess *discordgo.Session) func(string) string {

	reg := regexp.MustCompile(`https?://[\w!?/+\-_~;.,*&@#$%()'[\]]+`)

	return func(input string) string {
		return reg.ReplaceAllString(input, "URL")
	}
}
