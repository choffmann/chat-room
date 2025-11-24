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
	ID             uuid.UUID      `json:"id"`
	FirstName      string         `json:"firstName,omitempty"`
	LastName       string         `json:"lastName,omitempty"`
	Name           string         `json:"name,omitempty"`
	AdditionalInfo AdditionalInfo `json:"additionalInfo,omitempty"`
}

func getDisplayName(user User) string {
	displayName := user.Name
	if displayName == "" && user.FirstName != "" && user.LastName != "" {
		return fmt.Sprintf("%s %s", user.FirstName, user.LastName)
	} else if displayName == "" && user.FirstName != "" {
		return user.FirstName
	} else if displayName == "" {
		return "Anonymous"
	}
	return displayName
}

type OutgoingMessage struct {
	ID             uuid.UUID      `json:"id"`
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
	messageMu   sync.Mutex
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
		displayName := getDisplayName(c.user)
		timestamp := time.Now()

		leaveMsg := OutgoingMessage{
			ID:          uuid.New(),
			MessageType: SystemMessage,
			Message:     fmt.Sprintf("%s left room %d", displayName, c.room.id),
			Timestamp:   timestamp,
			User:        systemUser,
		}

		c.room.StoreMessage(leaveMsg)

		b, _ := json.Marshal(leaveMsg)
		if !c.room.tryBroadcast(b) {
			logger.Debug("failed to broadcast leave message, room may be closing", "roomID", c.room.id)
		}

		if !c.room.tryUnregister(c) {
			logger.Debug("failed to unregister client, room may be closing", "roomID", c.room.id, "userID", c.user.ID)
		}
	})
}

// GET /rooms/{roomID}/messages
func getRoomMessagesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID, err := strconv.ParseUint(vars["roomID"], 10, 64)
	if err != nil {
		logger.Warn("invalid room id for getting messages", "roomID", vars["roomID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "can't parse room id to uint", http.StatusBadRequest)
		return
	}

	room, ok := hub.GetRoom(uint(roomID))
	if !ok {
		logger.Warn("room not found for getting messages", "roomID", roomID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	messages := room.GetMessages()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]OutgoingMessage{"messages": messages})
}

// GET /rooms/{roomID}/messages/{messageID}
func getRoomMessageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID, err := strconv.ParseUint(vars["roomID"], 10, 64)
	if err != nil {
		logger.Warn("invalid room id for getting message", "roomID", vars["roomID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "can't parse room id to uint", http.StatusBadRequest)
		return
	}

	messageID, err := uuid.Parse(vars["messageID"])
	if err != nil {
		logger.Warn("invalid message id for getting", "messageID", vars["messageID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "can't parse message id to uuid", http.StatusBadRequest)
		return
	}

	room, ok := hub.GetRoom(uint(roomID))
	if !ok {
		logger.Warn("room not found for getting message", "roomID", roomID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	message, success := room.GetMessage(messageID)
	if !success {
		logger.Warn("message not found for getting", "roomID", roomID, "messageID", messageID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "message not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(message)
}

// PATCH /rooms/{roomID}/messages/{messageID}
func patchRoomMessageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID, err := strconv.ParseUint(vars["roomID"], 10, 64)
	if err != nil {
		logger.Warn("invalid room id for patching message", "roomID", vars["roomID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "can't parse room id to uint", http.StatusBadRequest)
		return
	}

	messageID, err := uuid.Parse(vars["messageID"])
	if err != nil {
		logger.Warn("invalid message id for patching", "messageID", vars["messageID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "can't parse message id to uuid", http.StatusBadRequest)
		return
	}

	room, ok := hub.GetRoom(uint(roomID))
	if !ok {
		logger.Warn("room not found for patching message", "roomID", roomID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	var patchRequest struct {
		Message        *string        `json:"message,omitempty"`
		AdditionalInfo AdditionalInfo `json:"additionalInfo,omitempty"`
	}
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&patchRequest)
	if err != nil {
		logger.Warn("failed to decode message patch request", "roomID", roomID, "messageID", messageID, "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// At least one field must be provided
	if patchRequest.Message == nil && patchRequest.AdditionalInfo == nil {
		logger.Warn("no fields to patch in message update request", "roomID", roomID, "messageID", messageID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "at least one field (message or additionalInfo) must be provided", http.StatusBadRequest)
		return
	}

	// If message is provided, it should not be empty
	if patchRequest.Message != nil && *patchRequest.Message == "" {
		logger.Warn("empty message content in patch request", "roomID", roomID, "messageID", messageID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "message content cannot be empty", http.StatusBadRequest)
		return
	}

	success := room.PatchMessage(messageID, patchRequest.Message, patchRequest.AdditionalInfo)
	if !success {
		logger.Warn("message not found for patch", "roomID", roomID, "messageID", messageID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "message not found", http.StatusNotFound)
		return
	}

	logger.Info("message patched", "roomID", roomID, "messageID", messageID)

	updatedMessage, _ := room.GetMessage(messageID)
	b, _ := json.Marshal(updatedMessage)
	room.tryBroadcast(b)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedMessage)
}

// PUT /rooms/{roomID}/messages/{messageID}
func putRoomMessageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID, err := strconv.ParseUint(vars["roomID"], 10, 64)
	if err != nil {
		logger.Warn("invalid room id for updating message", "roomID", vars["roomID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "can't parse room id to uint", http.StatusBadRequest)
		return
	}

	messageID, err := uuid.Parse(vars["messageID"])
	if err != nil {
		logger.Warn("invalid message id for updating", "messageID", vars["messageID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "can't parse message id to uuid", http.StatusBadRequest)
		return
	}

	room, ok := hub.GetRoom(uint(roomID))
	if !ok {
		logger.Warn("room not found for updating message", "roomID", roomID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	var patchRequest struct {
		Message        string         `json:"message"`
		AdditionalInfo AdditionalInfo `json:"additionalInfo,omitempty"`
	}
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&patchRequest)
	if err != nil {
		logger.Warn("failed to decode message put request", "roomID", roomID, "messageID", messageID, "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	success := room.UpdateMessage(messageID, patchRequest.Message, patchRequest.AdditionalInfo)
	if !success {
		logger.Warn("message not found for updating", "roomID", roomID, "messageID", messageID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "message not found", http.StatusNotFound)
		return
	}

	logger.Info("message updated", "roomID", roomID, "messageID", messageID)

	updatedMessage, _ := room.GetMessage(messageID)
	b, _ := json.Marshal(updatedMessage)
	room.tryBroadcast(b)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedMessage)
}

// DELETE /rooms/{roomID}/messages/{messageID}
func deleteRoomMessageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID, err := strconv.ParseUint(vars["roomID"], 10, 64)
	if err != nil {
		logger.Warn("invalid room id for deleting message", "roomID", vars["roomID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "can't parse room id to uint", http.StatusBadRequest)
		return
	}

	messageID, err := uuid.Parse(vars["messageID"])
	if err != nil {
		logger.Warn("invalid message id for deleting", "messageID", vars["messageID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "can't parse message id to uuid", http.StatusBadRequest)
		return
	}

	room, ok := hub.GetRoom(uint(roomID))
	if !ok {
		logger.Warn("room not found for deleting message", "roomID", roomID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	success := room.UpdateMessage(messageID, "deleted", AdditionalInfo{"deleted": true})
	if !success {
		logger.Warn("message not found for deleting", "roomID", roomID, "messageID", messageID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "message not found", http.StatusNotFound)
		return
	}

	logger.Info("message deleted", "roomID", roomID, "messageID", messageID)

	deletedMessage, _ := room.GetMessage(messageID)
	b, _ := json.Marshal(deletedMessage)
	room.tryBroadcast(b)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(deletedMessage)
}

// GET /join/{roomID}?user=""&userId=""
func wsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID, err := strconv.ParseUint(vars["roomID"], 10, 64)
	if err != nil {
		logger.Warn("invalid room id for websocket join", "roomID", vars["roomID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "can't parse room id to uint", http.StatusBadRequest)
		return
	}

	var user User

	// Check if userId parameter is provided
	userIDStr := r.URL.Query().Get("userId")
	if userIDStr != "" {
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			logger.Warn("invalid user id for websocket join", "userID", userIDStr, "remoteAddr", r.RemoteAddr, "error", err)
			http.Error(w, "invalid user id", http.StatusBadRequest)
			return
		}

		// Try to get user from registry
		registeredUser, ok := userRegistry.GetUser(userID)
		if !ok {
			logger.Warn("user not found in registry", "userID", userID, "remoteAddr", r.RemoteAddr)
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		user = *registeredUser
		logger.Info("user from registry joining room", "userID", user.ID, "roomID", roomID)
	} else {
		// Fallback: create ephemeral user from query parameter
		userName := r.URL.Query().Get("user")
		if userName == "" {
			userName = defaultUserName[rand.Intn(len(defaultUserName))]
		}

		user = User{
			ID:   uuid.New(),
			Name: userName,
		}
		logger.Info("ephemeral user joining room", "userID", user.ID, "userName", user.Name, "roomID", roomID)
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

	displayName := getDisplayName(user)
	timestamp := time.Now()

	hello := OutgoingMessage{
		ID:          uuid.New(),
		MessageType: SystemMessage,
		Message:     fmt.Sprintf("%s joined room %d", displayName, roomID),
		Timestamp:   timestamp,
		User:        systemUser,
	}

	room.StoreMessage(hello)

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

func shouldStoreMessage(msgType MessageType) bool {
	return msgType == SystemMessage || msgType == UserMessage
}

const (
	B   = 1
	KiB = 1024 * B
	MiB = 1024 * KiB
)

func (c *Client) readPump() {
	defer func() {
		c.disconnect()
		c.conn.Close()
	}()

	c.conn.SetReadLimit(10 * MiB)
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

		timestamp := time.Now()

		payload := OutgoingMessage{
			ID:             uuid.New(),
			MessageType:    message.MessageType,
			Message:        message.Message,
			Timestamp:      timestamp,
			User:           c.user,
			AdditionalInfo: message.AdditionalInfo,
		}

		b, _ := json.Marshal(payload)
		if !c.room.tryBroadcast(b) {
			logger.Warn("failed to broadcast message, room may be closing", "roomID", c.room.id, "userID", c.user.ID)
			break
		}

		if shouldStoreMessage(message.MessageType) && len(b) < 2*MiB && len(b) > 0 {
			c.room.StoreMessage(payload)
		}

		logger.Info("new message received", "roomID", c.room.id, "userID", c.user.ID, "messageID", payload.ID, "messageType", payload.MessageType)
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

	// Room routes
	r.HandleFunc("/rooms", createRoomHandler).Methods("POST")
	r.HandleFunc("/rooms", getAllRoomsHandler).Methods("GET")
	r.HandleFunc("/rooms/users", getAllUsersInRoomsHandler).Methods("GET")
	r.HandleFunc("/rooms/{roomID}", getRoomIDHandler).Methods("GET")
	r.HandleFunc("/rooms/{roomID}", patchRoomHandler).Methods("PATCH")
	r.HandleFunc("/rooms/{roomID}", putRoomHandler).Methods("PUT")
	r.HandleFunc("/rooms/{roomID}/users", getRoomUsersHandler).Methods("GET")
	r.HandleFunc("/rooms/{roomID}/messages", getRoomMessagesHandler).Methods("GET")
	r.HandleFunc("/rooms/{roomID}/messages/{messageID}", getRoomMessageHandler).Methods("GET")
	r.HandleFunc("/rooms/{roomID}/messages/{messageID}", patchRoomMessageHandler).Methods("PATCH")
	r.HandleFunc("/rooms/{roomID}/messages/{messageID}", putRoomMessageHandler).Methods("PUT")
	r.HandleFunc("/rooms/{roomID}/messages/{messageID}", deleteRoomMessageHandler).Methods("DELETE")

	// User routes
	r.HandleFunc("/users", getAllUsersHandler).Methods("GET")
	r.HandleFunc("/users", createUserHandler).Methods("POST")
	r.HandleFunc("/users/{userID}", putUserHandler).Methods("PUT")
	r.HandleFunc("/users/{userID}", patchUserHandler).Methods("PATCH")
	r.HandleFunc("/users/{userID}", deleteUserHandler).Methods("DELETE")

	// WebSocket route
	r.HandleFunc("/join/{roomID}", wsHandler).Methods("GET")

	// Info routes
	r.HandleFunc("/info", getInfoHandler).Methods("GET")
	r.HandleFunc("/healthz", healthzHandler).Methods("GET")

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
