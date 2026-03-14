package config

import (
	"os"
	"strings"
)

func BaseURL() string {
	return strings.TrimSpace(os.Getenv("BASE_URL"))
}

func UploadDir() string {
	v := strings.TrimSpace(os.Getenv("UPLOAD_DIR"))
	if v == "" {
		return "./uploads"
	}
	return v
}

func LegacyRoutes() bool {
	v := strings.TrimSpace(os.Getenv("LEGACY_ROUTES"))
	if v == "" {
		return true
	}
	return v == "true" || v == "1"
}
