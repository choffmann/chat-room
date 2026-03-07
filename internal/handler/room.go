package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/choffmann/chat-room/internal/model"
	"github.com/gorilla/mux"
)

// POST /rooms
func (h *Handler) createRoomHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var additionalInfo model.AdditionalInfo
	err := decoder.Decode(&additionalInfo)
	if err != nil {
		h.logger.Warn("failed to decode additional room info", "remoteAddr", r.RemoteAddr, "error", err)
		additionalInfo = map[string]any{}
	}
	room := h.hub.CreateRoom(additionalInfo)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]uint{"roomID": room.ID()})
}

// GET /rooms
func (h *Handler) getAllRoomsHandler(w http.ResponseWriter, r *http.Request) {
	rooms := h.hub.GetAllRoomIDs()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]model.RoomResponse{"rooms": rooms})
}

// GET /rooms/{roomID}
func (h *Handler) getRoomIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID, err := strconv.ParseUint(vars["roomID"], 10, 64)
	if err != nil {
		h.logger.Warn("invalid room id requested", "roomID", vars["roomID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "can't parse room id to uint", http.StatusBadRequest)
		return
	}

	room, ok := h.hub.GetRoom(uint(roomID))
	if !ok {
		h.logger.Warn("room not found", "roomID", roomID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	payload := model.RoomResponse{
		ID:             room.ID(),
		UserCount:      room.GetClientCount(),
		AdditionalInfo: room.GetAdditionalInfo(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}

// PATCH /rooms/{roomID}
func (h *Handler) patchRoomHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID, err := strconv.ParseUint(vars["roomID"], 10, 64)
	if err != nil {
		h.logger.Warn("invalid room id for patch", "roomID", vars["roomID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "can't parse room id to uint", http.StatusBadRequest)
		return
	}

	room, ok := h.hub.GetRoom(uint(roomID))
	if !ok {
		h.logger.Warn("room not found for patch", "roomID", roomID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	decoder := json.NewDecoder(r.Body)
	var updates model.AdditionalInfo
	err = decoder.Decode(&updates)
	if err != nil {
		h.logger.Warn("failed to decode patch request body", "roomID", roomID, "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	room.PatchAdditionalInfo(updates)
	h.logger.Info("room patched", "roomID", roomID)

	payload := model.RoomResponse{
		ID:             room.ID(),
		AdditionalInfo: room.GetAdditionalInfo(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}

// PUT /rooms/{roomID}
func (h *Handler) putRoomHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID, err := strconv.ParseUint(vars["roomID"], 10, 64)
	if err != nil {
		h.logger.Warn("invalid room id for put", "roomID", vars["roomID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "can't parse room id to uint", http.StatusBadRequest)
		return
	}

	room, ok := h.hub.GetRoom(uint(roomID))
	if !ok {
		h.logger.Warn("room not found for put", "roomID", roomID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	decoder := json.NewDecoder(r.Body)
	var newInfo model.AdditionalInfo
	err = decoder.Decode(&newInfo)
	if err != nil {
		h.logger.Warn("failed to decode put request body", "roomID", roomID, "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	room.UpdateAdditionalInfo(newInfo)
	h.logger.Info("room updated", "roomID", roomID)

	payload := model.RoomResponse{
		ID:             room.ID(),
		AdditionalInfo: room.GetAdditionalInfo(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}
