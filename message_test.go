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

func setupMessageTests() *Room {
	// Reset hub and room counter for tests
	hub = &Hub{
		rooms: make(map[uint]*Room),
	}
	roomCounter = 0

	// Create a test room
	room := hub.CreateRoom(AdditionalInfo{"name": "Test Room"})
	close(room.shutdown) // Stop the room goroutine to prevent interference
	return room
}

func TestPatchMessage_OnlyMessage(t *testing.T) {
	room := setupMessageTests()

	// Create a test message
	originalMsg := OutgoingMessage{
		ID:          uuid.New(),
		MessageType: UserMessage,
		Message:     "Original message",
		User:        User{ID: uuid.New(), Name: "Alice"},
		AdditionalInfo: AdditionalInfo{
			"replyTo": "msg-123",
		},
	}
	room.StoreMessage(originalMsg)

	// Patch only the message content
	newContent := "Updated message"
	success := room.PatchMessage(originalMsg.ID, &newContent, nil)

	if !success {
		t.Fatal("expected PatchMessage to return true")
	}

	// Verify the message was updated
	updatedMsg, ok := room.GetMessage(originalMsg.ID)
	if !ok {
		t.Fatal("message not found after patch")
	}

	if updatedMsg.Message != newContent {
		t.Errorf("expected message to be '%s', got '%s'", newContent, updatedMsg.Message)
	}

	// Verify additionalInfo was preserved
	if updatedMsg.AdditionalInfo["replyTo"] != "msg-123" {
		t.Error("expected additionalInfo to be preserved")
	}
}

func TestPatchMessage_OnlyAdditionalInfo(t *testing.T) {
	room := setupMessageTests()

	// Create a test message
	originalMsg := OutgoingMessage{
		ID:          uuid.New(),
		MessageType: UserMessage,
		Message:     "Original message",
		User:        User{ID: uuid.New(), Name: "Alice"},
		AdditionalInfo: AdditionalInfo{
			"replyTo": "msg-123",
		},
	}
	room.StoreMessage(originalMsg)

	// Patch only the additionalInfo
	newInfo := AdditionalInfo{
		"edited":       true,
		"editedAt":     "2024-01-01T00:00:00Z",
		"editedReason": "Fixed typo",
	}
	success := room.PatchMessage(originalMsg.ID, nil, newInfo)

	if !success {
		t.Fatal("expected PatchMessage to return true")
	}

	// Verify the additionalInfo was updated
	updatedMsg, ok := room.GetMessage(originalMsg.ID)
	if !ok {
		t.Fatal("message not found after patch")
	}

	if updatedMsg.Message != originalMsg.Message {
		t.Error("expected message content to remain unchanged")
	}

	if updatedMsg.AdditionalInfo["edited"] != true {
		t.Error("expected additionalInfo to be updated")
	}

	if updatedMsg.AdditionalInfo["editedReason"] != "Fixed typo" {
		t.Error("expected additionalInfo editedReason to be set")
	}

	// Verify old additionalInfo was completely replaced
	if _, exists := updatedMsg.AdditionalInfo["replyTo"]; exists {
		t.Error("expected old additionalInfo to be replaced, not merged")
	}
}

func TestPatchMessage_BothFields(t *testing.T) {
	room := setupMessageTests()

	// Create a test message
	originalMsg := OutgoingMessage{
		ID:          uuid.New(),
		MessageType: UserMessage,
		Message:     "Original message",
		User:        User{ID: uuid.New(), Name: "Alice"},
		AdditionalInfo: AdditionalInfo{
			"replyTo": "msg-123",
		},
	}
	room.StoreMessage(originalMsg)

	// Patch both message and additionalInfo
	newContent := "Updated message"
	newInfo := AdditionalInfo{
		"edited":   true,
		"editedAt": "2024-01-01T00:00:00Z",
	}
	success := room.PatchMessage(originalMsg.ID, &newContent, newInfo)

	if !success {
		t.Fatal("expected PatchMessage to return true")
	}

	// Verify both fields were updated
	updatedMsg, ok := room.GetMessage(originalMsg.ID)
	if !ok {
		t.Fatal("message not found after patch")
	}

	if updatedMsg.Message != newContent {
		t.Errorf("expected message to be '%s', got '%s'", newContent, updatedMsg.Message)
	}

	if updatedMsg.AdditionalInfo["edited"] != true {
		t.Error("expected additionalInfo to be updated")
	}
}

func TestPatchMessage_NonExistentMessage(t *testing.T) {
	room := setupMessageTests()

	// Try to patch a message that doesn't exist
	nonExistentID := uuid.New()
	newContent := "Updated message"
	success := room.PatchMessage(nonExistentID, &newContent, nil)

	if success {
		t.Error("expected PatchMessage to return false for non-existent message")
	}
}

func TestPatchRoomMessageHandler_OnlyMessage(t *testing.T) {
	room := setupMessageTests()

	// Create a test message
	testMsg := OutgoingMessage{
		ID:          uuid.New(),
		MessageType: UserMessage,
		Message:     "Original message",
		User:        User{ID: uuid.New(), Name: "Alice"},
		AdditionalInfo: AdditionalInfo{
			"replyTo": "msg-123",
		},
	}
	room.StoreMessage(testMsg)

	// Prepare request to patch only the message
	patchPayload := map[string]interface{}{
		"message": "Updated via handler",
	}
	body, _ := json.Marshal(patchPayload)

	req := httptest.NewRequest("PATCH", "/rooms/1/messages/"+testMsg.ID.String(), bytes.NewBuffer(body))
	req = mux.SetURLVars(req, map[string]string{
		"roomID":    "1",
		"messageID": testMsg.ID.String(),
	})
	w := httptest.NewRecorder()

	patchRoomMessageHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response OutgoingMessage
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Message != "Updated via handler" {
		t.Errorf("expected message to be 'Updated via handler', got '%s'", response.Message)
	}

	// Verify additionalInfo was preserved
	if response.AdditionalInfo["replyTo"] != "msg-123" {
		t.Error("expected additionalInfo to be preserved")
	}
}

func TestPatchRoomMessageHandler_OnlyAdditionalInfo(t *testing.T) {
	room := setupMessageTests()

	// Create a test message
	testMsg := OutgoingMessage{
		ID:          uuid.New(),
		MessageType: UserMessage,
		Message:     "Original message",
		User:        User{ID: uuid.New(), Name: "Alice"},
	}
	room.StoreMessage(testMsg)

	// Prepare request to patch only the additionalInfo
	patchPayload := map[string]interface{}{
		"additionalInfo": map[string]interface{}{
			"edited":   true,
			"editedAt": "2024-01-01T00:00:00Z",
		},
	}
	body, _ := json.Marshal(patchPayload)

	req := httptest.NewRequest("PATCH", "/rooms/1/messages/"+testMsg.ID.String(), bytes.NewBuffer(body))
	req = mux.SetURLVars(req, map[string]string{
		"roomID":    "1",
		"messageID": testMsg.ID.String(),
	})
	w := httptest.NewRecorder()

	patchRoomMessageHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response OutgoingMessage
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Message != "Original message" {
		t.Error("expected original message to be preserved")
	}

	if response.AdditionalInfo["edited"] != true {
		t.Error("expected additionalInfo to be updated")
	}
}

func TestPatchRoomMessageHandler_BothFields(t *testing.T) {
	room := setupMessageTests()

	// Create a test message
	testMsg := OutgoingMessage{
		ID:          uuid.New(),
		MessageType: UserMessage,
		Message:     "Original message",
		User:        User{ID: uuid.New(), Name: "Alice"},
	}
	room.StoreMessage(testMsg)

	// Prepare request to patch both fields
	patchPayload := map[string]interface{}{
		"message": "Updated message and info",
		"additionalInfo": map[string]interface{}{
			"edited":   true,
			"editedAt": "2024-01-01T00:00:00Z",
		},
	}
	body, _ := json.Marshal(patchPayload)

	req := httptest.NewRequest("PATCH", "/rooms/1/messages/"+testMsg.ID.String(), bytes.NewBuffer(body))
	req = mux.SetURLVars(req, map[string]string{
		"roomID":    "1",
		"messageID": testMsg.ID.String(),
	})
	w := httptest.NewRecorder()

	patchRoomMessageHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response OutgoingMessage
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Message != "Updated message and info" {
		t.Errorf("expected message to be updated")
	}

	if response.AdditionalInfo["edited"] != true {
		t.Error("expected additionalInfo to be updated")
	}
}

func TestPatchRoomMessageHandler_NoFieldsProvided(t *testing.T) {
	room := setupMessageTests()

	// Create a test message
	testMsg := OutgoingMessage{
		ID:          uuid.New(),
		MessageType: UserMessage,
		Message:     "Original message",
		User:        User{ID: uuid.New(), Name: "Alice"},
	}
	room.StoreMessage(testMsg)

	// Prepare request with empty payload
	patchPayload := map[string]interface{}{}
	body, _ := json.Marshal(patchPayload)

	req := httptest.NewRequest("PATCH", "/rooms/1/messages/"+testMsg.ID.String(), bytes.NewBuffer(body))
	req = mux.SetURLVars(req, map[string]string{
		"roomID":    "1",
		"messageID": testMsg.ID.String(),
	})
	w := httptest.NewRecorder()

	patchRoomMessageHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestPatchRoomMessageHandler_EmptyMessage(t *testing.T) {
	room := setupMessageTests()

	// Create a test message
	testMsg := OutgoingMessage{
		ID:          uuid.New(),
		MessageType: UserMessage,
		Message:     "Original message",
		User:        User{ID: uuid.New(), Name: "Alice"},
	}
	room.StoreMessage(testMsg)

	// Prepare request with empty message string
	patchPayload := map[string]interface{}{
		"message": "",
	}
	body, _ := json.Marshal(patchPayload)

	req := httptest.NewRequest("PATCH", "/rooms/1/messages/"+testMsg.ID.String(), bytes.NewBuffer(body))
	req = mux.SetURLVars(req, map[string]string{
		"roomID":    "1",
		"messageID": testMsg.ID.String(),
	})
	w := httptest.NewRecorder()

	patchRoomMessageHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestPatchRoomMessageHandler_InvalidRoomID(t *testing.T) {
	setupMessageTests()

	req := httptest.NewRequest("PATCH", "/rooms/invalid/messages/"+uuid.New().String(), bytes.NewBufferString("{}"))
	req = mux.SetURLVars(req, map[string]string{
		"roomID":    "invalid",
		"messageID": uuid.New().String(),
	})
	w := httptest.NewRecorder()

	patchRoomMessageHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestPatchRoomMessageHandler_InvalidMessageID(t *testing.T) {
	setupMessageTests()

	req := httptest.NewRequest("PATCH", "/rooms/1/messages/invalid", bytes.NewBufferString("{}"))
	req = mux.SetURLVars(req, map[string]string{
		"roomID":    "1",
		"messageID": "invalid",
	})
	w := httptest.NewRecorder()

	patchRoomMessageHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestPatchRoomMessageHandler_RoomNotFound(t *testing.T) {
	setupMessageTests()

	messageID := uuid.New()
	patchPayload := map[string]interface{}{
		"message": "Updated message",
	}
	body, _ := json.Marshal(patchPayload)

	req := httptest.NewRequest("PATCH", "/rooms/999/messages/"+messageID.String(), bytes.NewBuffer(body))
	req = mux.SetURLVars(req, map[string]string{
		"roomID":    "999",
		"messageID": messageID.String(),
	})
	w := httptest.NewRecorder()

	patchRoomMessageHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestPatchRoomMessageHandler_MessageNotFound(t *testing.T) {
	setupMessageTests()

	nonExistentMsgID := uuid.New()
	patchPayload := map[string]interface{}{
		"message": "Updated message",
	}
	body, _ := json.Marshal(patchPayload)

	req := httptest.NewRequest("PATCH", "/rooms/1/messages/"+nonExistentMsgID.String(), bytes.NewBuffer(body))
	req = mux.SetURLVars(req, map[string]string{
		"roomID":    "1",
		"messageID": nonExistentMsgID.String(),
	})
	w := httptest.NewRecorder()

	patchRoomMessageHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestGetRoomMessages(t *testing.T) {
	room := setupMessageTests()

	// Add some test messages
	msg1 := OutgoingMessage{
		ID:          uuid.New(),
		MessageType: UserMessage,
		Message:     "First message",
		User:        User{ID: uuid.New(), Name: "Alice"},
	}
	msg2 := OutgoingMessage{
		ID:          uuid.New(),
		MessageType: UserMessage,
		Message:     "Second message",
		User:        User{ID: uuid.New(), Name: "Bob"},
	}
	room.StoreMessage(msg1)
	room.StoreMessage(msg2)

	req := httptest.NewRequest("GET", "/rooms/1/messages", nil)
	req = mux.SetURLVars(req, map[string]string{"roomID": "1"})
	w := httptest.NewRecorder()

	getRoomMessagesHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string][]OutgoingMessage
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	messages, ok := response["messages"]
	if !ok {
		t.Fatal("expected 'messages' key in response")
	}

	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(messages))
	}
}

func TestGetRoomMessages_RoomNotFound(t *testing.T) {
	setupMessageTests()

	req := httptest.NewRequest("GET", "/rooms/999/messages", nil)
	req = mux.SetURLVars(req, map[string]string{"roomID": "999"})
	w := httptest.NewRecorder()

	getRoomMessagesHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestShouldStoreMessage(t *testing.T) {
	tests := []struct {
		name     string
		msgType  MessageType
		expected bool
	}{
		{
			name:     "Store system messages",
			msgType:  SystemMessage,
			expected: true,
		},
		{
			name:     "Store user messages",
			msgType:  UserMessage,
			expected: true,
		},
		{
			name:     "Do not store image messages",
			msgType:  ImageMessage,
			expected: false,
		},
		{
			name:     "Do not store user_typing events",
			msgType:  "user_typing",
			expected: false,
		},
		{
			name:     "Do not store message_updated events",
			msgType:  "message_updated",
			expected: false,
		},
		{
			name:     "Do not store custom events",
			msgType:  "custom_event",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldStoreMessage(tt.msgType)
			if result != tt.expected {
				t.Errorf("shouldStoreMessage(%s) = %v, expected %v", tt.msgType, result, tt.expected)
			}
		})
	}
}

func TestGetRoomMessageHandler_Success(t *testing.T) {
	room := setupMessageTests()

	testMsg := OutgoingMessage{
		ID:          uuid.New(),
		MessageType: UserMessage,
		Message:     "Test message",
		User:        User{ID: uuid.New(), Name: "Alice"},
		AdditionalInfo: AdditionalInfo{
			"key": "value",
		},
	}
	room.StoreMessage(testMsg)

	req := httptest.NewRequest("GET", "/rooms/1/messages/"+testMsg.ID.String(), nil)
	req = mux.SetURLVars(req, map[string]string{
		"roomID":    "1",
		"messageID": testMsg.ID.String(),
	})
	w := httptest.NewRecorder()

	getRoomMessageHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response OutgoingMessage
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.ID != testMsg.ID {
		t.Errorf("expected message ID %s, got %s", testMsg.ID, response.ID)
	}

	if response.Message != "Test message" {
		t.Errorf("expected message 'Test message', got '%s'", response.Message)
	}
}

func TestGetRoomMessageHandler_InvalidRoomID(t *testing.T) {
	setupMessageTests()

	req := httptest.NewRequest("GET", "/rooms/invalid/messages/"+uuid.New().String(), nil)
	req = mux.SetURLVars(req, map[string]string{
		"roomID":    "invalid",
		"messageID": uuid.New().String(),
	})
	w := httptest.NewRecorder()

	getRoomMessageHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGetRoomMessageHandler_InvalidMessageID(t *testing.T) {
	setupMessageTests()

	req := httptest.NewRequest("GET", "/rooms/1/messages/invalid", nil)
	req = mux.SetURLVars(req, map[string]string{
		"roomID":    "1",
		"messageID": "invalid",
	})
	w := httptest.NewRecorder()

	getRoomMessageHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGetRoomMessageHandler_RoomNotFound(t *testing.T) {
	setupMessageTests()

	req := httptest.NewRequest("GET", "/rooms/999/messages/"+uuid.New().String(), nil)
	req = mux.SetURLVars(req, map[string]string{
		"roomID":    "999",
		"messageID": uuid.New().String(),
	})
	w := httptest.NewRecorder()

	getRoomMessageHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestGetRoomMessageHandler_MessageNotFound(t *testing.T) {
	setupMessageTests()

	nonExistentID := uuid.New()
	req := httptest.NewRequest("GET", "/rooms/1/messages/"+nonExistentID.String(), nil)
	req = mux.SetURLVars(req, map[string]string{
		"roomID":    "1",
		"messageID": nonExistentID.String(),
	})
	w := httptest.NewRecorder()

	getRoomMessageHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestPutRoomMessageHandler_Success(t *testing.T) {
	room := setupMessageTests()

	testMsg := OutgoingMessage{
		ID:          uuid.New(),
		MessageType: UserMessage,
		Message:     "Original message",
		User:        User{ID: uuid.New(), Name: "Alice"},
		AdditionalInfo: AdditionalInfo{
			"replyTo": "msg-123",
		},
	}
	room.StoreMessage(testMsg)

	putPayload := map[string]interface{}{
		"message": "Completely replaced message",
		"additionalInfo": map[string]interface{}{
			"edited":   true,
			"editedAt": "2024-01-01T00:00:00Z",
		},
	}
	body, _ := json.Marshal(putPayload)

	req := httptest.NewRequest("PUT", "/rooms/1/messages/"+testMsg.ID.String(), bytes.NewBuffer(body))
	req = mux.SetURLVars(req, map[string]string{
		"roomID":    "1",
		"messageID": testMsg.ID.String(),
	})
	w := httptest.NewRecorder()

	putRoomMessageHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response OutgoingMessage
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Message != "Completely replaced message" {
		t.Errorf("expected message 'Completely replaced message', got '%s'", response.Message)
	}

	if response.AdditionalInfo["edited"] != true {
		t.Error("expected additionalInfo to be updated")
	}

	if _, exists := response.AdditionalInfo["replyTo"]; exists {
		t.Error("expected old additionalInfo to be completely replaced")
	}
}

func TestPutRoomMessageHandler_InvalidRoomID(t *testing.T) {
	setupMessageTests()

	putPayload := map[string]interface{}{
		"message": "Test",
	}
	body, _ := json.Marshal(putPayload)

	req := httptest.NewRequest("PUT", "/rooms/invalid/messages/"+uuid.New().String(), bytes.NewBuffer(body))
	req = mux.SetURLVars(req, map[string]string{
		"roomID":    "invalid",
		"messageID": uuid.New().String(),
	})
	w := httptest.NewRecorder()

	putRoomMessageHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestPutRoomMessageHandler_InvalidMessageID(t *testing.T) {
	setupMessageTests()

	putPayload := map[string]interface{}{
		"message": "Test",
	}
	body, _ := json.Marshal(putPayload)

	req := httptest.NewRequest("PUT", "/rooms/1/messages/invalid", bytes.NewBuffer(body))
	req = mux.SetURLVars(req, map[string]string{
		"roomID":    "1",
		"messageID": "invalid",
	})
	w := httptest.NewRecorder()

	putRoomMessageHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestPutRoomMessageHandler_RoomNotFound(t *testing.T) {
	setupMessageTests()

	putPayload := map[string]interface{}{
		"message": "Test",
	}
	body, _ := json.Marshal(putPayload)

	req := httptest.NewRequest("PUT", "/rooms/999/messages/"+uuid.New().String(), bytes.NewBuffer(body))
	req = mux.SetURLVars(req, map[string]string{
		"roomID":    "999",
		"messageID": uuid.New().String(),
	})
	w := httptest.NewRecorder()

	putRoomMessageHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestPutRoomMessageHandler_MessageNotFound(t *testing.T) {
	setupMessageTests()

	nonExistentID := uuid.New()
	putPayload := map[string]interface{}{
		"message": "Test",
	}
	body, _ := json.Marshal(putPayload)

	req := httptest.NewRequest("PUT", "/rooms/1/messages/"+nonExistentID.String(), bytes.NewBuffer(body))
	req = mux.SetURLVars(req, map[string]string{
		"roomID":    "1",
		"messageID": nonExistentID.String(),
	})
	w := httptest.NewRecorder()

	putRoomMessageHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestPutRoomMessageHandler_InvalidJSON(t *testing.T) {
	setupMessageTests()

	req := httptest.NewRequest("PUT", "/rooms/1/messages/"+uuid.New().String(), bytes.NewBufferString("{invalid json"))
	req = mux.SetURLVars(req, map[string]string{
		"roomID":    "1",
		"messageID": uuid.New().String(),
	})
	w := httptest.NewRecorder()

	putRoomMessageHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestDeleteRoomMessageHandler_Success(t *testing.T) {
	room := setupMessageTests()

	testMsg := OutgoingMessage{
		ID:          uuid.New(),
		MessageType: UserMessage,
		Message:     "Message to delete",
		User:        User{ID: uuid.New(), Name: "Alice"},
	}
	room.StoreMessage(testMsg)

	req := httptest.NewRequest("DELETE", "/rooms/1/messages/"+testMsg.ID.String(), nil)
	req = mux.SetURLVars(req, map[string]string{
		"roomID":    "1",
		"messageID": testMsg.ID.String(),
	})
	w := httptest.NewRecorder()

	deleteRoomMessageHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response OutgoingMessage
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Message != "deleted" {
		t.Errorf("expected message 'deleted', got '%s'", response.Message)
	}

	if response.AdditionalInfo["deleted"] != true {
		t.Error("expected additionalInfo to contain 'deleted: true'")
	}
}

func TestDeleteRoomMessageHandler_InvalidRoomID(t *testing.T) {
	setupMessageTests()

	req := httptest.NewRequest("DELETE", "/rooms/invalid/messages/"+uuid.New().String(), nil)
	req = mux.SetURLVars(req, map[string]string{
		"roomID":    "invalid",
		"messageID": uuid.New().String(),
	})
	w := httptest.NewRecorder()

	deleteRoomMessageHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestDeleteRoomMessageHandler_InvalidMessageID(t *testing.T) {
	setupMessageTests()

	req := httptest.NewRequest("DELETE", "/rooms/1/messages/invalid", nil)
	req = mux.SetURLVars(req, map[string]string{
		"roomID":    "1",
		"messageID": "invalid",
	})
	w := httptest.NewRecorder()

	deleteRoomMessageHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestDeleteRoomMessageHandler_RoomNotFound(t *testing.T) {
	setupMessageTests()

	req := httptest.NewRequest("DELETE", "/rooms/999/messages/"+uuid.New().String(), nil)
	req = mux.SetURLVars(req, map[string]string{
		"roomID":    "999",
		"messageID": uuid.New().String(),
	})
	w := httptest.NewRecorder()

	deleteRoomMessageHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestDeleteRoomMessageHandler_MessageNotFound(t *testing.T) {
	setupMessageTests()

	nonExistentID := uuid.New()
	req := httptest.NewRequest("DELETE", "/rooms/1/messages/"+nonExistentID.String(), nil)
	req = mux.SetURLVars(req, map[string]string{
		"roomID":    "1",
		"messageID": nonExistentID.String(),
	})
	w := httptest.NewRecorder()

	deleteRoomMessageHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}
