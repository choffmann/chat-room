package main

import (
	"encoding/json"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type UserRegistry struct {
	mu    sync.RWMutex
	users map[uuid.UUID]*User
}

var userRegistry = &UserRegistry{
	users: make(map[uuid.UUID]*User),
}

type CreateUserRequest struct {
	FirstName      string         `json:"firstName,omitempty"`
	LastName       string         `json:"lastName,omitempty"`
	Name           string         `json:"name,omitempty"`
	AdditionalInfo AdditionalInfo `json:"additionalInfo,omitempty"`
}

type UpdateUserRequest struct {
	FirstName      string         `json:"firstName,omitempty"`
	LastName       string         `json:"lastName,omitempty"`
	Name           string         `json:"name,omitempty"`
	AdditionalInfo AdditionalInfo `json:"additionalInfo,omitempty"`
}

type UserWithRoom struct {
	User   User `json:"user"`
	RoomID uint `json:"roomId"`
}

func parseRoomID(roomIDStr string) (uint, error) {
	roomID, err := strconv.ParseUint(roomIDStr, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(roomID), nil
}

func (ur *UserRegistry) CreateUser(firstName, lastName, name string, additionalInfo AdditionalInfo) *User {
	user := &User{
		ID:             uuid.New(),
		FirstName:      firstName,
		LastName:       lastName,
		Name:           name,
		AdditionalInfo: additionalInfo,
	}

	ur.mu.Lock()
	ur.users[user.ID] = user
	ur.mu.Unlock()

	logger.Info("user created", "userID", user.ID, "firstName", firstName, "lastName", lastName, "name", name)
	return user
}

func (ur *UserRegistry) GetUser(id uuid.UUID) (*User, bool) {
	ur.mu.RLock()
	defer ur.mu.RUnlock()
	user, ok := ur.users[id]
	return user, ok
}

func (ur *UserRegistry) GetAllUsers() []*User {
	ur.mu.RLock()
	defer ur.mu.RUnlock()

	return slices.Collect(maps.Values(ur.users))
}

func (ur *UserRegistry) UpdateUser(id uuid.UUID, firstName, lastName, name string, additionalInfo AdditionalInfo) (*User, bool) {
	ur.mu.Lock()
	defer ur.mu.Unlock()

	user, ok := ur.users[id]
	if !ok {
		return nil, false
	}

	user.FirstName = firstName
	user.LastName = lastName
	user.Name = name
	user.AdditionalInfo = additionalInfo

	logger.Info("user updated", "userID", id)
	return user, true
}

func (ur *UserRegistry) PatchUser(id uuid.UUID, updates map[string]any) (*User, bool) {
	ur.mu.Lock()
	defer ur.mu.Unlock()

	user, ok := ur.users[id]
	if !ok {
		return nil, false
	}

	if firstName, ok := updates["firstName"].(string); ok {
		user.FirstName = firstName
	}
	if lastName, ok := updates["lastName"].(string); ok {
		user.LastName = lastName
	}
	if name, ok := updates["name"].(string); ok {
		user.Name = name
	}
	if additionalInfo, ok := updates["additionalInfo"].(map[string]any); ok {
		if user.AdditionalInfo == nil {
			user.AdditionalInfo = make(AdditionalInfo)
		}
		maps.Copy(user.AdditionalInfo, additionalInfo)
	}

	logger.Info("user patched", "userID", id)
	return user, true
}

func (ur *UserRegistry) DeleteUser(id uuid.UUID) bool {
	ur.mu.Lock()
	defer ur.mu.Unlock()

	if _, ok := ur.users[id]; !ok {
		return false
	}

	delete(ur.users, id)
	logger.Info("user deleted", "userID", id)
	return true
}

// GET /users
func getAllUsersHandler(w http.ResponseWriter, r *http.Request) {
	user := userRegistry.GetAllUsers()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// POST /users
func createUserHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("failed to decode user creation request", "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	user := userRegistry.CreateUser(req.FirstName, req.LastName, req.Name, req.AdditionalInfo)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

// PUT /users/{userID}
func putUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := uuid.Parse(vars["userID"])
	if err != nil {
		logger.Warn("invalid user id for put", "userID", vars["userID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("failed to decode user update request", "userID", userID, "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	user, ok := userRegistry.UpdateUser(userID, req.FirstName, req.LastName, req.Name, req.AdditionalInfo)
	if !ok {
		logger.Warn("user not found for update", "userID", userID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// PATCH /users/{userID}
func patchUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := uuid.Parse(vars["userID"])
	if err != nil {
		logger.Warn("invalid user id for patch", "userID", vars["userID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		logger.Warn("failed to decode user patch request", "userID", userID, "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	user, ok := userRegistry.PatchUser(userID, updates)
	if !ok {
		logger.Warn("user not found for patch", "userID", userID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// DELETE /users/{userID}
func deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := uuid.Parse(vars["userID"])
	if err != nil {
		logger.Warn("invalid user id for delete", "userID", vars["userID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	if !userRegistry.DeleteUser(userID) {
		logger.Warn("user not found for delete", "userID", userID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GET /rooms/{roomID}/users
func getRoomUsersHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID, err := parseRoomID(vars["roomID"])
	if err != nil {
		logger.Warn("invalid room id for get users", "roomID", vars["roomID"], "remoteAddr", r.RemoteAddr, "error", err)
		http.Error(w, "invalid room id", http.StatusBadRequest)
		return
	}

	room, ok := hub.GetRoom(roomID)
	if !ok {
		logger.Warn("room not found for get users", "roomID", roomID, "remoteAddr", r.RemoteAddr)
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	users := room.GetUsers()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]User{"users": users})
}

// GET /rooms/users
func getAllUsersInRoomsHandler(w http.ResponseWriter, r *http.Request) {
	usersWithRooms := hub.GetAllUsersWithRooms()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]UserWithRoom{"users": usersWithRooms})
}
