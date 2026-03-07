package handler

import (
	"encoding/json"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/choffmann/chat-room/internal/config"
)

type Info struct {
	Version       string    `json:"version" example:"v1.0.0"`
	GitCommit     string    `json:"commit" example:"a1b2c3d"`
	GitRepository string    `json:"repository" example:"https://github.com/choffmann/chat-room"`
	BuildTime     time.Time `json:"build_time" example:"2024-04-09T12:45:00Z"`
	GoVersion     string    `json:"go_version" example:"go1.25.0"`
} // @name BuildInfo

// healthzHandler godoc
// @Summary      Health check
// @Description  Simple liveness probe. Returns plain text "OK".
// @Tags         info
// @Produce      plain
// @Success      200  {string}  string  "OK"
// @Router       /healthz [get]
func (h *Handler) healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

// getInfoHandler godoc
// @Summary      Get build info
// @Description  Exposes metadata about the running binary. Field values are populated at build time; when unavailable, they default to "unknown".
// @Tags         info
// @Produce      json
// @Success      200  {object}  Info
// @Router       /info [get]
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
