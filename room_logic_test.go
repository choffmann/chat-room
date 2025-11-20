package main

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func setupRoomLogicTests() {
	hub = &Hub{
		rooms: make(map[uint]*Room),
	}
	roomCounter = 0
}

func TestRoomBroadcastToAllClients(t *testing.T) {
	setupRoomLogicTests()

	room := &Room{
		id:         1,
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 10),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		closed:     make(chan struct{}),
		shutdown:   make(chan struct{}),
	}

	// Create mock clients with channels
	numClients := 3
	clients := make([]*Client, numClients)
	for i := range numClients {
		client := &Client{
			room: room,
			user: User{ID: uuid.New(), Name: "TestUser"},
			send: make(chan []byte, 256),
		}
		clients[i] = client
	}

	// Start room
	go room.run()

	// Register clients
	for _, client := range clients {
		room.register <- client
		time.Sleep(10 * time.Millisecond) // Give time to process
	}

	// Broadcast a message
	testMessage := []byte("test broadcast message")
	room.broadcast <- testMessage

	// Wait a bit for broadcast to complete
	time.Sleep(50 * time.Millisecond)

	// Verify all clients received the message
	for i, client := range clients {
		select {
		case msg := <-client.send:
			if string(msg) != string(testMessage) {
				t.Errorf("client %d: expected message %s, got %s", i, testMessage, msg)
			}
		case <-time.After(1 * time.Second):
			t.Errorf("client %d: did not receive broadcast message", i)
		}
	}

	// Cleanup
	close(room.shutdown)
	<-room.closed
}

func TestRoomRegisterAndUnregister(t *testing.T) {
	setupRoomLogicTests()

	room := &Room{
		id:         1,
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 10),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		closed:     make(chan struct{}),
		shutdown:   make(chan struct{}),
	}

	go room.run()

	client := &Client{
		room: room,
		user: User{ID: uuid.New(), Name: "TestUser"},
		send: make(chan []byte, 256),
	}

	// Register client
	room.register <- client
	time.Sleep(50 * time.Millisecond)

	// Check client is registered
	count := room.GetClientCount()
	if count != 1 {
		t.Errorf("expected 1 client, got %d", count)
	}

	// Unregister client
	room.unregister <- client
	time.Sleep(50 * time.Millisecond)

	// Check client is unregistered
	count = room.GetClientCount()
	if count != 0 {
		t.Errorf("expected 0 clients after unregister, got %d", count)
	}

	// Check that send channel is closed
	select {
	case _, ok := <-client.send:
		if ok {
			t.Error("expected send channel to be closed")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("send channel was not closed")
	}

	// Cleanup
	close(room.shutdown)
	<-room.closed
}

func TestRoomShutdown(t *testing.T) {
	setupRoomLogicTests()

	room := &Room{
		id:         1,
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 10),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		closed:     make(chan struct{}),
		shutdown:   make(chan struct{}),
	}

	// Add some clients
	for range 3 {
		client := &Client{
			room: room,
			user: User{ID: uuid.New(), Name: "TestUser"},
			send: make(chan []byte, 256),
		}
		room.clientsMu.Lock()
		room.clients[client] = true
		room.clientsMu.Unlock()
	}

	go room.run()

	// Trigger shutdown
	room.shutdownOnce.Do(func() {
		close(room.shutdown)
	})

	// Wait for room to close
	select {
	case <-room.closed:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("room did not close after shutdown signal")
	}
}

func TestRoomTryBroadcastAfterShutdown(t *testing.T) {
	setupRoomLogicTests()

	room := &Room{
		id:         1,
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		closed:     make(chan struct{}),
		shutdown:   make(chan struct{}),
	}

	go room.run()

	room.shutdownOnce.Do(func() {
		close(room.shutdown)
	})
	<-room.closed
	time.Sleep(10 * time.Millisecond)

	result := room.tryBroadcast([]byte("test"))
	if result {
		t.Error("tryBroadcast should return false after shutdown")
	}
}

func TestRoomTryRegisterAfterShutdown(t *testing.T) {
	setupRoomLogicTests()

	room := &Room{
		id:         1,
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 10),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		closed:     make(chan struct{}),
		shutdown:   make(chan struct{}),
	}

	go room.run()

	room.shutdownOnce.Do(func() {
		close(room.shutdown)
	})
	<-room.closed
	time.Sleep(10 * time.Millisecond)

	client := &Client{
		room: room,
		user: User{ID: uuid.New(), Name: "TestUser"},
		send: make(chan []byte, 256),
	}

	result := room.tryRegister(client)
	if result {
		t.Error("tryRegister should return false after shutdown")
	}
}

func TestRoomTimeoutWithNoActivity(t *testing.T) {
	setupRoomLogicTests()

	hub = &Hub{
		rooms: make(map[uint]*Room),
	}

	// Create room with old activity time
	room := &Room{
		id:           1,
		clients:      make(map[*Client]bool),
		broadcast:    make(chan []byte, 10),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		closed:       make(chan struct{}),
		shutdown:     make(chan struct{}),
		lastActivity: time.Now().Add(-4 * time.Hour), // Old activity
	}
	hub.rooms[1] = room

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start timeout checker with very short interval for testing
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				room.activityMu.RLock()
				timeSinceActivity := time.Since(room.lastActivity)
				room.activityMu.RUnlock()

				if timeSinceActivity > roomTimeout {
					room.shutdownOnce.Do(func() {
						close(room.shutdown)
					})
					room.disconnectAllClients()
					hub.DeleteRoom(room.id)
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for room to be deleted
	time.Sleep(300 * time.Millisecond)

	// Check room was deleted
	_, ok := hub.GetRoom(1)
	if ok {
		t.Error("room should have been deleted due to timeout")
	}

	cancel()
}

func TestRoomTimeoutPreventedByActivity(t *testing.T) {
	setupRoomLogicTests()

	room := &Room{
		id:           1,
		clients:      make(map[*Client]bool),
		broadcast:    make(chan []byte, 10),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		closed:       make(chan struct{}),
		shutdown:     make(chan struct{}),
		lastActivity: time.Now().Add(-30 * time.Minute),
	}

	// Update activity (simulating recent message)
	room.UpdateActivityNow()

	// Check that activity was updated
	room.activityMu.RLock()
	timeSince := time.Since(room.lastActivity)
	room.activityMu.RUnlock()

	if timeSince > 1*time.Second {
		t.Errorf("expected recent activity, but it was %v ago", timeSince)
	}

	// Room should not timeout now
	if timeSince > roomTimeout {
		t.Error("room should not timeout after activity update")
	}
}

func TestRoomDisconnectAllClients(t *testing.T) {
	setupRoomLogicTests()

	room := &Room{
		id:      1,
		clients: make(map[*Client]bool),
	}

	// Create clients
	numClients := 5
	clients := make([]*Client, numClients)
	for i := range numClients {
		client := &Client{
			room: room,
			user: User{ID: uuid.New(), Name: "TestUser"},
			send: make(chan []byte, 256),
		}
		clients[i] = client
		room.clients[client] = true
	}

	// Disconnect all clients
	room.disconnectAllClients()

	// Verify all send channels are closed
	for i, client := range clients {
		select {
		case _, ok := <-client.send:
			if ok {
				t.Errorf("client %d: expected send channel to be closed", i)
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("client %d: send channel was not closed", i)
		}
	}
}

func TestRoomBroadcastWithFailedClient(t *testing.T) {
	setupRoomLogicTests()

	room := &Room{
		id:         1,
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 10),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		closed:     make(chan struct{}),
		shutdown:   make(chan struct{}),
	}

	go room.run()

	// Create normal client
	goodClient := &Client{
		room: room,
		user: User{ID: uuid.New(), Name: "GoodClient"},
		send: make(chan []byte, 256),
	}

	// Create client with full buffer (will fail to receive)
	badClient := &Client{
		room: room,
		user: User{ID: uuid.New(), Name: "BadClient"},
		send: make(chan []byte, 1), // Small buffer
	}
	badClient.send <- []byte("block") // Fill the buffer

	// Register clients
	room.register <- goodClient
	room.register <- badClient
	time.Sleep(50 * time.Millisecond)

	// Broadcast message
	testMessage := []byte("broadcast test")
	room.broadcast <- testMessage
	time.Sleep(100 * time.Millisecond)

	// Good client should receive message
	select {
	case msg := <-goodClient.send:
		if string(msg) != string(testMessage) {
			t.Errorf("expected message %s, got %s", testMessage, msg)
		}
	case <-time.After(1 * time.Second):
		t.Error("good client did not receive message")
	}

	// Bad client should be removed from room
	time.Sleep(100 * time.Millisecond)
	count := room.GetClientCount()
	if count != 1 {
		t.Errorf("expected 1 client (bad client should be removed), got %d", count)
	}

	// Cleanup
	close(room.shutdown)
	<-room.closed
}

func TestConcurrentRoomOperations(t *testing.T) {
	setupRoomLogicTests()

	room := &Room{
		id:         1,
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 100),
		register:   make(chan *Client, 100),
		unregister: make(chan *Client, 100),
		closed:     make(chan struct{}),
		shutdown:   make(chan struct{}),
	}

	go room.run()

	var wg sync.WaitGroup
	numGoroutines := 10

	// Concurrent registrations
	wg.Add(numGoroutines)
	for i := range numGoroutines {
		go func(n int) {
			defer wg.Done()
			client := &Client{
				room: room,
				user: User{ID: uuid.New(), Name: "TestUser"},
				send: make(chan []byte, 256),
			}
			room.register <- client
			time.Sleep(10 * time.Millisecond)
		}(i)
	}
	wg.Wait()

	time.Sleep(100 * time.Millisecond)

	// Verify clients were registered
	count := room.GetClientCount()
	if count != numGoroutines {
		t.Errorf("expected %d clients, got %d", numGoroutines, count)
	}

	// Concurrent broadcasts
	wg.Add(numGoroutines)
	for i := range numGoroutines {
		go func(n int) {
			defer wg.Done()
			msg := OutgoingMessage{
				MessageType: "message",
				Message:     "concurrent test",
			}
			data, _ := json.Marshal(msg)
			room.broadcast <- data
		}(i)
	}
	wg.Wait()

	time.Sleep(100 * time.Millisecond)

	// Cleanup
	close(room.shutdown)
	<-room.closed
}

func TestRoomGetUsersLogic(t *testing.T) {
	setupRoomLogicTests()

	room := &Room{
		id:      1,
		clients: make(map[*Client]bool),
	}

	// Add clients with different users
	users := []User{
		{ID: uuid.New(), FirstName: "John", LastName: "Doe"},
		{ID: uuid.New(), FirstName: "Jane", LastName: "Smith"},
		{ID: uuid.New(), Name: "alice"},
	}

	for _, user := range users {
		client := &Client{
			room: room,
			user: user,
			send: make(chan []byte, 256),
		}
		room.clients[client] = true
	}

	// Get users
	retrievedUsers := room.GetUsers()

	if len(retrievedUsers) != len(users) {
		t.Errorf("expected %d users, got %d", len(users), len(retrievedUsers))
	}

	// Verify users are in the list
	for _, expectedUser := range users {
		found := false
		for _, retrievedUser := range retrievedUsers {
			if retrievedUser.ID == expectedUser.ID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("user %s not found in retrieved users", expectedUser.ID)
		}
	}
}

func TestHubCreateAndGetRoom(t *testing.T) {
	setupRoomLogicTests()

	testHub := &Hub{
		rooms: make(map[uint]*Room),
	}

	additionalInfo := AdditionalInfo{
		"name": "Test Room",
		"type": "public",
	}

	room := testHub.CreateRoom(additionalInfo)
	defer func() {
		room.shutdownOnce.Do(func() { close(room.shutdown) })
		<-room.closed
	}()

	if room == nil {
		t.Fatal("CreateRoom returned nil")
	}

	if room.id == 0 {
		t.Error("room ID should not be 0")
	}

	// Verify additionalInfo was set
	info := room.GetAdditionalInfo()
	if info["name"] != "Test Room" {
		t.Errorf("expected name 'Test Room', got %v", info["name"])
	}

	// Get room from hub
	retrievedRoom, ok := testHub.GetRoom(room.id)
	if !ok {
		t.Error("GetRoom should return the created room")
	}

	if retrievedRoom.id != room.id {
		t.Errorf("expected room ID %d, got %d", room.id, retrievedRoom.id)
	}
}

func TestHubDeleteRoom(t *testing.T) {
	setupRoomLogicTests()

	testHub := &Hub{
		rooms: make(map[uint]*Room),
	}

	room := testHub.CreateRoom(nil)
	roomID := room.id

	// Close room properly
	room.shutdownOnce.Do(func() { close(room.shutdown) })
	<-room.closed

	// Delete room
	testHub.DeleteRoom(roomID)

	// Verify room is deleted
	_, ok := testHub.GetRoom(roomID)
	if ok {
		t.Error("room should be deleted from hub")
	}
}

func TestHubGetAllRoomIDs(t *testing.T) {
	setupRoomLogicTests()

	testHub := &Hub{
		rooms: make(map[uint]*Room),
	}

	// Create multiple rooms
	numRooms := 5
	rooms := make([]*Room, numRooms)
	for i := range numRooms {
		room := testHub.CreateRoom(AdditionalInfo{
			"name": "Room " + string(rune(i+1)),
		})
		rooms[i] = room
	}

	// Get all room IDs
	roomResponses := testHub.GetAllRoomIDs()

	if len(roomResponses) != numRooms {
		t.Errorf("expected %d rooms, got %d", numRooms, len(roomResponses))
	}

	// Verify rooms are sorted by ID
	for i := 1; i < len(roomResponses); i++ {
		if roomResponses[i].ID <= roomResponses[i-1].ID {
			t.Error("rooms should be sorted by ID")
		}
	}

	// Cleanup
	for _, room := range rooms {
		room.shutdownOnce.Do(func() { close(room.shutdown) })
		<-room.closed
	}
}

func TestNewRoomID(t *testing.T) {
	setupRoomLogicTests()

	// Get multiple IDs
	id1 := newRoomID()
	id2 := newRoomID()
	id3 := newRoomID()

	// Verify IDs are sequential
	if id2 != id1+1 {
		t.Errorf("expected sequential IDs, got %d and %d", id1, id2)
	}

	if id3 != id2+1 {
		t.Errorf("expected sequential IDs, got %d and %d", id2, id3)
	}
}

func TestRoomUpdateActivityOnBroadcast(t *testing.T) {
	setupRoomLogicTests()

	room := &Room{
		id:           1,
		clients:      make(map[*Client]bool),
		broadcast:    make(chan []byte, 10),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		closed:       make(chan struct{}),
		shutdown:     make(chan struct{}),
		lastActivity: time.Now().Add(-1 * time.Hour),
	}

	// Add a client
	client := &Client{
		room: room,
		user: User{ID: uuid.New(), Name: "TestUser"},
		send: make(chan []byte, 256),
	}
	room.clientsMu.Lock()
	room.clients[client] = true
	room.clientsMu.Unlock()

	go room.run()

	// Send broadcast
	room.broadcast <- []byte("test message")
	time.Sleep(100 * time.Millisecond)

	// Check activity was updated
	room.activityMu.RLock()
	timeSince := time.Since(room.lastActivity)
	room.activityMu.RUnlock()

	if timeSince > 1*time.Second {
		t.Errorf("expected activity to be updated on broadcast, but it was %v ago", timeSince)
	}

	// Cleanup
	close(room.shutdown)
	<-room.closed
}

func TestClientCloseSend(t *testing.T) {
	client := &Client{
		send:   make(chan []byte, 256),
		closed: false,
	}

	// First close
	client.closeSend()

	// Verify channel is closed
	_, ok := <-client.send
	if ok {
		t.Error("send channel should be closed")
	}

	// Second close should not panic
	client.closeSend()
}

func TestMessageTypeValidation(t *testing.T) {
	validTypes := []MessageType{SystemMessage, "message", "image"}

	for _, msgType := range validTypes {
		msg := OutgoingMessage{
			MessageType: msgType,
			Message:     "test",
			Timestamp:   time.Now(),
		}

		data, err := json.Marshal(msg)
		if err != nil {
			t.Errorf("failed to marshal message type %s: %v", msgType, err)
		}

		var parsed OutgoingMessage
		err = json.Unmarshal(data, &parsed)
		if err != nil {
			t.Errorf("failed to unmarshal message type %s: %v", msgType, err)
		}

		if parsed.MessageType != msgType {
			t.Errorf("expected message type %s, got %s", msgType, parsed.MessageType)
		}
	}
}

func TestWebSocketUpgrader(t *testing.T) {
	// Just verify the upgrader is configured correctly
	if upgrader.CheckOrigin == nil {
		t.Error("CheckOrigin should be set")
	}

	if upgrader.ReadBufferSize == 0 {
		t.Error("ReadBufferSize should be set")
	}

	if upgrader.WriteBufferSize == 0 {
		t.Error("WriteBufferSize should be set")
	}
}
