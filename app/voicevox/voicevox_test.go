package voicevox_test

import (
	"io"
	"os"
	"testing"

	"github.com/rs/zerolog/log"
	"github.com/streamwest-1629/permanent-apps/apps/voicevox"
	"github.com/streamwest-1629/permanent-apps/util"
)

func TestVoiceVox(t *testing.T) {

	logger := util.NewDevelopment("voicevox-test", false)
	config := voicevox.InitConfig{
		NumThreads:    2,
		LoadAllModels: true,
	}
	vv, err := voicevox.Start(logger, os.Getenv("VOICEVOX_COREPATH"), os.Getenv("VOICEVOX_JTALKDIR"), config)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed open voicevox client")
	}
	defer vv.Quit()

	if _, err := vv.GetSpeakers("", false); err == voicevox.ErrRestarting {
		logger.Info().Err(err).Msg("expected error: restarting")
	} else if err == nil {
		logger.Fatal().Msg("succeed? (expected restarting error)")
	} else {
		logger.Fatal().Err(err).Msg("failed get speakers: unexpected error")
	}

	speakerName := "春日部つむぎ"
	speakers, err := vv.GetSpeakers(speakerName, true)
	if err != nil {
		logger.Fatal().Err(err).Msg("cannot get speakers")
	} else if len(speakers) == 0 {
		logger.Fatal().Err(err).Msgf("cannot found speaker: %s", speakerName)
	}

	logger.Info().Str("speakerName", speakers[0].Name).Int("id", speakers[0].Id).Msg("speaker information")
	rawWav, err := vv.GenerateVoice("おはようございます，先輩", speakers[0].Id, false)
	if err != nil {
		log.Fatal().Err(err).Msg("failed generate voice")
	}
	logger.Info().Msg("generated voice")
	defer rawWav.Close()

	file, err := os.Create("new-world.wav")
	if err != nil {
		logger.Fatal().Err(err).Msg("failed save file")
	}
	defer file.Close()

	if _, err := io.Copy(file, rawWav); err != nil {
		logger.Fatal().Err(err).Msg("failed save file")
	}

}
