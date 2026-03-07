package handler

import (
	"encoding/json"
	"net/http"

	"github.com/choffmann/chat-room/internal/model"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// GET /users
func (h *Handler) getAllUsersHandler(w http.ResponseWriter, r *http.Request) {
	users := h.userRegistry.GetAllUsers()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// POST /users
func (h *Handler) createUserHandler(w http.ResponseWriter, r *http.Request) {
	var req model.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("failed to decode user creation request", "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	user := h.userRegistry.CreateUser(req.FirstName, req.LastName, req.Name, req.AdditionalInfo)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

// GET /users/{userID}
func (h *Handler) getUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := uuid.Parse(vars["userID"])
	if err != nil {
		h.logger.Warn("invalid user id for get", "userID", vars["userID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	user, ok := h.userRegistry.GetUser(userID)
	if !ok {
		h.logger.Warn("user id not found in user registry", "userID", vars["userID"], "remoteAddr", r.RemoteAddr)
		http.Error(w, "user id not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// PUT /users/{userID}
func (h *Handler) putUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := uuid.Parse(vars["userID"])
	if err != nil {
		h.logger.Warn("invalid user id for put", "userID", vars["userID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	var req model.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("failed to decode user update request", "userID", userID, "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	user, ok := h.userRegistry.UpdateUser(userID, req.FirstName, req.LastName, req.Name, req.AdditionalInfo)
	if !ok {
		h.logger.Warn("user not found for update", "userID", userID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// PATCH /users/{userID}
func (h *Handler) patchUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := uuid.Parse(vars["userID"])
	if err != nil {
		h.logger.Warn("invalid user id for patch", "userID", vars["userID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		h.logger.Warn("failed to decode user patch request", "userID", userID, "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	user, ok := h.userRegistry.PatchUser(userID, updates)
	if !ok {
		h.logger.Warn("user not found for patch", "userID", userID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// DELETE /users/{userID}
func (h *Handler) deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := uuid.Parse(vars["userID"])
	if err != nil {
		h.logger.Warn("invalid user id for delete", "userID", vars["userID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	if !h.userRegistry.DeleteUser(userID) {
		h.logger.Warn("user not found for delete", "userID", userID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GET /rooms/{roomID}/users
func (h *Handler) getRoomUsersHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID, err := model.ParseRoomID(vars["roomID"])
	if err != nil {
		h.logger.Warn("invalid room id for get users", "roomID", vars["roomID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "invalid room id", http.StatusBadRequest)
		return
	}

	room, ok := h.hub.GetRoom(roomID)
	if !ok {
		h.logger.Warn("room not found for get users", "roomID", roomID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	users := room.GetUsers()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]model.User{"users": users})
}

// GET /rooms/users
func (h *Handler) getAllUsersInRoomsHandler(w http.ResponseWriter, r *http.Request) {
	usersWithRooms := h.hub.GetAllUsersWithRooms()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]model.UserWithRoom{"users": usersWithRooms})
}
