package config

import (
	"os"
	"strings"
)

func BaseURL() string {
	return strings.TrimSpace(os.Getenv("BASE_URL"))
}

func LegacyRoutes() bool {
	v := strings.TrimSpace(os.Getenv("LEGACY_ROUTES"))
	if v == "" {
		return true
	}
	return v == "true" || v == "1"
}
