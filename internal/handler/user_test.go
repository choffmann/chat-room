package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/choffmann/chat-room/internal/chat"
	"github.com/choffmann/chat-room/internal/model"
	"github.com/choffmann/chat-room/internal/user"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func TestCreateUser(t *testing.T) {
	h := setupHandler(t)

	tests := []struct {
		name           string
		payload        model.CreateUserRequest
		expectedStatus int
	}{
		{
			name: "Create user with all fields",
			payload: model.CreateUserRequest{
				FirstName: "John",
				LastName:  "Doe",
				Name:      "johndoe",
				AdditionalInfo: model.AdditionalInfo{
					"email": "john@example.com",
				},
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Create user with minimal fields",
			payload: model.CreateUserRequest{
				FirstName: "Jane",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "Create user with empty payload",
			payload:        model.CreateUserRequest{},
			expectedStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("POST", "/users", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			h.createUserHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if w.Code == http.StatusCreated {
				var user model.User
				if err := json.NewDecoder(w.Body).Decode(&user); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if user.ID == uuid.Nil {
					t.Error("expected user ID to be set")
				}

				if user.FirstName != tt.payload.FirstName {
					t.Errorf("expected firstName %s, got %s", tt.payload.FirstName, user.FirstName)
				}
			}
		})
	}
}

func TestCreateUserInvalidJSON(t *testing.T) {
	h := setupHandler(t)

	req := httptest.NewRequest("POST", "/users", bytes.NewBufferString("invalid json"))
	w := httptest.NewRecorder()

	h.createUserHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestPutUser(t *testing.T) {
	h := setupHandler(t)

	u := h.userRegistry.CreateUser("John", "Doe", "johndoe", nil)

	tests := []struct {
		name           string
		userID         string
		payload        model.UpdateUserRequest
		expectedStatus int
	}{
		{
			name:   "Update existing user",
			userID: u.ID.String(),
			payload: model.UpdateUserRequest{
				FirstName: "Jane",
				LastName:  "Smith",
				Name:      "janesmith",
				AdditionalInfo: model.AdditionalInfo{
					"email": "jane@example.com",
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Update non-existent user",
			userID: uuid.New().String(),
			payload: model.UpdateUserRequest{
				FirstName: "Test",
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid user ID",
			userID:         "invalid-uuid",
			payload:        model.UpdateUserRequest{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("PUT", "/users/"+tt.userID, bytes.NewBuffer(body))
			req = mux.SetURLVars(req, map[string]string{"userID": tt.userID})
			w := httptest.NewRecorder()

			h.putUserHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if w.Code == http.StatusOK {
				var updatedUser model.User
				if err := json.NewDecoder(w.Body).Decode(&updatedUser); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if updatedUser.FirstName != tt.payload.FirstName {
					t.Errorf("expected firstName %s, got %s", tt.payload.FirstName, updatedUser.FirstName)
				}
			}
		})
	}
}

func TestPatchUser(t *testing.T) {
	h := setupHandler(t)

	u := h.userRegistry.CreateUser("John", "Doe", "johndoe", model.AdditionalInfo{"role": "user"})

	tests := []struct {
		name           string
		userID         string
		payload        map[string]any
		expectedStatus int
		checkFunc      func(*testing.T, model.User)
	}{
		{
			name:   "Patch firstName only",
			userID: u.ID.String(),
			payload: map[string]any{
				"firstName": "Jane",
			},
			expectedStatus: http.StatusOK,
			checkFunc: func(t *testing.T, u model.User) {
				if u.FirstName != "Jane" {
					t.Errorf("expected firstName Jane, got %s", u.FirstName)
				}
				if u.LastName != "Doe" {
					t.Errorf("expected lastName Doe (unchanged), got %s", u.LastName)
				}
			},
		},
		{
			name:   "Patch additionalInfo",
			userID: u.ID.String(),
			payload: map[string]any{
				"additionalInfo": map[string]any{
					"email": "test@example.com",
				},
			},
			expectedStatus: http.StatusOK,
			checkFunc: func(t *testing.T, u model.User) {
				if u.AdditionalInfo["email"] != "test@example.com" {
					t.Error("expected additionalInfo to be updated")
				}
			},
		},
		{
			name:           "Patch non-existent user",
			userID:         uuid.New().String(),
			payload:        map[string]any{"firstName": "Test"},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid user ID",
			userID:         "invalid-uuid",
			payload:        map[string]any{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("PATCH", "/users/"+tt.userID, bytes.NewBuffer(body))
			req = mux.SetURLVars(req, map[string]string{"userID": tt.userID})
			w := httptest.NewRecorder()

			h.patchUserHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if w.Code == http.StatusOK && tt.checkFunc != nil {
				var patchedUser model.User
				if err := json.NewDecoder(w.Body).Decode(&patchedUser); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				tt.checkFunc(t, patchedUser)
			}
		})
	}
}

func TestGetUser(t *testing.T) {
	h := setupHandler(t)

	u := h.userRegistry.CreateUser("John", "Doe", "johndoe", model.AdditionalInfo{"email": "john@example.com"})

	tests := []struct {
		name           string
		userID         string
		expectedStatus int
	}{
		{
			name:           "Get existing user",
			userID:         u.ID.String(),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Get non-existent user",
			userID:         uuid.New().String(),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid user ID",
			userID:         "invalid-uuid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/users/"+tt.userID, nil)
			req = mux.SetURLVars(req, map[string]string{"userID": tt.userID})
			w := httptest.NewRecorder()

			h.getUserHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if w.Code == http.StatusOK {
				var fetchedUser model.User
				if err := json.NewDecoder(w.Body).Decode(&fetchedUser); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if fetchedUser.ID != u.ID {
					t.Errorf("expected user ID %s, got %s", u.ID, fetchedUser.ID)
				}
				if fetchedUser.FirstName != u.FirstName {
					t.Errorf("expected firstName %s, got %s", u.FirstName, fetchedUser.FirstName)
				}
				if fetchedUser.LastName != u.LastName {
					t.Errorf("expected lastName %s, got %s", u.LastName, fetchedUser.LastName)
				}
				if fetchedUser.Name != u.Name {
					t.Errorf("expected name %s, got %s", u.Name, fetchedUser.Name)
				}
			}
		})
	}
}

func TestDeleteUser(t *testing.T) {
	h := setupHandler(t)

	u := h.userRegistry.CreateUser("John", "Doe", "johndoe", nil)

	tests := []struct {
		name           string
		userID         string
		expectedStatus int
	}{
		{
			name:           "Delete existing user",
			userID:         u.ID.String(),
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "Delete non-existent user",
			userID:         uuid.New().String(),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid user ID",
			userID:         "invalid-uuid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", "/users/"+tt.userID, nil)
			req = mux.SetURLVars(req, map[string]string{"userID": tt.userID})
			w := httptest.NewRecorder()

			h.deleteUserHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestGetRoomUsers(t *testing.T) {
	logger := testLogger()
	hub := chat.NewHub(logger)
	ur := user.NewRegistry(logger)
	h := New(hub, ur, logger)

	room := hub.CreateRoom(nil)
	close(room.Shutdown())

	// We can't directly add clients from outside the chat package,
	// so we test the empty case and the route itself
	req := httptest.NewRequest("GET", "/rooms/1/users", nil)
	req = mux.SetURLVars(req, map[string]string{"roomID": "1"})
	w := httptest.NewRecorder()

	h.getRoomUsersHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string][]model.User
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	users, ok := response["users"]
	if !ok {
		t.Fatal("expected 'users' key in response")
	}

	if len(users) != 0 {
		t.Errorf("expected 0 users, got %d", len(users))
	}

	// Test non-existent room
	req = httptest.NewRequest("GET", "/rooms/999/users", nil)
	req = mux.SetURLVars(req, map[string]string{"roomID": "999"})
	w = httptest.NewRecorder()

	h.getRoomUsersHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	// Test invalid room ID
	req = httptest.NewRequest("GET", "/rooms/invalid/users", nil)
	req = mux.SetURLVars(req, map[string]string{"roomID": "invalid"})
	w = httptest.NewRecorder()

	h.getRoomUsersHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGetAllUsers(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(*Handler)
		expectedCount int
	}{
		{
			name: "Get all users with multiple users",
			setupFunc: func(h *Handler) {
				h.userRegistry.CreateUser("John", "Doe", "johndoe", model.AdditionalInfo{"role": "admin"})
				h.userRegistry.CreateUser("Jane", "Smith", "janesmith", nil)
				h.userRegistry.CreateUser("Bob", "Johnson", "bobjohnson", model.AdditionalInfo{"email": "bob@example.com"})
			},
			expectedCount: 3,
		},
		{
			name:          "Get all users with empty registry",
			setupFunc:     func(h *Handler) {},
			expectedCount: 0,
		},
		{
			name: "Get all users with single user",
			setupFunc: func(h *Handler) {
				h.userRegistry.CreateUser("Alice", "Wonder", "alice", nil)
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := setupHandler(t)
			tt.setupFunc(h)

			req := httptest.NewRequest("GET", "/users", nil)
			w := httptest.NewRecorder()

			h.getAllUsersHandler(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
			}

			var users []*model.User
			if err := json.NewDecoder(w.Body).Decode(&users); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if len(users) != tt.expectedCount {
				t.Errorf("expected %d users, got %d", tt.expectedCount, len(users))
			}

			for _, u := range users {
				if u.ID == uuid.Nil {
					t.Error("expected user ID to be set")
				}
			}
		})
	}
}

func TestGetAllUsersInRooms(t *testing.T) {
	h := setupHandler(t)

	// Create rooms - they'll be empty since we can't add clients from handler level
	room1 := h.hub.CreateRoom(nil)
	close(room1.Shutdown())

	req := httptest.NewRequest("GET", "/rooms/users", nil)
	w := httptest.NewRecorder()

	h.getAllUsersInRoomsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string][]model.UserWithRoom
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	_, ok := response["users"]
	if !ok {
		t.Fatal("expected 'users' key in response")
	}
}
