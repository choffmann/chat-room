package main

import (
	"encoding/json"
	"net/http"
	"time"
)

type Info struct {
	Version       string    `json:"version"`
	GitCommit     string    `json:"commit"`
	GitBranch     string    `json:"branch"`
	GitRepository string    `json:"repository"`
	BuildTime     time.Time `json:"build_time"`
}

// GET /healthz
func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

// GET /info
func getInfoHandler(w http.ResponseWriter, r *http.Request) {
	var bt time.Time
	if buildTime == "" || buildTime == "unkown" || buildTime == "now" {
		bt = time.Now()
	} else {
		var err error
		bt, err = time.Parse("2006-01-02T15:04:05Z", buildTime)
		if err != nil {
			http.Error(w, "can't parse build time", http.StatusInternalServerError)
		}
	}

	info := Info{
		Version:       version,
		GitCommit:     gitCommit,
		GitBranch:     gitBranch,
		GitRepository: gitRepository,
		BuildTime:     bt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}
