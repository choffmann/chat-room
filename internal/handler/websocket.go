package handler

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/choffmann/chat-room/internal/chat"
	"github.com/choffmann/chat-room/internal/model"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// wsHandler godoc
// @Summary      Join room via WebSocket
// @Description  Upgrades the HTTP connection to WebSocket and joins the requested room.
// @Description
// @Description  **Authentication options:**
// @Description  - `userId` (UUID): Join as a registered user from the registry. Takes precedence over `user`.
// @Description  - `user` (string): Join as an ephemeral user with the given display name.
// @Description  - Neither: Server assigns a random display name.
// @Description
// @Description  **User info extraction:** Set `userInfo=true` to receive a self-join message with a `self` flag, allowing clients to extract their user information.
// @Description
// @Description  **Connection management:** Server sends ping every 30s, expects pong within 60s. Max message size: 10 MiB.
// @Tags         websocket
// @Param        roomID    path   int     true   "Room ID"
// @Param        userId    query  string  false  "Registered user UUID"
// @Param        user      query  string  false  "Ephemeral display name"
// @Param        userInfo  query  bool    false  "Enable self-join message with user info"
// @Success      101       "Switching Protocols - WebSocket connection established"
// @Failure      400       {string}  string  "invalid room or user ID"
// @Failure      404       {string}  string  "room or user not found"
// @Router       /join/{roomID} [get]
func (h *Handler) wsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID, err := strconv.ParseUint(vars["roomID"], 10, 64)
	if err != nil {
		h.logger.Warn("invalid room id for websocket join", "roomID", vars["roomID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "can't parse room id to uint", http.StatusBadRequest)
		return
	}

	var user model.User

	userIDStr := r.URL.Query().Get("userId")
	if userIDStr != "" {
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			h.logger.Warn("invalid user id for websocket join", "userID", userIDStr, "remoteAddr", r.RemoteAddr, "error", err)
			http.Error(w, "invalid user id", http.StatusBadRequest)
			return
		}

		registeredUser, ok := h.userRegistry.GetUser(userID)
		if !ok {
			h.logger.Warn("user not found in registry", "userID", userID, "remoteAddr", r.RemoteAddr)
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		user = *registeredUser
		h.logger.Info("user from registry joining room", "userID", user.ID, "roomID", roomID)
	} else {
		userName := r.URL.Query().Get("user")
		if userName == "" {
			userName = h.defaultNames[rand.Intn(len(h.defaultNames))]
		}

		user = model.User{
			ID:   uuid.New(),
			Name: userName,
		}
		h.logger.Info("ephemeral user joining room", "userID", user.ID, "userName", user.Name, "roomID", roomID)
	}

	room, ok := h.hub.GetRoom(uint(roomID))
	if !ok {
		h.logger.Warn("websocket join attempted for missing room", "roomID", roomID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed", "roomID", roomID, "userID", user.ID, "userName", user.Name, "error", err)
		return
	}

	client := chat.NewClient(room, conn, user, h.systemUser, h.logger)

	displayName := model.GetDisplayName(user)
	timestamp := time.Now()

	supportsUserInfo := r.URL.Query().Get("userInfo") == "true"

	if supportsUserInfo {
		selfJoin := model.OutgoingMessage{
			ID:          uuid.New(),
			MessageType: model.SystemMessage,
			Message:     fmt.Sprintf("%s joined room %d", displayName, roomID),
			Timestamp:   timestamp,
			User:        h.systemUser,
			AdditionalInfo: model.AdditionalInfo{
				"joinedUserId":   user.ID.String(),
				"joinedUserName": displayName,
				"self":           true,
			},
		}
		selfJoinBytes, _ := json.Marshal(selfJoin)

		if err := conn.WriteMessage(websocket.TextMessage, selfJoinBytes); err != nil {
			h.logger.Warn("failed to send join message to new client", "roomID", roomID, "userID", user.ID, "error", err)
			conn.Close()
			return
		}
	}

	hello := model.OutgoingMessage{
		ID:          uuid.New(),
		MessageType: model.SystemMessage,
		Message:     fmt.Sprintf("%s joined room %d", displayName, roomID),
		Timestamp:   timestamp,
		User:        h.systemUser,
		AdditionalInfo: model.AdditionalInfo{
			"joinedUserId":   user.ID.String(),
			"joinedUserName": displayName,
		},
	}

	room.StoreMessage(hello)

	b, _ := json.Marshal(hello)
	if !room.TryBroadcast(b) {
		h.logger.Warn("failed to broadcast join message, room may be closing", "roomID", roomID)
	}

	if !room.TryRegister(client) {
		h.logger.Warn("failed to register client, room may be closing", "roomID", roomID, "userID", user.ID)
		conn.Close()
		return
	}
	h.logger.Info("client joined room", "roomID", roomID, "userID", user.ID, "userName", user.Name)

	go client.WritePump()
	client.ReadPump()
}
