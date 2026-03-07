package chat

import (
	"io"
	"log/slog"
	"testing"

	"github.com/choffmann/chat-room/internal/model"
	"github.com/google/uuid"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestHubCreateAndGetRoom(t *testing.T) {
	h := NewHub(testLogger())

	additionalInfo := model.AdditionalInfo{
		"name": "Test Room",
		"type": "public",
	}

	room := h.CreateRoom(additionalInfo)
	defer func() {
		room.ShutdownOnce(func() { close(room.shutdown) })
		<-room.closed
	}()

	if room == nil {
		t.Fatal("CreateRoom returned nil")
	}

	if room.id == 0 {
		t.Error("room ID should not be 0")
	}

	info := room.GetAdditionalInfo()
	if info["name"] != "Test Room" {
		t.Errorf("expected name 'Test Room', got %v", info["name"])
	}

	retrievedRoom, ok := h.GetRoom(room.id)
	if !ok {
		t.Error("GetRoom should return the created room")
	}

	if retrievedRoom.id != room.id {
		t.Errorf("expected room ID %d, got %d", room.id, retrievedRoom.id)
	}
}

func TestHubDeleteRoom(t *testing.T) {
	h := NewHub(testLogger())

	room := h.CreateRoom(nil)
	roomID := room.id

	room.ShutdownOnce(func() { close(room.shutdown) })
	<-room.closed

	h.DeleteRoom(roomID)

	_, ok := h.GetRoom(roomID)
	if ok {
		t.Error("room should be deleted from hub")
	}
}

func TestHubGetAllRoomIDs(t *testing.T) {
	h := NewHub(testLogger())

	numRooms := 5
	rooms := make([]*Room, numRooms)
	for i := range numRooms {
		room := h.CreateRoom(model.AdditionalInfo{
			"name": "Room " + string(rune(i+1)),
		})
		rooms[i] = room
	}

	roomResponses := h.GetAllRoomIDs()

	if len(roomResponses) != numRooms {
		t.Errorf("expected %d rooms, got %d", numRooms, len(roomResponses))
	}

	for i := 1; i < len(roomResponses); i++ {
		if roomResponses[i].ID <= roomResponses[i-1].ID {
			t.Error("rooms should be sorted by ID")
		}
	}

	for _, room := range rooms {
		room.ShutdownOnce(func() { close(room.shutdown) })
		<-room.closed
	}
}

func TestNewRoomID(t *testing.T) {
	h := NewHub(testLogger())

	id1 := h.newRoomID()
	id2 := h.newRoomID()
	id3 := h.newRoomID()

	if id2 != id1+1 {
		t.Errorf("expected sequential IDs, got %d and %d", id1, id2)
	}

	if id3 != id2+1 {
		t.Errorf("expected sequential IDs, got %d and %d", id2, id3)
	}
}

func TestHubGetAllUsersWithRooms(t *testing.T) {
	h := NewHub(testLogger())

	room1 := &Room{
		id:      1,
		clients: make(map[*Client]bool),
	}
	room2 := &Room{
		id:      2,
		clients: make(map[*Client]bool),
	}

	user1 := model.User{FirstName: "John"}
	user2 := model.User{FirstName: "Jane"}

	room1.clients[&Client{user: user1}] = true
	room2.clients[&Client{user: user2}] = true

	h.rooms[1] = room1
	h.rooms[2] = room2

	usersWithRooms := h.GetAllUsersWithRooms()

	if len(usersWithRooms) != 2 {
		t.Errorf("expected 2 users with rooms, got %d", len(usersWithRooms))
	}

	for _, uwr := range usersWithRooms {
		if uwr.RoomID == 0 {
			t.Error("expected roomID to be set")
		}
	}
}

func TestMessageTypeValidation(t *testing.T) {
	validTypes := []model.MessageType{model.SystemMessage, "message", "image"}

	for _, msgType := range validTypes {
		msg := model.OutgoingMessage{
			MessageType: msgType,
			Message:     "test",
		}

		if msg.MessageType != msgType {
			t.Errorf("expected message type %s, got %s", msgType, msg.MessageType)
		}
	}
}

func TestRoomGetUsers(t *testing.T) {
	room := &Room{
		id:      1,
		clients: make(map[*Client]bool),
	}

	client1 := &Client{user: model.User{FirstName: "John", LastName: "Doe"}}
	client2 := &Client{user: model.User{FirstName: "Jane", LastName: "Smith"}}

	room.clients[client1] = true
	room.clients[client2] = true

	users := room.GetUsers()

	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestRoomGetClientCount(t *testing.T) {
	room := &Room{
		id:      1,
		clients: make(map[*Client]bool),
	}

	if count := room.GetClientCount(); count != 0 {
		t.Errorf("expected 0 clients, got %d", count)
	}

	room.clients[&Client{user: model.User{ID: uuid.New()}}] = true
	room.clients[&Client{user: model.User{ID: uuid.New()}}] = true

	if count := room.GetClientCount(); count != 2 {
		t.Errorf("expected 2 clients, got %d", count)
	}
}
