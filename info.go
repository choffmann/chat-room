package main

import (
	"encoding/json"
	"net/http"
	"runtime/debug"
	"time"
)

type Info struct {
	Version       string    `json:"version"`
	GitCommit     string    `json:"commit"`
	GitRepository string    `json:"repository"`
	BuildTime     time.Time `json:"build_time"`
	GoVersion     string    `json:"go_version"`
}

func init() {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	vcsSettings := make(map[string]string)
	for _, s := range bi.Settings {
		vcsSettings[s.Key] = s.Value
	}

	if version == "unknown" && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		version = bi.Main.Version
	}
	if gitCommit == "unknown" {
		if rev, ok := vcsSettings["vcs.revision"]; ok {
			gitCommit = rev
		}
	}
	if buildTime == "unknown" {
		if vcsTime, ok := vcsSettings["vcs.time"]; ok {
			buildTime = vcsTime
		}
	}
}

// GET /healthz
func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

// GET /info
func getInfoHandler(w http.ResponseWriter, r *http.Request) {
	bi, _ := debug.ReadBuildInfo()

	var goVersion string
	if bi != nil {
		goVersion = bi.GoVersion
	}

	var bt time.Time
	if buildTime == "" || buildTime == "unknown" {
		bt = time.Now()
	} else {
		var err error
		bt, err = time.Parse(time.RFC3339, buildTime)
		if err != nil {
			bt, err = time.Parse("2006-01-02T15:04:05Z", buildTime)
			if err != nil {
				http.Error(w, "can't parse build time", http.StatusInternalServerError)
				return
			}
		}
	}

	info := Info{
		Version:       version,
		GitCommit:     gitCommit,
		GitRepository: gitRepository,
		BuildTime:     bt,
		GoVersion:     goVersion,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}
