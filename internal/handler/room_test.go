package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/choffmann/chat-room/internal/model"
	"github.com/gorilla/mux"
)

func TestCreateRoom(t *testing.T) {
	h := setupHandler(t)

	tests := []struct {
		name           string
		payload        model.AdditionalInfo
		expectedStatus int
	}{
		{
			name: "Create room with additionalInfo",
			payload: model.AdditionalInfo{
				"name":        "Test Room",
				"description": "A test room",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Create room without additionalInfo",
			payload:        model.AdditionalInfo{},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("POST", "/rooms", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			h.createRoomHandler(w, req)

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
	h := setupHandler(t)

	req := httptest.NewRequest("POST", "/rooms", bytes.NewBufferString("invalid json"))
	w := httptest.NewRecorder()

	h.createRoomHandler(w, req)

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
	h := setupHandler(t)

	room1 := h.hub.CreateRoom(model.AdditionalInfo{"name": "Room 1"})
	room2 := h.hub.CreateRoom(model.AdditionalInfo{"name": "Room 2"})

	close(room1.Shutdown())
	close(room2.Shutdown())

	req := httptest.NewRequest("GET", "/rooms", nil)
	w := httptest.NewRecorder()

	h.getAllRoomsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string][]model.RoomResponse
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

	if len(rooms) >= 2 && rooms[0].ID > rooms[1].ID {
		t.Error("expected rooms to be sorted by ID")
	}
}

func TestGetRoomByID(t *testing.T) {
	h := setupHandler(t)

	room := h.hub.CreateRoom(model.AdditionalInfo{"name": "Test Room"})
	close(room.Shutdown())

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

			h.getRoomIDHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if w.Code == http.StatusOK {
				var response model.RoomResponse
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
	h := setupHandler(t)

	room := h.hub.CreateRoom(model.AdditionalInfo{"name": "Original Name", "description": "Original"})
	close(room.Shutdown())

	tests := []struct {
		name           string
		roomID         string
		payload        model.AdditionalInfo
		expectedStatus int
		checkFunc      func(*testing.T)
	}{
		{
			name:   "Patch room additionalInfo",
			roomID: "1",
			payload: model.AdditionalInfo{
				"name": "Updated Name",
			},
			expectedStatus: http.StatusOK,
			checkFunc: func(t *testing.T) {
				room, _ := h.hub.GetRoom(1)
				info := room.GetAdditionalInfo()
				if info["name"] != "Updated Name" {
					t.Errorf("expected name to be updated to 'Updated Name', got %v", info["name"])
				}
			},
		},
		{
			name:           "Patch non-existent room",
			roomID:         "999",
			payload:        model.AdditionalInfo{"name": "Test"},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid room ID",
			roomID:         "invalid",
			payload:        model.AdditionalInfo{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("PATCH", "/rooms/"+tt.roomID, bytes.NewBuffer(body))
			req = mux.SetURLVars(req, map[string]string{"roomID": tt.roomID})
			w := httptest.NewRecorder()

			h.patchRoomHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if w.Code == http.StatusOK && tt.checkFunc != nil {
				tt.checkFunc(t)
			}
		})
	}
}

func TestPutRoom(t *testing.T) {
	h := setupHandler(t)

	room := h.hub.CreateRoom(model.AdditionalInfo{"name": "Original Name", "description": "Original"})
	close(room.Shutdown())

	tests := []struct {
		name           string
		roomID         string
		payload        model.AdditionalInfo
		expectedStatus int
		checkFunc      func(*testing.T)
	}{
		{
			name:   "Put room with new additionalInfo",
			roomID: "1",
			payload: model.AdditionalInfo{
				"name": "Completely New Name",
			},
			expectedStatus: http.StatusOK,
			checkFunc: func(t *testing.T) {
				room, _ := h.hub.GetRoom(1)
				info := room.GetAdditionalInfo()
				if info["name"] != "Completely New Name" {
					t.Errorf("expected name to be 'Completely New Name', got %v", info["name"])
				}
				if _, exists := info["description"]; exists {
					t.Error("expected old 'description' field to be removed after PUT")
				}
			},
		},
		{
			name:           "Put non-existent room",
			roomID:         "999",
			payload:        model.AdditionalInfo{"name": "Test"},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid room ID",
			roomID:         "invalid",
			payload:        model.AdditionalInfo{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("PUT", "/rooms/"+tt.roomID, bytes.NewBuffer(body))
			req = mux.SetURLVars(req, map[string]string{"roomID": tt.roomID})
			w := httptest.NewRecorder()

			h.putRoomHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if w.Code == http.StatusOK && tt.checkFunc != nil {
				tt.checkFunc(t)
			}
		})
	}
}
