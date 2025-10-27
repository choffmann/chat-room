package main

import (
	"log"
	"sync"
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
		additionalInfo: additionalInfo,
	}

	log.Printf("creating new room with id: %d", id)
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

func (h *Hub) GetAllRoomIDs() []uint {
	h.mu.RLock()
	defer h.mu.RUnlock()
	rooms := make([]uint, 0, len(h.rooms))
	for _, room := range h.rooms {
		rooms = append(rooms, room.id)
	}
	return rooms
}

func (h *Hub) DeleteRoom(id uint) {
	log.Printf("delete room with id: %d", id)
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.rooms, id)
}

func (r *Room) run() {
	defer close(r.closed)

	for {
		select {
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
