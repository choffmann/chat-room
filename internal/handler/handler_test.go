package handler

import (
	"io"
	"log/slog"
	"testing"

	"github.com/choffmann/chat-room/internal/chat"
	"github.com/choffmann/chat-room/internal/user"
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
