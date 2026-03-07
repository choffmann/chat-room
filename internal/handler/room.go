package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/choffmann/chat-room/internal/model"
	"github.com/gorilla/mux"
)

// createRoomHandler godoc
// @Summary      Create a new room
// @Description  Creates a new chat room. The request body is optional and can carry additional metadata that will be echoed back when the room is queried. If the JSON payload cannot be decoded, an empty additionalInfo is used instead.
// @Tags         rooms
// @Accept       json
// @Produce      json
// @Param        body  body      CreateRoomRequestDoc  false  "Optional room metadata (arbitrary JSON object)"
// @Success      200   {object}  CreateRoomResponse
// @Router       /rooms [post]
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

// getAllRoomsHandler godoc
// @Summary      List all rooms
// @Description  Retrieves all currently active rooms with user counts and metadata.
// @Tags         rooms
// @Produce      json
// @Success      200  {object}  RoomsListResponse
// @Router       /rooms [get]
func (h *Handler) getAllRoomsHandler(w http.ResponseWriter, r *http.Request) {
	rooms := h.hub.GetAllRoomIDs()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]model.RoomResponse{"rooms": rooms})
}

// getRoomIDHandler godoc
// @Summary      Get room details
// @Description  Returns metadata for a specific room including online user count.
// @Tags         rooms
// @Produce      json
// @Param        roomID  path      int  true  "Room ID"
// @Success      200     {object}  RoomResponseDoc
// @Failure      400     {string}  string  "can't parse room id to uint"
// @Failure      404     {string}  string  "room not found"
// @Router       /rooms/{roomID} [get]
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

// patchRoomHandler godoc
// @Summary      Partially update room metadata
// @Description  Partially updates room metadata. The provided fields are merged with existing additionalInfo, preserving fields not included in the request.
// @Tags         rooms
// @Accept       json
// @Produce      json
// @Param        roomID  path      int     true  "Room ID"
// @Param        body    body      PatchRoomRequestDoc  true  "Fields to merge into room metadata (arbitrary JSON object)"
// @Success      200     {object}  RoomResponseDoc
// @Failure      400     {string}  string  "invalid request body"
// @Failure      404     {string}  string  "room not found"
// @Router       /rooms/{roomID} [patch]
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

// putRoomHandler godoc
// @Summary      Replace room metadata
// @Description  Replaces all room metadata. This completely overwrites the existing additionalInfo.
// @Tags         rooms
// @Accept       json
// @Produce      json
// @Param        roomID  path      int     true  "Room ID"
// @Param        body    body      PutRoomRequestDoc  true  "New room metadata (arbitrary JSON object)"
// @Success      200     {object}  RoomResponseDoc
// @Failure      400     {string}  string  "invalid request body"
// @Failure      404     {string}  string  "room not found"
// @Router       /rooms/{roomID} [put]
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
