package main

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/streamwest-1629/chatspace/app/chatspace"
	"go.uber.org/zap"
)

var (
	logger *zap.Logger
)

func init() {
	logger, _ = zap.NewDevelopment()
}

func main() {

	// healthcheck
	logger.Debug("initialize application")
	http.HandleFunc("/_hck", healthcheck)

	// chatspace application
	discordToken := os.Getenv("CHATSPACE_DISCORD_TOKEN")
	controller, err := chatspace.NewService(logger, discordToken)
	if err != nil {
		logger.Error("cannot start chatspace application", zap.String("discordToken", discordToken[:4]+"***"+discordToken[len(discordToken)-4:]), zap.Error(err))
	}
	defer controller.Close()

	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Fatal("listen server is ended", zap.Error(err))
	}
}

func healthcheck(rw http.ResponseWriter, req *http.Request) {
	logger.Debug("call healthcheck")

	current := time.Now().UTC().Format(time.RFC3339)

	b, _ := json.Marshal(map[string]string{
		"currentTime": current,
	})

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	rw.Write(b)
}
