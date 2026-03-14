package chat

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/choffmann/chat-room/internal/model"
	"github.com/google/uuid"
)

type mockUploadStore struct {
	savedData []byte
	savedRoom uint
	relPath   string
	err       error
}

func (m *mockUploadStore) Save(roomID uint, data []byte) (string, error) {
	m.savedData = data
	m.savedRoom = roomID
	if m.err != nil {
		return "", m.err
	}
	return m.relPath, nil
}

// newTestRoom creates a running room and returns it along with a cleanup function.
func newTestRoom(t *testing.T) *Room {
	t.Helper()
	hub := NewHub(testLogger())
	room := hub.CreateRoom(nil)
	t.Cleanup(func() {
		room.shutdownOnce.Do(func() { close(room.shutdown) })
		<-room.closed
	})
	return room
}

// newTestClient creates a client wired to the given room (no real WebSocket conn).
func newTestClient(room *Room, store UploadStore, baseURL string) *Client {
	return &Client{
		room:          room,
		user:          model.User{ID: uuid.New(), Name: "tester"},
		send:          make(chan []byte, 256),
		systemUser:    model.User{ID: uuid.New(), Name: "system"},
		uploadStore:   store,
		uploadBaseURL: baseURL,
		logger:        testLogger(),
	}
}

// drainMessage reads one message from the room broadcast channel (via send) after broadcasting.
func drainBroadcast(t *testing.T, room *Room) []byte {
	t.Helper()
	// Give the room goroutine time to process
	time.Sleep(50 * time.Millisecond)
	return nil
}

func TestClientCloseSend(t *testing.T) {
	client := &Client{
		send:   make(chan []byte, 256),
		closed: false,
		logger: testLogger(),
	}

	client.CloseSend()

	_, ok := <-client.send
	if ok {
		t.Error("send channel should be closed")
	}

	// Second close should not panic
	client.CloseSend()
}

// --- handleTextMessage tests ---

func TestHandleTextMessage_DefaultType(t *testing.T) {
	room := newTestRoom(t)
	client := newTestClient(room, nil, "")
	room.register <- client
	time.Sleep(50 * time.Millisecond)

	data := []byte(`{"message": "hello"}`)
	ok := client.handleTextMessage(data)
	if !ok {
		t.Fatal("expected handleTextMessage to return true")
	}

	// Wait for broadcast to be processed and delivered to send channel
	time.Sleep(50 * time.Millisecond)

	select {
	case msg := <-client.send:
		var out model.OutgoingMessage
		if err := json.Unmarshal(msg, &out); err != nil {
			t.Fatalf("failed to unmarshal broadcast: %v", err)
		}
		if out.MessageType != model.UserMessage {
			t.Errorf("expected type %q, got %q", model.UserMessage, out.MessageType)
		}
		if out.Message != "hello" {
			t.Errorf("expected message 'hello', got %q", out.Message)
		}
		if out.User.ID != client.user.ID {
			t.Errorf("expected user ID %s, got %s", client.user.ID, out.User.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for broadcast")
	}

	// Message should be stored (type "message" is storable)
	msgs := room.GetMessages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 stored message, got %d", len(msgs))
	}
	if msgs[0].MessageType != model.UserMessage {
		t.Errorf("stored message type = %q, want %q", msgs[0].MessageType, model.UserMessage)
	}
}

func TestHandleTextMessage_CustomType(t *testing.T) {
	room := newTestRoom(t)
	client := newTestClient(room, nil, "")
	room.register <- client
	time.Sleep(50 * time.Millisecond)

	data := []byte(`{"type": "poll", "message": "vote?", "additionalInfo": {"options": ["A", "B"]}}`)
	ok := client.handleTextMessage(data)
	if !ok {
		t.Fatal("expected handleTextMessage to return true")
	}

	time.Sleep(50 * time.Millisecond)

	select {
	case msg := <-client.send:
		var out model.OutgoingMessage
		if err := json.Unmarshal(msg, &out); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if out.MessageType != "poll" {
			t.Errorf("expected type 'poll', got %q", out.MessageType)
		}
		if out.AdditionalInfo == nil {
			t.Fatal("expected additionalInfo to be set")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}

	msgs := room.GetMessages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 stored message, got %d", len(msgs))
	}
}

func TestHandleTextMessage_InvalidJSON(t *testing.T) {
	room := newTestRoom(t)
	client := newTestClient(room, nil, "")

	ok := client.handleTextMessage([]byte("not json"))
	if !ok {
		t.Error("expected handleTextMessage to return true on invalid JSON (skip, don't disconnect)")
	}

	// Nothing should be broadcast
	select {
	case <-client.send:
		t.Error("expected no message on send channel")
	default:
	}
}

func TestHandleTextMessage_ImageTypeNotStored(t *testing.T) {
	room := newTestRoom(t)
	client := newTestClient(room, nil, "")
	room.register <- client
	time.Sleep(50 * time.Millisecond)

	data := []byte(`{"type": "image", "message": "base64data..."}`)
	ok := client.handleTextMessage(data)
	if !ok {
		t.Fatal("expected true")
	}

	time.Sleep(50 * time.Millisecond)
	// Drain broadcast
	<-client.send

	// Image text messages should NOT be stored (nonStorableTypes)
	msgs := room.GetMessages()
	if len(msgs) != 0 {
		t.Errorf("expected 0 stored messages for image text, got %d", len(msgs))
	}
}

// --- handleBinaryMessage tests ---

func TestHandleBinaryMessage_ImageUpload(t *testing.T) {
	room := newTestRoom(t)
	store := &mockUploadStore{relPath: "1/abc.png"}
	client := newTestClient(room, store, "http://localhost:8080/uploads")
	room.register <- client
	time.Sleep(50 * time.Millisecond)

	// PNG header bytes
	pngData := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0, 0, 0, 0, 0, 0, 0, 0}

	ok := client.handleBinaryMessage(pngData)
	if !ok {
		t.Fatal("expected true")
	}

	time.Sleep(50 * time.Millisecond)

	select {
	case msg := <-client.send:
		var out model.OutgoingMessage
		if err := json.Unmarshal(msg, &out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if out.MessageType != model.ImageMessage {
			t.Errorf("expected type %q, got %q", model.ImageMessage, out.MessageType)
		}
		if out.Message != "http://localhost:8080/uploads/1/abc.png" {
			t.Errorf("expected full URL, got %q", out.Message)
		}
		if out.AdditionalInfo["contentType"] != "image/png" {
			t.Errorf("expected contentType image/png, got %v", out.AdditionalInfo["contentType"])
		}
		if out.AdditionalInfo["fileName"] != "abc.png" {
			t.Errorf("expected fileName abc.png, got %v", out.AdditionalInfo["fileName"])
		}
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}

	if store.savedRoom != room.id {
		t.Errorf("expected roomID %d, got %d", room.id, store.savedRoom)
	}

	// Binary image uploads should be stored
	msgs := room.GetMessages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 stored message, got %d", len(msgs))
	}
	if msgs[0].MessageType != model.ImageMessage {
		t.Errorf("stored type = %q, want %q", msgs[0].MessageType, model.ImageMessage)
	}
}

func TestHandleBinaryMessage_NonImageFile(t *testing.T) {
	room := newTestRoom(t)
	store := &mockUploadStore{relPath: "1/doc.pdf"}
	client := newTestClient(room, store, "http://example.com/uploads")
	room.register <- client
	time.Sleep(50 * time.Millisecond)

	// PDF header
	pdfData := []byte("%PDF-1.4 some content here padding")

	ok := client.handleBinaryMessage(pdfData)
	if !ok {
		t.Fatal("expected true")
	}

	time.Sleep(50 * time.Millisecond)

	select {
	case msg := <-client.send:
		var out model.OutgoingMessage
		if err := json.Unmarshal(msg, &out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if out.MessageType != "file" {
			t.Errorf("expected type 'file', got %q", out.MessageType)
		}
		if out.Message != "http://example.com/uploads/1/doc.pdf" {
			t.Errorf("expected full URL, got %q", out.Message)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
}

func TestHandleBinaryMessage_UploadsDisabled(t *testing.T) {
	room := newTestRoom(t)
	client := newTestClient(room, nil, "")

	ok := client.handleBinaryMessage([]byte("data"))
	if !ok {
		t.Fatal("expected true")
	}

	select {
	case msg := <-client.send:
		var out model.OutgoingMessage
		if err := json.Unmarshal(msg, &out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !strings.Contains(out.Message, "uploads are disabled") {
			t.Errorf("expected error about disabled uploads, got %q", out.Message)
		}
		if out.AdditionalInfo["error"] != true {
			t.Error("expected error flag in additionalInfo")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for error message")
	}
}

func TestHandleBinaryMessage_TooLarge(t *testing.T) {
	room := newTestRoom(t)
	store := &mockUploadStore{relPath: "1/big.bin"}
	client := newTestClient(room, store, "http://localhost/uploads")

	bigData := make([]byte, maxUploadSize+1)
	ok := client.handleBinaryMessage(bigData)
	if !ok {
		t.Fatal("expected true")
	}

	select {
	case msg := <-client.send:
		var out model.OutgoingMessage
		json.Unmarshal(msg, &out)
		if !strings.Contains(out.Message, "upload too large") {
			t.Errorf("expected size error, got %q", out.Message)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}

	if store.savedData != nil {
		t.Error("store.Save should not have been called")
	}
}

func TestHandleBinaryMessage_EmptyData(t *testing.T) {
	room := newTestRoom(t)
	store := &mockUploadStore{relPath: "1/empty.bin"}
	client := newTestClient(room, store, "http://localhost/uploads")

	ok := client.handleBinaryMessage([]byte{})
	if !ok {
		t.Fatal("expected true")
	}

	select {
	case msg := <-client.send:
		var out model.OutgoingMessage
		json.Unmarshal(msg, &out)
		if !strings.Contains(out.Message, "empty upload") {
			t.Errorf("expected empty error, got %q", out.Message)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
}

func TestHandleBinaryMessage_SaveError(t *testing.T) {
	room := newTestRoom(t)
	store := &mockUploadStore{err: errors.New("disk full")}
	client := newTestClient(room, store, "http://localhost/uploads")

	ok := client.handleBinaryMessage([]byte("some data"))
	if !ok {
		t.Fatal("expected true")
	}

	select {
	case msg := <-client.send:
		var out model.OutgoingMessage
		json.Unmarshal(msg, &out)
		if !strings.Contains(out.Message, "failed to save upload") {
			t.Errorf("expected save error, got %q", out.Message)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
}

// --- sendError tests ---

func TestSendError(t *testing.T) {
	room := newTestRoom(t)
	client := newTestClient(room, nil, "")

	client.sendError("something went wrong")

	select {
	case msg := <-client.send:
		var out model.OutgoingMessage
		if err := json.Unmarshal(msg, &out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if out.MessageType != model.SystemMessage {
			t.Errorf("expected system type, got %q", out.MessageType)
		}
		if out.Message != "something went wrong" {
			t.Errorf("expected error message, got %q", out.Message)
		}
		if out.AdditionalInfo["error"] != true {
			t.Error("expected error=true in additionalInfo")
		}
		if out.User.ID != client.systemUser.ID {
			t.Errorf("expected system user, got %s", out.User.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
}

func TestSendError_FullChannel(t *testing.T) {
	room := newTestRoom(t)
	client := newTestClient(room, nil, "")

	// Fill the send channel
	for range cap(client.send) {
		client.send <- []byte("filler")
	}

	// Should not panic or block
	client.sendError("overflow")

	// Channel should still be full with original messages
	if len(client.send) != cap(client.send) {
		t.Errorf("expected channel to remain full, got len=%d", len(client.send))
	}
}

// --- handleBinaryMessage with exact size boundary ---

func TestHandleBinaryMessage_ExactMaxSize(t *testing.T) {
	room := newTestRoom(t)
	store := &mockUploadStore{relPath: fmt.Sprintf("%d/max.bin", room.id)}
	client := newTestClient(room, store, "http://localhost/uploads")
	room.register <- client
	time.Sleep(50 * time.Millisecond)

	data := make([]byte, maxUploadSize)
	ok := client.handleBinaryMessage(data)
	if !ok {
		t.Fatal("expected true for exact max size")
	}

	time.Sleep(50 * time.Millisecond)

	select {
	case msg := <-client.send:
		var out model.OutgoingMessage
		json.Unmarshal(msg, &out)
		// Should succeed, not error
		if out.AdditionalInfo["error"] == true {
			t.Error("expected success for exact max size, got error")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
}
