package handler

import (
	"encoding/json"
	"net/http"

	"github.com/choffmann/chat-room/internal/model"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// getAllUsersHandler godoc
// @Summary      List all registered users
// @Description  Returns all users registered in the user registry. Returns an empty array if no users are registered.
// @Tags         users
// @Produce      json
// @Success      200  {array}  UserDoc
// @Router       /users [get]
func (h *Handler) getAllUsersHandler(w http.ResponseWriter, r *http.Request) {
	users := h.userRegistry.GetAllUsers()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// createUserHandler godoc
// @Summary      Create a user
// @Description  Creates a new user in the user registry. All fields are optional. This allows pre-registering users before they join rooms.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body      CreateUserRequestDoc  true  "User data"
// @Success      201   {object}  UserDoc
// @Failure      400   {string}  string  "invalid request body"
// @Router       /users [post]
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

// getUserHandler godoc
// @Summary      Get a user
// @Description  Returns a specific user from the user registry.
// @Tags         users
// @Produce      json
// @Param        userID  path      string  true  "User UUID"
// @Success      200     {object}  UserDoc
// @Failure      400     {string}  string  "invalid user id"
// @Failure      404     {string}  string  "user id not found"
// @Router       /users/{userID} [get]
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

// putUserHandler godoc
// @Summary      Replace a user
// @Description  Completely replaces all user information. Fields not included will be cleared.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        userID  path      string                  true  "User UUID"
// @Param        body    body      UpdateUserRequestDoc  true  "New user data"
// @Success      200     {object}  UserDoc
// @Failure      400     {string}  string  "invalid user id or request body"
// @Failure      404     {string}  string  "user not found"
// @Router       /users/{userID} [put]
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

// patchUserHandler godoc
// @Summary      Partially update a user
// @Description  Partially updates user information. Only provided fields are updated, others remain unchanged.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        userID  path      string  true  "User UUID"
// @Param        body    body      PatchUserRequestDoc  true  "Fields to update"
// @Success      200     {object}  UserDoc
// @Failure      400     {string}  string  "invalid user id or request body"
// @Failure      404     {string}  string  "user not found"
// @Router       /users/{userID} [patch]
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

// deleteUserHandler godoc
// @Summary      Delete a user
// @Description  Deletes a user from the user registry.
// @Tags         users
// @Param        userID  path      string  true  "User UUID"
// @Success      204     "No Content"
// @Failure      400     {string}  string  "invalid user id"
// @Failure      404     {string}  string  "user not found"
// @Router       /users/{userID} [delete]
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

// getRoomUsersHandler godoc
// @Summary      Get users in a room
// @Description  Returns all users currently connected to a specific room.
// @Tags         rooms
// @Produce      json
// @Param        roomID  path      int  true  "Room ID"
// @Success      200     {object}  UsersListResponse
// @Failure      400     {string}  string  "invalid room id"
// @Failure      404     {string}  string  "room not found"
// @Router       /rooms/{roomID}/users [get]
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

// getAllUsersInRoomsHandler godoc
// @Summary      Get all users in all rooms
// @Description  Returns all users currently connected to any room, along with their room IDs.
// @Tags         rooms
// @Produce      json
// @Success      200  {object}  UsersWithRoomListResponse
// @Router       /rooms/users [get]
func (h *Handler) getAllUsersInRoomsHandler(w http.ResponseWriter, r *http.Request) {
	usersWithRooms := h.hub.GetAllUsersWithRooms()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]model.UserWithRoom{"users": usersWithRooms})
}
