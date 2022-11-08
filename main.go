package main

import (
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"
)

var (
	logger *zap.Logger
)

func init() {
	logger, _ = zap.NewDevelopment()
}

func main() {
	logger.Debug("initialize application")
	http.HandleFunc("/_hck", healthcheck)
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
