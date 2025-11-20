package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func setupRoomTests() {
	// Reset hub and room counter for tests
	hub = &Hub{
		rooms: make(map[uint]*Room),
	}
	roomCounter = 0
}

func TestCreateRoom(t *testing.T) {
	setupRoomTests()

	tests := []struct {
		name           string
		payload        AdditionalInfo
		expectedStatus int
	}{
		{
			name: "Create room with additionalInfo",
			payload: AdditionalInfo{
				"name":        "Test Room",
				"description": "A test room",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Create room without additionalInfo",
			payload:        AdditionalInfo{},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("POST", "/rooms", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			createRoomHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if w.Code == http.StatusOK {
				var response map[string]uint
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if response["roomID"] == 0 {
					t.Error("expected room ID to be set")
				}
			}
		})
	}
}

func TestCreateRoomInvalidJSON(t *testing.T) {
	setupRoomTests()

	req := httptest.NewRequest("POST", "/rooms", bytes.NewBufferString("invalid json"))
	w := httptest.NewRecorder()

	createRoomHandler(w, req)

	// Handler accepts invalid JSON and creates room with empty additionalInfo
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]uint
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["roomID"] == 0 {
		t.Error("expected room ID to be set even with invalid JSON")
	}
}

func TestGetAllRooms(t *testing.T) {
	setupRoomTests()

	// Create some rooms
	room1 := hub.CreateRoom(AdditionalInfo{"name": "Room 1"})
	room2 := hub.CreateRoom(AdditionalInfo{"name": "Room 2"})

	// Stop the rooms to prevent goroutine issues in tests
	close(room1.shutdown)
	close(room2.shutdown)

	req := httptest.NewRequest("GET", "/rooms", nil)
	w := httptest.NewRecorder()

	getAllRoomsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string][]RoomResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	rooms, ok := response["rooms"]
	if !ok {
		t.Fatal("expected 'rooms' key in response")
	}

	if len(rooms) != 2 {
		t.Errorf("expected 2 rooms, got %d", len(rooms))
	}

	// Verify rooms are sorted by ID
	if len(rooms) >= 2 && rooms[0].ID > rooms[1].ID {
		t.Error("expected rooms to be sorted by ID")
	}
}

func TestGetRoomByID(t *testing.T) {
	setupRoomTests()

	room := hub.CreateRoom(AdditionalInfo{"name": "Test Room"})
	close(room.shutdown) // Stop the room goroutine

	tests := []struct {
		name           string
		roomID         string
		expectedStatus int
	}{
		{
			name:           "Get existing room",
			roomID:         "1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Get non-existent room",
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
			req := httptest.NewRequest("GET", "/rooms/"+tt.roomID, nil)
			req = mux.SetURLVars(req, map[string]string{"roomID": tt.roomID})
			w := httptest.NewRecorder()

			getRoomIDHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if w.Code == http.StatusOK {
				var response RoomResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if response.ID == 0 {
					t.Error("expected room ID to be set")
				}
			}
		})
	}
}

func TestPatchRoom(t *testing.T) {
	setupRoomTests()

	room := hub.CreateRoom(AdditionalInfo{"name": "Original Name", "description": "Original"})
	close(room.shutdown) // Stop the room goroutine

	tests := []struct {
		name           string
		roomID         string
		payload        AdditionalInfo
		expectedStatus int
		checkFunc      func(*testing.T, *Room)
	}{
		{
			name:   "Patch room additionalInfo",
			roomID: "1",
			payload: AdditionalInfo{
				"name": "Updated Name",
			},
			expectedStatus: http.StatusOK,
			checkFunc: func(t *testing.T, r *Room) {
				info := r.GetAdditionalInfo()
				if info["name"] != "Updated Name" {
					t.Errorf("expected name to be updated to 'Updated Name', got %v", info["name"])
				}
			},
		},
		{
			name:           "Patch non-existent room",
			roomID:         "999",
			payload:        AdditionalInfo{"name": "Test"},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid room ID",
			roomID:         "invalid",
			payload:        AdditionalInfo{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("PATCH", "/rooms/"+tt.roomID, bytes.NewBuffer(body))
			req = mux.SetURLVars(req, map[string]string{"roomID": tt.roomID})
			w := httptest.NewRecorder()

			patchRoomHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if w.Code == http.StatusOK && tt.checkFunc != nil {
				room, ok := hub.GetRoom(1)
				if !ok {
					t.Fatal("room not found after patch")
				}
				tt.checkFunc(t, room)
			}
		})
	}
}

func TestPutRoom(t *testing.T) {
	setupRoomTests()

	room := hub.CreateRoom(AdditionalInfo{"name": "Original Name", "description": "Original"})
	close(room.shutdown) // Stop the room goroutine

	tests := []struct {
		name           string
		roomID         string
		payload        AdditionalInfo
		expectedStatus int
		checkFunc      func(*testing.T, *Room)
	}{
		{
			name:   "Put room with new additionalInfo",
			roomID: "1",
			payload: AdditionalInfo{
				"name": "Completely New Name",
			},
			expectedStatus: http.StatusOK,
			checkFunc: func(t *testing.T, r *Room) {
				info := r.GetAdditionalInfo()
				if info["name"] != "Completely New Name" {
					t.Errorf("expected name to be 'Completely New Name', got %v", info["name"])
				}
				// PUT should replace, so old fields should be gone
				if _, exists := info["description"]; exists {
					t.Error("expected old 'description' field to be removed after PUT")
				}
			},
		},
		{
			name:           "Put non-existent room",
			roomID:         "999",
			payload:        AdditionalInfo{"name": "Test"},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid room ID",
			roomID:         "invalid",
			payload:        AdditionalInfo{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("PUT", "/rooms/"+tt.roomID, bytes.NewBuffer(body))
			req = mux.SetURLVars(req, map[string]string{"roomID": tt.roomID})
			w := httptest.NewRecorder()

			putRoomHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if w.Code == http.StatusOK && tt.checkFunc != nil {
				room, ok := hub.GetRoom(1)
				if !ok {
					t.Fatal("room not found after put")
				}
				tt.checkFunc(t, room)
			}
		})
	}
}

func TestRoomGetUsers(t *testing.T) {
	setupRoomTests()

	room := &Room{
		id:      1,
		clients: make(map[*Client]bool),
	}

	// Add test clients
	client1 := &Client{user: User{FirstName: "John", LastName: "Doe"}}
	client2 := &Client{user: User{FirstName: "Jane", LastName: "Smith"}}

	room.clients[client1] = true
	room.clients[client2] = true

	users := room.GetUsers()

	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestRoomGetClientCount(t *testing.T) {
	setupRoomTests()

	room := &Room{
		id:      1,
		clients: make(map[*Client]bool),
	}

	if count := room.GetClientCount(); count != 0 {
		t.Errorf("expected 0 clients, got %d", count)
	}

	room.clients[&Client{}] = true
	room.clients[&Client{}] = true

	if count := room.GetClientCount(); count != 2 {
		t.Errorf("expected 2 clients, got %d", count)
	}
}

func TestHubGetAllUsersWithRooms(t *testing.T) {
	setupRoomTests()

	room1 := &Room{
		id:      1,
		clients: make(map[*Client]bool),
	}
	room2 := &Room{
		id:      2,
		clients: make(map[*Client]bool),
	}

	user1 := User{FirstName: "John"}
	user2 := User{FirstName: "Jane"}

	room1.clients[&Client{user: user1}] = true
	room2.clients[&Client{user: user2}] = true

	hub.rooms[1] = room1
	hub.rooms[2] = room2

	usersWithRooms := hub.GetAllUsersWithRooms()

	if len(usersWithRooms) != 2 {
		t.Errorf("expected 2 users with rooms, got %d", len(usersWithRooms))
	}

	for _, uwr := range usersWithRooms {
		if uwr.RoomID == 0 {
			t.Error("expected roomID to be set")
		}
	}
}
