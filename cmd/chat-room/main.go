//go:generate go tool swag init -d ../../cmd/chat-room,../../internal/handler -g main.go -o ../../docs

package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/choffmann/chat-room/docs"
	"github.com/choffmann/chat-room/internal/chat"
	"github.com/choffmann/chat-room/internal/config"
	"github.com/choffmann/chat-room/internal/handler"
	"github.com/choffmann/chat-room/internal/upload"
	"github.com/choffmann/chat-room/internal/user"
	"github.com/gorilla/mux"
)

// @title           Chat Room API
// @version         1.0
// @description     A lightweight real-time chat backend written in Go. The server manages ephemeral chat rooms where clients connect via WebSocket to exchange messages. Rooms are created on-demand and automatically deleted after 3 hours of inactivity.
// @description
// @description     ## Additional Info
// @description     Many resources (rooms, users, messages) carry an `additionalInfo` field. This is a free-form JSON object that the server stores as-is — it has no predefined schema and is never validated or interpreted by the backend.
// @description
// @description     Use it to attach arbitrary metadata to any resource, for example:
// @description     - **Rooms:** theme, description, language, or feature flags for your UI.
// @description     - **Users:** avatar URL, bio, status text, or client-specific preferences.
// @description     - **Messages:** formatting hints, link previews, or custom reaction data.
// @description
// @description     The field is always optional. If omitted it defaults to an empty object (`{}`). On `PATCH` requests the provided keys are merged into the existing object; on `PUT` the entire object is replaced.
// @host            localhost:8080
// @BasePath        /api/v1
func main() {
	if baseURL := config.BaseURL(); baseURL != "" {
		docs.SwaggerInfo.Host = baseURL
	}

	logger := config.NewLogger()

	uploadStore := upload.NewStore(config.UploadDir(), logger)

	hub := chat.NewHub(logger)
	hub.SetOnRoomDelete(func(roomID uint) {
		if err := uploadStore.DeleteRoomDir(roomID); err != nil {
			logger.Warn("failed to delete room upload dir", "roomID", roomID, "error", err)
		}
	})

	userRegistry := user.NewRegistry(logger)

	h := handler.New(hub, userRegistry, logger, uploadStore)

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

	go func() {
		logger.Info("server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.Info("shutdown signal received", "signal", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("http server shutdown error", "error", err)
	}

	hub.ShutdownAll()

	if err := uploadStore.DeleteAll(); err != nil {
		logger.Warn("failed to clean up upload directory", "error", err)
	}

	logger.Info("server stopped")
}
