package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/choffmann/chat-room/internal/model"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type MessagePatchRequest struct {
	Message        *string              `json:"message,omitempty" example:"Hello everyone! (edited)"`
	AdditionalInfo model.AdditionalInfo `json:"additionalInfo,omitempty" swaggertype:"object"`
}

type MessagePutRequest struct {
	Message        string               `json:"message" example:"Completely new message content"`
	AdditionalInfo model.AdditionalInfo `json:"additionalInfo,omitempty" swaggertype:"object"`
}

// getRoomMessagesHandler godoc
// @Summary      Get all messages in a room
// @Description  Returns all messages that have been sent in a specific room. Messages are stored in memory and include system messages (joins/leaves) as well as user messages. Only messages smaller than 2 MiB are stored.
// @Tags         messages
// @Produce      json
// @Param        roomID  path      int  true  "Room ID"
// @Success      200     {object}  MessagesListResponse
// @Failure      400     {string}  string  "can't parse room id to uint"
// @Failure      404     {string}  string  "room not found"
// @Router       /rooms/{roomID}/messages [get]
func (h *Handler) getRoomMessagesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID, err := strconv.ParseUint(vars["roomID"], 10, 64)
	if err != nil {
		h.logger.Warn("invalid room id for getting messages", "roomID", vars["roomID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "can't parse room id to uint", http.StatusBadRequest)
		return
	}

	room, ok := h.hub.GetRoom(uint(roomID))
	if !ok {
		h.logger.Warn("room not found for getting messages", "roomID", roomID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	messages := room.GetMessages()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]model.OutgoingMessage{"messages": messages})
}

// getRoomMessageHandler godoc
// @Summary      Get a specific message
// @Description  Retrieves a specific message from a room by its ID.
// @Tags         messages
// @Produce      json
// @Param        roomID     path      int     true  "Room ID"
// @Param        messageID  path      string  true  "Message UUID"
// @Success      200        {object}  OutgoingMessageDoc
// @Failure      400        {string}  string  "can't parse room id or message id"
// @Failure      404        {string}  string  "room or message not found"
// @Router       /rooms/{roomID}/messages/{messageID} [get]
func (h *Handler) getRoomMessageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID, err := strconv.ParseUint(vars["roomID"], 10, 64)
	if err != nil {
		h.logger.Warn("invalid room id for getting message", "roomID", vars["roomID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "can't parse room id to uint", http.StatusBadRequest)
		return
	}

	messageID, err := uuid.Parse(vars["messageID"])
	if err != nil {
		h.logger.Warn("invalid message id for getting", "messageID", vars["messageID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "can't parse message id to uuid", http.StatusBadRequest)
		return
	}

	room, ok := h.hub.GetRoom(uint(roomID))
	if !ok {
		h.logger.Warn("room not found for getting message", "roomID", roomID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	message, success := room.GetMessage(messageID)
	if !success {
		h.logger.Warn("message not found for getting", "roomID", roomID, "messageID", messageID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "message not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(message)
}

// patchRoomMessageHandler godoc
// @Summary      Partially update a message
// @Description  Partially updates a specific message. You can update the message text, additionalInfo, or both. Only provided fields are updated. The server automatically sets modified: true in additionalInfo.
// @Tags         messages
// @Accept       json
// @Produce      json
// @Param        roomID     path      int                  true  "Room ID"
// @Param        messageID  path      string               true  "Message UUID"
// @Param        body       body      MessagePatchRequestDoc  true  "Fields to update"
// @Success      200        {object}  OutgoingMessageDoc
// @Failure      400        {string}  string  "invalid request"
// @Failure      404        {string}  string  "room or message not found"
// @Router       /rooms/{roomID}/messages/{messageID} [patch]
func (h *Handler) patchRoomMessageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID, err := strconv.ParseUint(vars["roomID"], 10, 64)
	if err != nil {
		h.logger.Warn("invalid room id for patching message", "roomID", vars["roomID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "can't parse room id to uint", http.StatusBadRequest)
		return
	}

	messageID, err := uuid.Parse(vars["messageID"])
	if err != nil {
		h.logger.Warn("invalid message id for patching", "messageID", vars["messageID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "can't parse message id to uuid", http.StatusBadRequest)
		return
	}

	room, ok := h.hub.GetRoom(uint(roomID))
	if !ok {
		h.logger.Warn("room not found for patching message", "roomID", roomID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	var patchRequest MessagePatchRequest
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&patchRequest)
	if err != nil {
		h.logger.Warn("failed to decode message patch request", "roomID", roomID, "messageID", messageID, "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if patchRequest.Message == nil && patchRequest.AdditionalInfo == nil {
		h.logger.Warn("no fields to patch in message update request", "roomID", roomID, "messageID", messageID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "at least one field (message or additionalInfo) must be provided", http.StatusBadRequest)
		return
	}

	if patchRequest.Message != nil && *patchRequest.Message == "" {
		h.logger.Warn("empty message content in patch request", "roomID", roomID, "messageID", messageID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "message content cannot be empty", http.StatusBadRequest)
		return
	}

	success := room.PatchMessage(messageID, patchRequest.Message, patchRequest.AdditionalInfo)
	if !success {
		h.logger.Warn("message not found for patch", "roomID", roomID, "messageID", messageID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "message not found", http.StatusNotFound)
		return
	}

	h.logger.Info("message patched", "roomID", roomID, "messageID", messageID)

	updatedMessage, _ := room.GetMessage(messageID)
	b, _ := json.Marshal(updatedMessage)
	room.TryBroadcast(b)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedMessage)
}

// putRoomMessageHandler godoc
// @Summary      Replace a message
// @Description  Completely replaces a message. Unlike PATCH, this requires all fields and replaces the entire message content. The server automatically sets modified: true in additionalInfo.
// @Tags         messages
// @Accept       json
// @Produce      json
// @Param        roomID     path      int                true  "Room ID"
// @Param        messageID  path      string             true  "Message UUID"
// @Param        body       body      MessagePutRequestDoc  true  "New message content"
// @Success      200        {object}  OutgoingMessageDoc
// @Failure      400        {string}  string  "invalid request"
// @Failure      404        {string}  string  "room or message not found"
// @Router       /rooms/{roomID}/messages/{messageID} [put]
func (h *Handler) putRoomMessageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID, err := strconv.ParseUint(vars["roomID"], 10, 64)
	if err != nil {
		h.logger.Warn("invalid room id for updating message", "roomID", vars["roomID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "can't parse room id to uint", http.StatusBadRequest)
		return
	}

	messageID, err := uuid.Parse(vars["messageID"])
	if err != nil {
		h.logger.Warn("invalid message id for updating", "messageID", vars["messageID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "can't parse message id to uuid", http.StatusBadRequest)
		return
	}

	room, ok := h.hub.GetRoom(uint(roomID))
	if !ok {
		h.logger.Warn("room not found for updating message", "roomID", roomID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	var putRequest MessagePutRequest
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&putRequest)
	if err != nil {
		h.logger.Warn("failed to decode message put request", "roomID", roomID, "messageID", messageID, "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	success := room.UpdateMessage(messageID, putRequest.Message, putRequest.AdditionalInfo)
	if !success {
		h.logger.Warn("message not found for updating", "roomID", roomID, "messageID", messageID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "message not found", http.StatusNotFound)
		return
	}

	h.logger.Info("message updated", "roomID", roomID, "messageID", messageID)

	updatedMessage, _ := room.GetMessage(messageID)
	b, _ := json.Marshal(updatedMessage)
	room.TryBroadcast(b)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedMessage)
}

// deleteRoomMessageHandler godoc
// @Summary      Delete a message
// @Description  Marks a message as deleted. The message is not actually removed but its content is replaced with "deleted" and a deleted flag is added to additionalInfo.
// @Tags         messages
// @Produce      json
// @Param        roomID     path      int     true  "Room ID"
// @Param        messageID  path      string  true  "Message UUID"
// @Success      200        {object}  OutgoingMessageDoc
// @Failure      400        {string}  string  "can't parse room id or message id"
// @Failure      404        {string}  string  "room or message not found"
// @Router       /rooms/{roomID}/messages/{messageID} [delete]
func (h *Handler) deleteRoomMessageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID, err := strconv.ParseUint(vars["roomID"], 10, 64)
	if err != nil {
		h.logger.Warn("invalid room id for deleting message", "roomID", vars["roomID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "can't parse room id to uint", http.StatusBadRequest)
		return
	}

	messageID, err := uuid.Parse(vars["messageID"])
	if err != nil {
		h.logger.Warn("invalid message id for deleting", "messageID", vars["messageID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "can't parse message id to uuid", http.StatusBadRequest)
		return
	}

	room, ok := h.hub.GetRoom(uint(roomID))
	if !ok {
		h.logger.Warn("room not found for deleting message", "roomID", roomID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	success := room.UpdateMessage(messageID, "deleted", model.AdditionalInfo{"deleted": true})
	if !success {
		h.logger.Warn("message not found for deleting", "roomID", roomID, "messageID", messageID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "message not found", http.StatusNotFound)
		return
	}

	h.logger.Info("message deleted", "roomID", roomID, "messageID", messageID)

	deletedMessage, _ := room.GetMessage(messageID)
	b, _ := json.Marshal(deletedMessage)
	room.TryBroadcast(b)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(deletedMessage)
}
