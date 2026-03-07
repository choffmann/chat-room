package handler

import (
	"log/slog"
	"net/http"

	"github.com/choffmann/chat-room/internal/chat"
	"github.com/choffmann/chat-room/internal/model"
	"github.com/choffmann/chat-room/internal/user"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

type Handler struct {
	hub          *chat.Hub
	userRegistry *user.Registry
	upgrader     websocket.Upgrader
	systemUser   model.User
	defaultNames []string
	logger       *slog.Logger
}

func New(hub *chat.Hub, userRegistry *user.Registry, logger *slog.Logger) *Handler {
	return &Handler{
		hub:          hub,
		userRegistry: userRegistry,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		systemUser: model.User{
			ID:   uuid.New(),
			Name: "system",
		},
		defaultNames: []string{
			"Toni Tester",
			"Harald Hüftschmerz",
			"Andre Android",
			"Hans Hotfix",
			"Peter Push",
			"Rebase Randy",
			"Prof. Prokrastination",
			"Mira Mobil",
			"Lars Launcher",
			"Paul Pixel",
			"Nora Nexus",
			"Timo Touch",
			"Benny Bluetooth",
			"Hanna Hotspot",
			"Pixel Peter",
			"APK Alex",
			"Touchscreen Toni",
			"Kotlin Kevin",
			"Async Andy",
			"Compose Chris",
			"Composable Clara",
			"SideEffect Susi",
			"Gradle Gero",
			"Activity Anni",
			"Manifest Mona",
			"Resource Rhea",
			"ViewModel Viktor",
			"Intent Ingo",
		},
		logger: logger,
	}
}

func (h *Handler) RegisterRoutes(r *mux.Router, legacyRoutes bool) {
	v1 := r.PathPrefix("/api/v1").Subrouter()
	h.registerV1Routes(v1)

	v1.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("/api/v1/swagger/doc.json"),
	))

	if legacyRoutes {
		h.registerV1Routes(r)
	}
}

func (h *Handler) registerV1Routes(r *mux.Router) {
	// Room routes
	r.HandleFunc("/rooms", h.createRoomHandler).Methods("POST")
	r.HandleFunc("/rooms", h.getAllRoomsHandler).Methods("GET")
	r.HandleFunc("/rooms/users", h.getAllUsersInRoomsHandler).Methods("GET")
	r.HandleFunc("/rooms/{roomID}", h.getRoomIDHandler).Methods("GET")
	r.HandleFunc("/rooms/{roomID}", h.patchRoomHandler).Methods("PATCH")
	r.HandleFunc("/rooms/{roomID}", h.putRoomHandler).Methods("PUT")
	r.HandleFunc("/rooms/{roomID}/users", h.getRoomUsersHandler).Methods("GET")
	r.HandleFunc("/rooms/{roomID}/messages", h.getRoomMessagesHandler).Methods("GET")
	r.HandleFunc("/rooms/{roomID}/messages/{messageID}", h.getRoomMessageHandler).Methods("GET")
	r.HandleFunc("/rooms/{roomID}/messages/{messageID}", h.patchRoomMessageHandler).Methods("PATCH")
	r.HandleFunc("/rooms/{roomID}/messages/{messageID}", h.putRoomMessageHandler).Methods("PUT")
	r.HandleFunc("/rooms/{roomID}/messages/{messageID}", h.deleteRoomMessageHandler).Methods("DELETE")

	// User routes
	r.HandleFunc("/users", h.getAllUsersHandler).Methods("GET")
	r.HandleFunc("/users", h.createUserHandler).Methods("POST")
	r.HandleFunc("/users/{userID}", h.getUserHandler).Methods("GET")
	r.HandleFunc("/users/{userID}", h.putUserHandler).Methods("PUT")
	r.HandleFunc("/users/{userID}", h.patchUserHandler).Methods("PATCH")
	r.HandleFunc("/users/{userID}", h.deleteUserHandler).Methods("DELETE")

	// WebSocket route
	r.HandleFunc("/join/{roomID}", h.wsHandler).Methods("GET")

	// Info routes
	r.HandleFunc("/info", h.getInfoHandler).Methods("GET")
	r.HandleFunc("/healthz", h.healthzHandler).Methods("GET")
}

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
