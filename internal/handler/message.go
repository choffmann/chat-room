package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/choffmann/chat-room/internal/model"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// GET /rooms/{roomID}/messages
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

// GET /rooms/{roomID}/messages/{messageID}
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

// PATCH /rooms/{roomID}/messages/{messageID}
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

	var patchRequest struct {
		Message        *string              `json:"message,omitempty"`
		AdditionalInfo model.AdditionalInfo `json:"additionalInfo,omitempty"`
	}
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

// PUT /rooms/{roomID}/messages/{messageID}
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

	var putRequest struct {
		Message        string               `json:"message"`
		AdditionalInfo model.AdditionalInfo `json:"additionalInfo,omitempty"`
	}
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

// DELETE /rooms/{roomID}/messages/{messageID}
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
