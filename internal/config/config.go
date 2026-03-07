package config

import (
	"os"
	"strings"
)

func LegacyRoutes() bool {
	v := strings.TrimSpace(os.Getenv("LEGACY_ROUTES"))
	if v == "" {
		return true
	}
	return v == "true" || v == "1"
}
