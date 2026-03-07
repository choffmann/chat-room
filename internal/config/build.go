package config

import (
	"runtime/debug"
)

var (
	Version       string = "unknown"
	GitCommit     string = "unknown"
	GitRepository string = "unknown"
	BuildTime     string = "unknown"
)

func init() {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	vcsSettings := make(map[string]string)
	for _, s := range bi.Settings {
		vcsSettings[s.Key] = s.Value
	}

	if Version == "unknown" && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		Version = bi.Main.Version
	}
	if GitCommit == "unknown" {
		if rev, ok := vcsSettings["vcs.revision"]; ok {
			GitCommit = rev
		}
	}
	if BuildTime == "unknown" {
		if vcsTime, ok := vcsSettings["vcs.time"]; ok {
			BuildTime = vcsTime
		}
	}
}
