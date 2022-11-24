package main

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/streamwest-1629/chatspace/app/chatspace"
	"github.com/streamwest-1629/chatspace/app/voicevox"
	"github.com/streamwest-1629/chatspace/lib/s3"
	"github.com/streamwest-1629/chatspace/util"
	"go.uber.org/zap"
)

var (
	logger *zap.Logger
)

func init() {
	if val, exist := os.LookupEnv("DEBUG"); !exist || (exist && (val == "0" || val == "false" || val == "False" || val == "FALSE")) {
		// production mode
		logger, _ = zap.NewProduction()
		logger.Info("initializing as production mode")

		creds, err := s3.LoadEnvCredentials()
		if err != nil {
			logger.Error(err.Error())
		} else {
			for k, v := range creds {
				os.Setenv(k, v)
			}
		}

		chatspace.SetTimeStep(time.Minute)

	} else {
		// development mode
		logger, _ = zap.NewDevelopment()
		logger.Info("initializing as debug mode")

		chatspace.SetTimeStep(4 * time.Second)
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
	discordToken := os.Getenv("CHATSPACE_DISCORD_TOKEN")
	controller, err := chatspace.NewService(logger, discordToken, vv)
	if err != nil {
		logger.Error("cannot start chatspace application", zap.String("discordToken", discordToken[:8]+"***"+discordToken[len(discordToken)-8:]), zap.Error(err))
	}
	defer controller.Close()

	if err := http.ListenAndServe(":80", nil); err != nil {
		logger.Fatal("listen server is ended", zap.Error(err))
	}
}

func healthcheck(rw http.ResponseWriter, req *http.Request) {
	logger.Debug("call healthcheck")

	// current time
	current := time.Now().UTC().Format(time.RFC3339)

	var resources interface{}
	resources, err := util.Utilization()
	if err != nil {
		resources = err.Error()
	}

	// format json
	b, _ := json.Marshal(map[string]interface{}{
		"currentTime": current,
		"resources":   resources,
	})

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	rw.Write(b)
}
