package voicevox

import "math/rand"

type CharacterExpressions struct {
	hello       []string
	seeYou      []string
	cannotRead  []string
	shutdown    []string
	callMeLater []string
}

func CharacterExpression(characterName string) *CharacterExpressions {

	config := CharacterExpressions{
		hello:       []string{"おはようございます", "こんにちは", "よろしくお願いします"},
		seeYou:      []string{"お疲れ様です", "また会いましょう"},
		cannotRead:  []string{"うまく読めません", "なんて読むんですか？"},
		shutdown:    []string{"ごめんなさい、一度アプリを落とすように言われました"},
		callMeLater: []string{"また呼んでください", "また気軽に声をかけてください"},
	}
	characterConfigs := map[string]CharacterExpressions{
		"ずんだもん": {
			hello:       []string{"おはようなのだ", "よろしくなのだ"},
			seeYou:      []string{"お疲れ様なのだ", "また会おうなのだ"},
			cannotRead:  []string{"うまく読めないのだ", "なんて読むのだ？"},
			shutdown:    []string{"ごめんなのだ、一度アプリを落とさなきゃいけないのだ"},
			callMeLater: []string{"また呼んでほしいのだ", "また気軽に声をかけてほしいのだ"},
		},
	}

	characterConfig, exist := characterConfigs[characterName]
	if !exist {
		return &config
	}

	if len(characterConfig.hello) > 0 {
		config.hello = characterConfig.hello
	}
	if len(characterConfig.seeYou) > 0 {
		config.seeYou = characterConfig.seeYou
	}
	if len(characterConfig.cannotRead) > 0 {
		config.cannotRead = characterConfig.cannotRead
	}
	if len(characterConfig.shutdown) > 0 {
		config.shutdown = characterConfig.shutdown
	}
	if len(characterConfig.callMeLater) > 0 {
		config.callMeLater = characterConfig.callMeLater
	}

	return &config
}

func (c *CharacterExpressions) Hello() string {
	return randStrs(c.hello)
}

func (c *CharacterExpressions) SeeYou() string {
	return randStrs(c.seeYou)
}

func (c *CharacterExpressions) CannotRead() string {
	return randStrs(c.cannotRead)
}

func (c *CharacterExpressions) ShutDown() string {
	return randStrs(c.shutdown)
}

func (c *CharacterExpressions) CallMeLater() string {
	return randStrs(c.callMeLater)
}

func randStrs(opt []string) string {
	return opt[rand.Intn(len(opt))]
}
