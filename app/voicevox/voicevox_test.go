package voicevox_test

import (
	"io"
	"os"
	"testing"

	"github.com/rs/zerolog/log"
	"github.com/streamwest-1629/chatspace/app/voicevox"
	"go.uber.org/zap"
)

func TestVoiceVox(t *testing.T) {

	logger, _ := zap.NewDevelopment()
	config := voicevox.InitConfig{
		NumThreads:    2,
		LoadAllModels: true,
	}
	vv, err := voicevox.Start(logger, os.Getenv("VOICEVOX_COREPATH"), os.Getenv("VOICEVOX_JTALKDIR"), config)
	if err != nil {
		logger.Fatal("failed open voicevox client", zap.Error(err))
	}
	defer vv.Quit()

	if _, err := vv.GetSpeakers("", false); err == voicevox.ErrRestarting {
		logger.Info("expected error: restarting", zap.Error(err))
	} else if err == nil {
		logger.Fatal("succeed? (expected restarting error)")
	} else {
		logger.Fatal("failed get speakers: unexpected error", zap.Error(err))
	}

	speakerName := "春日部つむぎ"
	speakers, err := vv.GetSpeakers(speakerName, true)
	if err != nil {
		logger.Fatal("cannot get speakers", zap.Error(err))
	} else if len(speakers) == 0 {
		logger.Fatal("cannot found speaker", zap.String("speakerName", speakerName), zap.Error(err))
	}

	logger.Info("speaker information", zap.String("speakerName", speakers[0].Name), zap.Int("id", speakers[0].Id))
	rawWav, err := vv.GenerateVoice("おはようございます，先輩", speakers[0].Id, false)
	if err != nil {
		log.Fatal().Err(err).Msg("failed generate voice")
	}
	logger.Info("generated voice")
	defer rawWav.Close()

	file, err := os.Create("new-world.wav")
	if err != nil {
		logger.Fatal("failed save file", zap.Error(err))
	}
	defer file.Close()

	if _, err := io.Copy(file, rawWav); err != nil {
		logger.Fatal("failed save file", zap.Error(err))
	}

}
