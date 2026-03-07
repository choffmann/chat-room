package main

import (
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/choffmann/chat-room/internal/chat"
	"github.com/choffmann/chat-room/internal/config"
	"github.com/choffmann/chat-room/internal/handler"
	"github.com/choffmann/chat-room/internal/user"
	"github.com/gorilla/mux"
)

func main() {
	logger := config.NewLogger()

	hub := chat.NewHub(logger)
	userRegistry := user.NewRegistry(logger)

	h := handler.New(hub, userRegistry, logger)

	r := mux.NewRouter()
	h.RegisterRoutes(r, config.LegacyRoutes())

	httpHandler := handler.CORSMiddleware(r)

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      httpHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logger.Info("server listening", "addr", srv.Addr)

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server stopped unexpectedly", "error", err)
		os.Exit(1)
	}
}
