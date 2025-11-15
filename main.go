package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type MessageType string

const (
	SystemMessage MessageType = "system"
	UserMessage   MessageType = "message"
	ImageMessage  MessageType = "image"
)

type AdditionalInfo = map[string]any

type Client struct {
	room         *Room
	conn         *websocket.Conn
	user         User
	send         chan []byte
	closeMu      sync.Mutex
	closed       bool
	disconnected sync.Once
}

type User struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type OutgoingMessage struct {
	MessageType    MessageType    `json:"type"`
	Message        string         `json:"message"`
	Timestamp      time.Time      `json:"timestamp"`
	User           User           `json:"user"`
	AdditionalInfo AdditionalInfo `json:"additionalInfo"`
}

type IncomingMessage struct {
	MessageType    MessageType    `json:"type"`
	Message        string         `json:"message"`
	AdditionalInfo AdditionalInfo `json:"additionalInfo,omitempty"`
}

type RoomResponse struct {
	ID             uint           `json:"id"`
	UserCount      int            `json:"onlineUser"`
	AdditionalInfo AdditionalInfo `json:"additionalInfo,omitempty"`
}

var (
	hub      = &Hub{rooms: make(map[uint]*Room)}
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	roomCounter = 0
	roomMu      sync.Mutex
	systemUser  = User{
		ID:   uuid.New(),
		Name: "system",
	}
	defaultUserName = []string{
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
	}
)

var (
	version       string = "unknown"
	gitCommit     string = "unknown"
	gitBranch     string = "unknown"
	gitRepository string = "unknown"
	buildTime     string = "unknown"
)

// POST /rooms
func createRoomHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var additionalInfo AdditionalInfo
	err := decoder.Decode(&additionalInfo)
	if err != nil {
		logger.Warn("failed to decode additional room info", "remoteAddr", r.RemoteAddr, "error", err)
		additionalInfo = map[string]any{}
	}
	room := hub.CreateRoom(additionalInfo)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]uint{"roomID": room.id})
}

// GET /rooms
func getAllRoomsHandler(w http.ResponseWriter, r *http.Request) {
	rooms := hub.GetAllRoomIDs()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]RoomResponse{"rooms": rooms})
}

// GET /rooms/{roomID}
func getRoomIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID, err := strconv.ParseUint(vars["roomID"], 10, 64)
	if err != nil {
		logger.Warn("invalid room id requested", "roomID", vars["roomID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "can't parse room id to uint", http.StatusBadRequest)
		return
	}

	room, ok := hub.GetRoom(uint(roomID))
	if !ok {
		logger.Warn("room not found", "roomID", roomID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	payload := RoomResponse{
		ID:             room.id,
		AdditionalInfo: room.additionalInfo,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}

// PATCH /rooms/{roomID}
func patchRoomHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID, err := strconv.ParseUint(vars["roomID"], 10, 64)
	if err != nil {
		logger.Warn("invalid room id for patch", "roomID", vars["roomID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "can't parse room id to uint", http.StatusBadRequest)
		return
	}

	room, ok := hub.GetRoom(uint(roomID))
	if !ok {
		logger.Warn("room not found for patch", "roomID", roomID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	decoder := json.NewDecoder(r.Body)
	var updates AdditionalInfo
	err = decoder.Decode(&updates)
	if err != nil {
		logger.Warn("failed to decode patch request body", "roomID", roomID, "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	room.PatchAdditionalInfo(updates)
	logger.Info("room patched", "roomID", roomID)

	payload := RoomResponse{
		ID:             room.id,
		AdditionalInfo: room.GetAdditionalInfo(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}

// PUT /rooms/{roomID}
func putRoomHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID, err := strconv.ParseUint(vars["roomID"], 10, 64)
	if err != nil {
		logger.Warn("invalid room id for put", "roomID", vars["roomID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "can't parse room id to uint", http.StatusBadRequest)
		return
	}

	room, ok := hub.GetRoom(uint(roomID))
	if !ok {
		logger.Warn("room not found for put", "roomID", roomID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	decoder := json.NewDecoder(r.Body)
	var newInfo AdditionalInfo
	err = decoder.Decode(&newInfo)
	if err != nil {
		logger.Warn("failed to decode put request body", "roomID", roomID, "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	room.UpdateAdditionalInfo(newInfo)
	logger.Info("room updated", "roomID", roomID)

	payload := RoomResponse{
		ID:             room.id,
		AdditionalInfo: room.GetAdditionalInfo(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}

func (c *Client) closeSend() {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()
	if !c.closed {
		close(c.send)
		c.closed = true
	}
}

func (c *Client) disconnect() {
	c.disconnected.Do(func() {
		leaveMsg := OutgoingMessage{
			MessageType: SystemMessage,
			Message:     fmt.Sprintf("%s left room %d", c.user.Name, c.room.id),
			Timestamp:   time.Now(),
			User:        systemUser,
		}
		b, _ := json.Marshal(leaveMsg)
		if !c.room.tryBroadcast(b) {
			logger.Debug("failed to broadcast leave message, room may be closing", "roomID", c.room.id)
		}

		if !c.room.tryUnregister(c) {
			logger.Debug("failed to unregister client, room may be closing", "roomID", c.room.id, "userID", c.user.ID)
		}
	})
}

// GET /join/{roomID}?user=""
func wsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID, err := strconv.ParseUint(vars["roomID"], 10, 64)
	if err != nil {
		logger.Warn("invalid room id for websocket join", "roomID", vars["roomID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "can't parse room id to uint", http.StatusBadRequest)
		return
	}

	userName := r.URL.Query().Get("user")

	if userName == "" {
		userName = defaultUserName[rand.Intn(len(defaultUserName))]
	}

	user := User{
		ID:   uuid.New(),
		Name: userName,
	}

	room, ok := hub.GetRoom(uint(roomID))
	if !ok {
		logger.Warn("websocket join attempted for missing room", "roomID", roomID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("websocket upgrade failed", "roomID", roomID, "userID", user.ID, "userName", user.Name, "error", err)
		return
	}

	client := &Client{
		room: room,
		conn: conn,
		user: user,
		send: make(chan []byte, 256),
	}

	hello := OutgoingMessage{
		MessageType: SystemMessage,
		Message:     fmt.Sprintf("%s joined room %d", user.Name, roomID),
		Timestamp:   time.Now(),
		User:        systemUser,
	}

	b, _ := json.Marshal(hello)
	if !room.tryBroadcast(b) {
		logger.Warn("failed to broadcast join message, room may be closing", "roomID", roomID)
	}

	if !room.tryRegister(client) {
		logger.Warn("failed to register client, room may be closing", "roomID", roomID, "userID", user.ID)
		conn.Close()
		return
	}
	logger.Info("client joined room", "roomID", roomID, "userID", user.ID, "userName", user.Name)

	go client.writePump()
	client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.disconnect()
		c.conn.Close()
	}()

	c.conn.SetReadLimit(8192)
	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var message IncomingMessage
		if err := c.conn.ReadJSON(&message); err != nil {
			if !websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) && !strings.Contains(err.Error(), "use of closed network connection") {
				logger.Warn("websocket read failed", "roomID", c.room.id, "userID", c.user.ID, "error", err)
			}
			break
		}

		payload := OutgoingMessage{
			MessageType:    message.MessageType,
			Message:        message.Message,
			Timestamp:      time.Now(),
			User:           c.user,
			AdditionalInfo: message.AdditionalInfo,
		}
		b, _ := json.Marshal(payload)
		if !c.room.tryBroadcast(b) {
			logger.Warn("failed to broadcast message, room may be closing", "roomID", c.room.id, "userID", c.user.ID)
			break
		}
		logger.Info("new message received", "roomID", c.room.id, "userID", c.user.ID, "message", payload.Message)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.disconnect()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				logger.Warn("failed to write websocket message", "roomID", c.room.id, "userID", c.user.ID, "error", err)
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Warn("failed to send websocket ping", "roomID", c.room.id, "userID", c.user.ID, "error", err)
				return
			}
		}
	}
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/rooms", createRoomHandler).Methods("POST")
	r.HandleFunc("/rooms", getAllRoomsHandler).Methods("GET")
	r.HandleFunc("/rooms/{roomID}", getRoomIDHandler).Methods("GET")
	r.HandleFunc("/rooms/{roomID}", patchRoomHandler).Methods("PATCH")
	r.HandleFunc("/rooms/{roomID}", putRoomHandler).Methods("PUT")
	r.HandleFunc("/join/{roomID}", wsHandler).Methods("GET")
	r.HandleFunc("/info", getInfoHandler).Methods("GET")

	r.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      r,
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
