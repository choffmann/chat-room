package handler

import (
	"encoding/json"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/choffmann/chat-room/internal/config"
)

type Info struct {
	Version       string    `json:"version"`
	GitCommit     string    `json:"commit"`
	GitRepository string    `json:"repository"`
	BuildTime     time.Time `json:"build_time"`
	GoVersion     string    `json:"go_version"`
}

// GET /healthz
func (h *Handler) healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

// GET /info
func (h *Handler) getInfoHandler(w http.ResponseWriter, r *http.Request) {
	bi, _ := debug.ReadBuildInfo()

	var goVersion string
	if bi != nil {
		goVersion = bi.GoVersion
	}

	var bt time.Time
	if config.BuildTime == "" || config.BuildTime == "unknown" {
		bt = time.Now()
	} else {
		var err error
		bt, err = time.Parse(time.RFC3339, config.BuildTime)
		if err != nil {
			bt, err = time.Parse("2006-01-02T15:04:05Z", config.BuildTime)
			if err != nil {
				http.Error(w, "can't parse build time", http.StatusInternalServerError)
				return
			}
		}
	}

	info := Info{
		Version:       config.Version,
		GitCommit:     config.GitCommit,
		GitRepository: config.GitRepository,
		BuildTime:     bt,
		GoVersion:     goVersion,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}
