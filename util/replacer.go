package util

import (
	"regexp"

	"github.com/bwmarrin/discordgo"
)

func WordSpliter(input string) []string {
	splitWords := map[string]struct{}{
		"。":  {},
		"、":  {},
		"～":  {},
		"，":  {},
		"．":  {},
		",":  {},
		"\n": {},
		" ":  {},
		"　":  {},
		"「":  {},
		"」":  {},
		"『":  {},
		"』":  {},
		"\"": {},
		"・":  {},
	}

	inputRune := []rune(input)
	results := []string{}
	result := ""

	for i := 0; i < len(inputRune); i++ {
		if _, exist := splitWords[string(inputRune[i])]; exist {
			if result != "" {
				results = append(results, result)
				result = ""
			}
		} else {
			result = result + string(inputRune[i])
		}
	}

	if result != "" {
		results = append(results, result)
	}

	return results
}

func ReplaceMsgFunc(sess *discordgo.Session) func(string) string {

	reg := regexp.MustCompile(`https?://[\w!?/+\-_~;.,*&@#$%()'[\]]+`)

	return func(input string) string {
		return reg.ReplaceAllString(input, "URL")
	}
}
