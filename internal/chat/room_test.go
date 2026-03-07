package chat

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/choffmann/chat-room/internal/model"
	"github.com/google/uuid"
)

func TestRoomBroadcastToAllClients(t *testing.T) {
	h := NewHub(testLogger())

	room := &Room{
		id:         1,
		hub:        h,
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 10),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		closed:     make(chan struct{}),
		shutdown:   make(chan struct{}),
		logger:     testLogger(),
	}

	numClients := 3
	clients := make([]*Client, numClients)
	for i := range numClients {
		client := &Client{
			room:   room,
			user:   model.User{ID: uuid.New(), Name: "TestUser"},
			send:   make(chan []byte, 256),
			logger: testLogger(),
		}
		clients[i] = client
	}

	go room.Run()

	for _, client := range clients {
		room.register <- client
		time.Sleep(10 * time.Millisecond)
	}

	testMessage := []byte("test broadcast message")
	room.broadcast <- testMessage

	time.Sleep(50 * time.Millisecond)

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

	close(room.shutdown)
	<-room.closed
}

func TestRoomRegisterAndUnregister(t *testing.T) {
	h := NewHub(testLogger())

	room := &Room{
		id:         1,
		hub:        h,
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 10),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		closed:     make(chan struct{}),
		shutdown:   make(chan struct{}),
		logger:     testLogger(),
	}

	go room.Run()

	client := &Client{
		room:   room,
		user:   model.User{ID: uuid.New(), Name: "TestUser"},
		send:   make(chan []byte, 256),
		logger: testLogger(),
	}

	room.register <- client
	time.Sleep(50 * time.Millisecond)

	count := room.GetClientCount()
	if count != 1 {
		t.Errorf("expected 1 client, got %d", count)
	}

	room.unregister <- client
	time.Sleep(50 * time.Millisecond)

	count = room.GetClientCount()
	if count != 0 {
		t.Errorf("expected 0 clients after unregister, got %d", count)
	}

	select {
	case _, ok := <-client.send:
		if ok {
			t.Error("expected send channel to be closed")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("send channel was not closed")
	}

	close(room.shutdown)
	<-room.closed
}

func TestRoomShutdown(t *testing.T) {
	h := NewHub(testLogger())

	room := &Room{
		id:         1,
		hub:        h,
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 10),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		closed:     make(chan struct{}),
		shutdown:   make(chan struct{}),
		logger:     testLogger(),
	}

	for range 3 {
		client := &Client{
			room:   room,
			user:   model.User{ID: uuid.New(), Name: "TestUser"},
			send:   make(chan []byte, 256),
			logger: testLogger(),
		}
		room.clientsMu.Lock()
		room.clients[client] = true
		room.clientsMu.Unlock()
	}

	go room.Run()

	room.shutdownOnce.Do(func() {
		close(room.shutdown)
	})

	select {
	case <-room.closed:
	case <-time.After(2 * time.Second):
		t.Error("room did not close after shutdown signal")
	}
}

func TestRoomTryBroadcastAfterShutdown(t *testing.T) {
	h := NewHub(testLogger())

	room := &Room{
		id:         1,
		hub:        h,
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		closed:     make(chan struct{}),
		shutdown:   make(chan struct{}),
		logger:     testLogger(),
	}

	go room.Run()

	room.shutdownOnce.Do(func() {
		close(room.shutdown)
	})
	<-room.closed
	time.Sleep(10 * time.Millisecond)

	result := room.TryBroadcast([]byte("test"))
	if result {
		t.Error("TryBroadcast should return false after shutdown")
	}
}

func TestRoomTryRegisterAfterShutdown(t *testing.T) {
	h := NewHub(testLogger())

	room := &Room{
		id:         1,
		hub:        h,
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 10),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		closed:     make(chan struct{}),
		shutdown:   make(chan struct{}),
		logger:     testLogger(),
	}

	go room.Run()

	room.shutdownOnce.Do(func() {
		close(room.shutdown)
	})
	<-room.closed
	time.Sleep(10 * time.Millisecond)

	client := &Client{
		room:   room,
		user:   model.User{ID: uuid.New(), Name: "TestUser"},
		send:   make(chan []byte, 256),
		logger: testLogger(),
	}

	result := room.TryRegister(client)
	if result {
		t.Error("TryRegister should return false after shutdown")
	}
}

func TestRoomTimeoutWithNoActivity(t *testing.T) {
	h := NewHub(testLogger())

	room := &Room{
		id:           1,
		hub:          h,
		clients:      make(map[*Client]bool),
		broadcast:    make(chan []byte, 10),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		closed:       make(chan struct{}),
		shutdown:     make(chan struct{}),
		lastActivity: time.Now().Add(-4 * time.Hour),
		logger:       testLogger(),
	}
	h.rooms[1] = room

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				room.activityMu.RLock()
				timeSinceActivity := time.Since(room.lastActivity)
				room.activityMu.RUnlock()

				if timeSinceActivity > RoomTimeout {
					room.shutdownOnce.Do(func() {
						close(room.shutdown)
					})
					room.DisconnectAllClients()
					h.DeleteRoom(room.id)
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	time.Sleep(300 * time.Millisecond)

	_, ok := h.GetRoom(1)
	if ok {
		t.Error("room should have been deleted due to timeout")
	}

	cancel()
}

func TestRoomTimeoutPreventedByActivity(t *testing.T) {
	room := &Room{
		id:           1,
		clients:      make(map[*Client]bool),
		broadcast:    make(chan []byte, 10),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		closed:       make(chan struct{}),
		shutdown:     make(chan struct{}),
		lastActivity: time.Now().Add(-30 * time.Minute),
		logger:       testLogger(),
	}

	room.UpdateActivityNow()

	room.activityMu.RLock()
	timeSince := time.Since(room.lastActivity)
	room.activityMu.RUnlock()

	if timeSince > 1*time.Second {
		t.Errorf("expected recent activity, but it was %v ago", timeSince)
	}

	if timeSince > RoomTimeout {
		t.Error("room should not timeout after activity update")
	}
}

func TestRoomDisconnectAllClients(t *testing.T) {
	room := &Room{
		id:      1,
		clients: make(map[*Client]bool),
		logger:  testLogger(),
	}

	numClients := 5
	clients := make([]*Client, numClients)
	for i := range numClients {
		client := &Client{
			room:   room,
			user:   model.User{ID: uuid.New(), Name: "TestUser"},
			send:   make(chan []byte, 256),
			logger: testLogger(),
		}
		clients[i] = client
		room.clients[client] = true
	}

	room.DisconnectAllClients()

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
	h := NewHub(testLogger())

	room := &Room{
		id:         1,
		hub:        h,
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 10),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		closed:     make(chan struct{}),
		shutdown:   make(chan struct{}),
		logger:     testLogger(),
	}

	go room.Run()

	goodClient := &Client{
		room:   room,
		user:   model.User{ID: uuid.New(), Name: "GoodClient"},
		send:   make(chan []byte, 256),
		logger: testLogger(),
	}

	badClient := &Client{
		room:   room,
		user:   model.User{ID: uuid.New(), Name: "BadClient"},
		send:   make(chan []byte, 1),
		logger: testLogger(),
	}
	badClient.send <- []byte("block")

	room.register <- goodClient
	room.register <- badClient
	time.Sleep(50 * time.Millisecond)

	testMessage := []byte("broadcast test")
	room.broadcast <- testMessage
	time.Sleep(100 * time.Millisecond)

	select {
	case msg := <-goodClient.send:
		if string(msg) != string(testMessage) {
			t.Errorf("expected message %s, got %s", testMessage, msg)
		}
	case <-time.After(1 * time.Second):
		t.Error("good client did not receive message")
	}

	time.Sleep(100 * time.Millisecond)
	count := room.GetClientCount()
	if count != 1 {
		t.Errorf("expected 1 client (bad client should be removed), got %d", count)
	}

	close(room.shutdown)
	<-room.closed
}

func TestConcurrentRoomOperations(t *testing.T) {
	h := NewHub(testLogger())

	room := &Room{
		id:         1,
		hub:        h,
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 100),
		register:   make(chan *Client, 100),
		unregister: make(chan *Client, 100),
		closed:     make(chan struct{}),
		shutdown:   make(chan struct{}),
		logger:     testLogger(),
	}

	go room.Run()

	var wg sync.WaitGroup
	numGoroutines := 10

	wg.Add(numGoroutines)
	for i := range numGoroutines {
		go func(n int) {
			defer wg.Done()
			client := &Client{
				room:   room,
				user:   model.User{ID: uuid.New(), Name: "TestUser"},
				send:   make(chan []byte, 256),
				logger: testLogger(),
			}
			room.register <- client
			time.Sleep(10 * time.Millisecond)
		}(i)
	}
	wg.Wait()

	time.Sleep(100 * time.Millisecond)

	count := room.GetClientCount()
	if count != numGoroutines {
		t.Errorf("expected %d clients, got %d", numGoroutines, count)
	}

	wg.Add(numGoroutines)
	for i := range numGoroutines {
		go func(n int) {
			defer wg.Done()
			msg := model.OutgoingMessage{
				MessageType: "message",
				Message:     "concurrent test",
			}
			data, _ := json.Marshal(msg)
			room.broadcast <- data
		}(i)
	}
	wg.Wait()

	time.Sleep(100 * time.Millisecond)

	close(room.shutdown)
	<-room.closed
}

func TestRoomGetUsersLogic(t *testing.T) {
	room := &Room{
		id:      1,
		clients: make(map[*Client]bool),
		logger:  testLogger(),
	}

	users := []model.User{
		{ID: uuid.New(), FirstName: "John", LastName: "Doe"},
		{ID: uuid.New(), FirstName: "Jane", LastName: "Smith"},
		{ID: uuid.New(), Name: "alice"},
	}

	for _, user := range users {
		client := &Client{
			room:   room,
			user:   user,
			send:   make(chan []byte, 256),
			logger: testLogger(),
		}
		room.clients[client] = true
	}

	retrievedUsers := room.GetUsers()

	if len(retrievedUsers) != len(users) {
		t.Errorf("expected %d users, got %d", len(users), len(retrievedUsers))
	}

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

func TestRoomUpdateActivityOnBroadcast(t *testing.T) {
	h := NewHub(testLogger())

	room := &Room{
		id:           1,
		hub:          h,
		clients:      make(map[*Client]bool),
		broadcast:    make(chan []byte, 10),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		closed:       make(chan struct{}),
		shutdown:     make(chan struct{}),
		lastActivity: time.Now().Add(-1 * time.Hour),
		logger:       testLogger(),
	}

	client := &Client{
		room:   room,
		user:   model.User{ID: uuid.New(), Name: "TestUser"},
		send:   make(chan []byte, 256),
		logger: testLogger(),
	}
	room.clientsMu.Lock()
	room.clients[client] = true
	room.clientsMu.Unlock()

	go room.Run()

	room.broadcast <- []byte("test message")
	time.Sleep(100 * time.Millisecond)

	room.activityMu.RLock()
	timeSince := time.Since(room.lastActivity)
	room.activityMu.RUnlock()

	if timeSince > 1*time.Second {
		t.Errorf("expected activity to be updated on broadcast, but it was %v ago", timeSince)
	}

	close(room.shutdown)
	<-room.closed
}
