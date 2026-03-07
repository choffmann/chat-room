package handler

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/choffmann/chat-room/internal/chat"
	"github.com/choffmann/chat-room/internal/user"
	"github.com/gorilla/mux"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func setupHandler(t *testing.T) *Handler {
	t.Helper()
	logger := testLogger()
	h := chat.NewHub(logger)
	ur := user.NewRegistry(logger)
	return New(h, ur, logger)
}

func TestRegisterRoutes_VersionedPrefix(t *testing.T) {
	h := setupHandler(t)
	r := mux.NewRouter()
	h.RegisterRoutes(r, false)

	paths := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/rooms"},
		{"POST", "/api/v1/rooms"},
		{"GET", "/api/v1/users"},
		{"GET", "/api/v1/info"},
		{"GET", "/api/v1/healthz"},
	}

	for _, p := range paths {
		req := httptest.NewRequest(p.method, p.path, nil)
		match := mux.RouteMatch{}
		if !r.Match(req, &match) {
			t.Errorf("expected %s %s to match a route", p.method, p.path)
		}
	}

	// Legacy routes should not be registered
	req := httptest.NewRequest("GET", "/rooms", nil)
	match := mux.RouteMatch{}
	if r.Match(req, &match) {
		t.Error("expected /rooms to not match when legacy routes are disabled")
	}
}

func TestRegisterRoutes_LegacyEnabled(t *testing.T) {
	h := setupHandler(t)
	r := mux.NewRouter()
	h.RegisterRoutes(r, true)

	for _, path := range []string{"/rooms", "/api/v1/rooms"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		match := mux.RouteMatch{}
		if !r.Match(req, &match) {
			t.Errorf("expected GET %s to match a route", path)
		}
	}
}

func TestWebSocketUpgrader(t *testing.T) {
	h := setupHandler(t)

	if h.upgrader.CheckOrigin == nil {
		t.Error("CheckOrigin should be set")
	}

	if h.upgrader.ReadBufferSize == 0 {
		t.Error("ReadBufferSize should be set")
	}

	if h.upgrader.WriteBufferSize == 0 {
		t.Error("WriteBufferSize should be set")
	}
}
