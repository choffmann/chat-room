package main

import (
	"context"
	"sync"
	"time"
)

const (
	roomTimeout         = 1 * time.Hour
	roomTimeoutInterval = 30 * time.Minute
)

type Hub struct {
	mu    sync.RWMutex
	rooms map[uint]*Room
}

type Room struct {
	id             uint
	clients        map[*Client]bool
	broadcast      chan []byte
	register       chan *Client
	unregister     chan *Client
	closed         chan struct{}
	shutdown       chan struct{}
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
			UserCount:      len(room.clients),
		})
	}
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

func (r *Room) disconnectAllClients() {
	for c := range r.clients {
		close(c.send)
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
			r.clients[c] = true
		case c := <-r.unregister:
			if _, ok := r.clients[c]; ok {
				delete(r.clients, c)
				close(c.send)
				if len(r.clients) == 0 {
					hub.DeleteRoom(r.id)
					return
				}
			}

		case msg := <-r.broadcast:
			r.UpdateActivityNow()
			for c := range r.clients {
				select {
				case c.send <- msg:
				default:
					delete(r.clients, c)
					close(c.send)
				}
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

			if timeSinceActivity > roomTimeoutInterval {
				close(r.shutdown)
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
