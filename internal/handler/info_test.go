package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/choffmann/chat-room/internal/config"
)

func TestHealthzHandler(t *testing.T) {
	h := setupHandler(t)

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()

	h.healthzHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()
	if body != "OK" {
		t.Errorf("expected body 'OK', got '%s'", body)
	}
}

func TestGetInfoHandler(t *testing.T) {
	h := setupHandler(t)

	config.Version = "v1.0.0"
	config.GitCommit = "abc123"
	config.BuildTime = "2025-01-01T00:00:00Z"
	config.GitRepository = "https://github.com/choffmann/chat-room"

	req := httptest.NewRequest("GET", "/info", nil)
	w := httptest.NewRecorder()

	h.getInfoHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var info Info
	if err := json.NewDecoder(w.Body).Decode(&info); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if info.Version != config.Version {
		t.Errorf("expected version %s, got %s", config.Version, info.Version)
	}

	if info.GitCommit != config.GitCommit {
		t.Errorf("expected gitCommit %s, got %s", config.GitCommit, info.GitCommit)
	}

	if info.BuildTime.IsZero() {
		t.Error("expected buildTime to be set")
	}

	if info.GitRepository != config.GitRepository {
		t.Errorf("expected gitRepository %s, got %s", config.GitRepository, info.GitRepository)
	}
}
