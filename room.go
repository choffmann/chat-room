package main

import (
	"context"
	"sort"
	"sync"
	"time"
)

const (
	roomTimeout         = 3 * time.Hour
	roomTimeoutInterval = 25 * time.Second
)

type Hub struct {
	mu    sync.RWMutex
	rooms map[uint]*Room
}

type Room struct {
	id             uint
	clientsMu      sync.RWMutex
	clients        map[*Client]bool
	broadcast      chan []byte
	register       chan *Client
	unregister     chan *Client
	closed         chan struct{}
	shutdown       chan struct{}
	shutdownOnce   sync.Once
	activityMu     sync.RWMutex
	lastActivity   time.Time
	additionalInfo AdditionalInfo
}

func newRoomID() uint {
	roomMu.Lock()
	defer roomMu.Unlock()
	roomCounter++
	return uint(roomCounter)
}

func (h *Hub) CreateRoom(additionalInfo AdditionalInfo) *Room {
	id := newRoomID()
	room := &Room{
		id:             id,
		clients:        make(map[*Client]bool),
		broadcast:      make(chan []byte),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		closed:         make(chan struct{}),
		shutdown:       make(chan struct{}),
		lastActivity:   time.Now(),
		additionalInfo: additionalInfo,
	}

	logger.Info("creating new room", "roomID", id)
	h.mu.Lock()
	h.rooms[id] = room
	h.mu.Unlock()

	go room.run()
	return room
}

func (h *Hub) GetRoom(id uint) (*Room, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	r, ok := h.rooms[id]
	return r, ok
}

func (h *Hub) GetAllRoomIDs() []RoomResponse {
	h.mu.RLock()
	defer h.mu.RUnlock()
	rooms := make([]RoomResponse, 0, len(h.rooms))
	for _, room := range h.rooms {
		rooms = append(rooms, RoomResponse{
			ID:             room.id,
			AdditionalInfo: room.additionalInfo,
			UserCount:      room.GetClientCount(),
		})
	}

	sort.Slice(rooms, func(i, j int) bool {
		return rooms[i].ID < rooms[j].ID
	})
	return rooms
}

func (h *Hub) DeleteRoom(id uint) {
	logger.Info("deleting room", "roomID", id)
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.rooms, id)
}

func (r *Room) UpdateActivityNow() {
	r.activityMu.Lock()
	defer r.activityMu.Unlock()
	r.lastActivity = time.Now()
}

func (r *Room) UpdateAdditionalInfo(newInfo AdditionalInfo) {
	r.activityMu.Lock()
	defer r.activityMu.Unlock()
	r.additionalInfo = newInfo
}

func (r *Room) PatchAdditionalInfo(updates AdditionalInfo) {
	r.activityMu.Lock()
	defer r.activityMu.Unlock()
	if r.additionalInfo == nil {
		r.additionalInfo = make(AdditionalInfo)
	}
	for key, value := range updates {
		r.additionalInfo[key] = value
	}
}

func (r *Room) GetAdditionalInfo() AdditionalInfo {
	r.activityMu.RLock()
	defer r.activityMu.RUnlock()
	// Return a copy to prevent external modification
	info := make(AdditionalInfo, len(r.additionalInfo))
	for k, v := range r.additionalInfo {
		info[k] = v
	}
	return info
}

func (r *Room) disconnectAllClients() {
	r.clientsMu.Lock()
	defer r.clientsMu.Unlock()
	for c := range r.clients {
		c.closeSend()
	}
}

func (r *Room) GetClientCount() int {
	r.clientsMu.RLock()
	defer r.clientsMu.RUnlock()
	return len(r.clients)
}

func (r *Room) tryBroadcast(msg []byte) bool {
	select {
	case r.broadcast <- msg:
		return true
	case <-r.shutdown:
		return false
	}
}

func (r *Room) tryRegister(c *Client) bool {
	select {
	case r.register <- c:
		return true
	case <-r.shutdown:
		return false
	}
}

func (r *Room) tryUnregister(c *Client) bool {
	select {
	case r.unregister <- c:
		return true
	case <-r.shutdown:
		return false
	}
}

func (r *Room) run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		close(r.closed)
		cancel()
	}()

	go r.deleteRoomWithNoActivity(ctx)

	for {
		select {
		case <-r.shutdown:
			logger.Info("room shutdown signal received", "roomID", r.id)
			return

		case c := <-r.register:
			r.clientsMu.Lock()
			r.clients[c] = true
			r.clientsMu.Unlock()
			r.UpdateActivityNow()

		case c := <-r.unregister:
			r.clientsMu.Lock()
			if _, ok := r.clients[c]; ok {
				delete(r.clients, c)
				c.closeSend()
			}
			r.clientsMu.Unlock()

		case msg := <-r.broadcast:
			r.UpdateActivityNow()
			r.clientsMu.RLock()
			// Create a snapshot of clients to avoid holding lock during send
			clientsList := make([]*Client, 0, len(r.clients))
			for c := range r.clients {
				clientsList = append(clientsList, c)
			}
			r.clientsMu.RUnlock()

			// Now send to all clients
			failedClients := make([]*Client, 0)
			for _, c := range clientsList {
				select {
				case c.send <- msg:
				default:
					failedClients = append(failedClients, c)
				}
			}

			// Remove failed clients
			if len(failedClients) > 0 {
				r.clientsMu.Lock()
				for _, c := range failedClients {
					delete(r.clients, c)
					c.closeSend()
				}
				r.clientsMu.Unlock()
			}
		}
	}
}

func (r *Room) deleteRoomWithNoActivity(ctx context.Context) {
	ticker := time.NewTicker(roomTimeoutInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.activityMu.RLock()
			timeSinceActivity := time.Since(r.lastActivity)
			r.activityMu.RUnlock()

			if timeSinceActivity > roomTimeout {
				r.shutdownOnce.Do(func() {
					close(r.shutdown)
				})
				r.disconnectAllClients()
				hub.DeleteRoom(r.id)
				logger.Info("remove room due to timeout activity", "roomID", r.id)
				return
			}

		case <-ctx.Done():
			logger.Debug("stopping delete room scheduler")
			return
		}
	}
}
