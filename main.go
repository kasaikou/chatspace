package main

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/streamwest-1629/chatspace/app/chatspace"
	"github.com/streamwest-1629/chatspace/app/voicevox"
	"github.com/streamwest-1629/chatspace/lib/s3"
	"go.uber.org/zap"
)

var (
	logger       *zap.Logger
	environments map[string]string
)

func init() {
	logger, _ = zap.NewDevelopment()
	env, err := s3.LoadEnvCredentials()
	if err != nil {
		logger.Error(err.Error())
		environments = nil
	} else {
		environments = env
	}
}

func main() {

	// healthcheck
	logger.Debug("initialize application")
	http.HandleFunc("/_hck", healthcheck)

	// voicevox application
	config := voicevox.InitConfig{
		NumThreads:    2,
		LoadAllModels: true,
	}

	vv, err := voicevox.Start(logger, os.Getenv("VOICEVOX_COREPATH"), os.Getenv("VOICEVOX_JTALKDIR"), config)
	if err != nil {
		logger.Fatal("cannot start voicevox application", zap.Error(err))
	}
	defer vv.Quit()

	time.Sleep(200 * time.Millisecond)
	if _, err := vv.GetSpeakers("", true); err != nil {
		logger.Fatal("cannot start voicevox application", zap.Error(err))
	}

	// chatspace application
	// discordToken := os.Getenv("CHATSPACE_DISCORD_TOKEN")
	discordToken := environments["CHATSPACE_DISCORD_TOKEN"]
	controller, err := chatspace.NewService(logger, discordToken, vv)
	if err != nil {
		logger.Error("cannot start chatspace application", zap.String("discordToken", discordToken[:8]+"***"+discordToken[len(discordToken)-8:]), zap.Error(err))
	}
	defer controller.Close()

	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Fatal("listen server is ended", zap.Error(err))
	}
}

func healthcheck(rw http.ResponseWriter, req *http.Request) {
	logger.Debug("call healthcheck")

	current := time.Now().UTC().Format(time.RFC3339)

	envKeys := make([]string, 0, len(environments))
	for k := range environments {
		envKeys = append(envKeys, k)
	}

	b, _ := json.Marshal(map[string]interface{}{
		"currentTime": current,
		"environments": map[string]interface{}{
			"count": len(envKeys),
			"keys":  envKeys,
		},
	})

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	rw.Write(b)
}
