package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func setupUserTests() {
	// Reset user registry for tests
	userRegistry = &UserRegistry{
		users: make(map[uuid.UUID]*User),
	}
}

func TestCreateUser(t *testing.T) {
	setupUserTests()

	tests := []struct {
		name           string
		payload        CreateUserRequest
		expectedStatus int
	}{
		{
			name: "Create user with all fields",
			payload: CreateUserRequest{
				FirstName: "John",
				LastName:  "Doe",
				Name:      "johndoe",
				AdditionalInfo: AdditionalInfo{
					"email": "john@example.com",
				},
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Create user with minimal fields",
			payload: CreateUserRequest{
				FirstName: "Jane",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "Create user with empty payload",
			payload:        CreateUserRequest{},
			expectedStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("POST", "/users", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			createUserHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if w.Code == http.StatusCreated {
				var user User
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
	setupUserTests()

	req := httptest.NewRequest("POST", "/users", bytes.NewBufferString("invalid json"))
	w := httptest.NewRecorder()

	createUserHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestPutUser(t *testing.T) {
	setupUserTests()

	// Create a user first
	user := userRegistry.CreateUser("John", "Doe", "johndoe", nil)

	tests := []struct {
		name           string
		userID         string
		payload        UpdateUserRequest
		expectedStatus int
	}{
		{
			name:   "Update existing user",
			userID: user.ID.String(),
			payload: UpdateUserRequest{
				FirstName: "Jane",
				LastName:  "Smith",
				Name:      "janesmith",
				AdditionalInfo: AdditionalInfo{
					"email": "jane@example.com",
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Update non-existent user",
			userID: uuid.New().String(),
			payload: UpdateUserRequest{
				FirstName: "Test",
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid user ID",
			userID:         "invalid-uuid",
			payload:        UpdateUserRequest{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("PUT", "/users/"+tt.userID, bytes.NewBuffer(body))
			req = mux.SetURLVars(req, map[string]string{"userID": tt.userID})
			w := httptest.NewRecorder()

			putUserHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if w.Code == http.StatusOK {
				var updatedUser User
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
	setupUserTests()

	// Create a user first
	user := userRegistry.CreateUser("John", "Doe", "johndoe", AdditionalInfo{"role": "user"})

	tests := []struct {
		name           string
		userID         string
		payload        map[string]any
		expectedStatus int
		checkFunc      func(*testing.T, User)
	}{
		{
			name:   "Patch firstName only",
			userID: user.ID.String(),
			payload: map[string]any{
				"firstName": "Jane",
			},
			expectedStatus: http.StatusOK,
			checkFunc: func(t *testing.T, u User) {
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
			userID: user.ID.String(),
			payload: map[string]any{
				"additionalInfo": map[string]any{
					"email": "test@example.com",
				},
			},
			expectedStatus: http.StatusOK,
			checkFunc: func(t *testing.T, u User) {
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

			patchUserHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if w.Code == http.StatusOK && tt.checkFunc != nil {
				var patchedUser User
				if err := json.NewDecoder(w.Body).Decode(&patchedUser); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				tt.checkFunc(t, patchedUser)
			}
		})
	}
}

func TestDeleteUser(t *testing.T) {
	setupUserTests()

	// Create a user first
	user := userRegistry.CreateUser("John", "Doe", "johndoe", nil)

	tests := []struct {
		name           string
		userID         string
		expectedStatus int
	}{
		{
			name:           "Delete existing user",
			userID:         user.ID.String(),
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

			deleteUserHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestGetRoomUsers(t *testing.T) {
	setupUserTests()

	// Setup hub and room
	hub = &Hub{
		rooms: make(map[uint]*Room),
	}

	room := &Room{
		id:      1,
		clients: make(map[*Client]bool),
	}

	// Add some test users to the room
	user1 := User{ID: uuid.New(), FirstName: "John", LastName: "Doe"}
	user2 := User{ID: uuid.New(), FirstName: "Jane", LastName: "Smith"}

	client1 := &Client{user: user1}
	client2 := &Client{user: user2}

	room.clients[client1] = true
	room.clients[client2] = true

	hub.rooms[1] = room

	tests := []struct {
		name           string
		roomID         string
		expectedStatus int
		expectedCount  int
	}{
		{
			name:           "Get users from existing room",
			roomID:         "1",
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:           "Get users from non-existent room",
			roomID:         "999",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid room ID",
			roomID:         "invalid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/rooms/"+tt.roomID+"/users", nil)
			req = mux.SetURLVars(req, map[string]string{"roomID": tt.roomID})
			w := httptest.NewRecorder()

			getRoomUsersHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if w.Code == http.StatusOK {
				var response map[string][]User
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				users, ok := response["users"]
				if !ok {
					t.Fatal("expected 'users' key in response")
				}

				if len(users) != tt.expectedCount {
					t.Errorf("expected %d users, got %d", tt.expectedCount, len(users))
				}
			}
		})
	}
}

func TestGetAllUsers(t *testing.T) {
	setupUserTests()

	tests := []struct {
		name          string
		setupFunc     func()
		expectedCount int
	}{
		{
			name: "Get all users with multiple users",
			setupFunc: func() {
				userRegistry.CreateUser("John", "Doe", "johndoe", AdditionalInfo{"role": "admin"})
				userRegistry.CreateUser("Jane", "Smith", "janesmith", nil)
				userRegistry.CreateUser("Bob", "Johnson", "bobjohnson", AdditionalInfo{"email": "bob@example.com"})
			},
			expectedCount: 3,
		},
		{
			name:          "Get all users with empty registry",
			setupFunc:     func() {},
			expectedCount: 0,
		},
		{
			name: "Get all users with single user",
			setupFunc: func() {
				userRegistry.CreateUser("Alice", "Wonder", "alice", nil)
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupUserTests()
			tt.setupFunc()

			req := httptest.NewRequest("GET", "/users", nil)
			w := httptest.NewRecorder()

			getAllUsersHandler(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
			}

			var users []*User
			if err := json.NewDecoder(w.Body).Decode(&users); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if len(users) != tt.expectedCount {
				t.Errorf("expected %d users, got %d", tt.expectedCount, len(users))
			}

			for _, user := range users {
				if user.ID == uuid.Nil {
					t.Error("expected user ID to be set")
				}
			}
		})
	}
}

func TestGetAllUsersInRooms(t *testing.T) {
	setupUserTests()

	// Setup hub with multiple rooms
	hub = &Hub{
		rooms: make(map[uint]*Room),
	}

	room1 := &Room{
		id:      1,
		clients: make(map[*Client]bool),
	}
	room2 := &Room{
		id:      2,
		clients: make(map[*Client]bool),
	}

	user1 := User{ID: uuid.New(), FirstName: "John"}
	user2 := User{ID: uuid.New(), FirstName: "Jane"}
	user3 := User{ID: uuid.New(), FirstName: "Bob"}

	room1.clients[&Client{user: user1}] = true
	room1.clients[&Client{user: user2}] = true
	room2.clients[&Client{user: user3}] = true

	hub.rooms[1] = room1
	hub.rooms[2] = room2

	req := httptest.NewRequest("GET", "/rooms/users", nil)
	w := httptest.NewRecorder()

	getAllUsersInRoomsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string][]UserWithRoom
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	users, ok := response["users"]
	if !ok {
		t.Fatal("expected 'users' key in response")
	}

	if len(users) != 3 {
		t.Errorf("expected 3 users total, got %d", len(users))
	}

	// Verify each user has a roomId
	for _, userWithRoom := range users {
		if userWithRoom.RoomID == 0 {
			t.Error("expected roomId to be set")
		}
		if userWithRoom.User.ID == uuid.Nil {
			t.Error("expected user ID to be set")
		}
	}
}
