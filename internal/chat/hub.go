package chat

import (
	"log/slog"
	"sort"
	"sync"

	"github.com/choffmann/chat-room/internal/model"
)

type Hub struct {
	mu          sync.RWMutex
	rooms       map[uint]*Room
	roomCounter int
	roomMu      sync.Mutex
	logger      *slog.Logger
}

func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		rooms:  make(map[uint]*Room),
		logger: logger,
	}
}

func (h *Hub) newRoomID() uint {
	h.roomMu.Lock()
	defer h.roomMu.Unlock()
	h.roomCounter++
	return uint(h.roomCounter)
}

func (h *Hub) CreateRoom(additionalInfo model.AdditionalInfo) *Room {
	id := h.newRoomID()
	room := &Room{
		id:             id,
		hub:            h,
		clients:        make(map[*Client]bool),
		broadcast:      make(chan []byte),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		closed:         make(chan struct{}),
		shutdown:       make(chan struct{}),
		lastActivity:   timeNow(),
		additionalInfo: additionalInfo,
		messages:       make([]model.OutgoingMessage, 0),
		logger:         h.logger,
	}

	h.logger.Info("creating new room", "roomID", id)
	h.mu.Lock()
	h.rooms[id] = room
	h.mu.Unlock()

	go room.Run()
	return room
}

func (h *Hub) GetRoom(id uint) (*Room, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	r, ok := h.rooms[id]
	return r, ok
}

func (h *Hub) GetAllRoomIDs() []model.RoomResponse {
	h.mu.RLock()
	defer h.mu.RUnlock()
	rooms := make([]model.RoomResponse, 0, len(h.rooms))
	for _, room := range h.rooms {
		rooms = append(rooms, model.RoomResponse{
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
	h.logger.Info("deleting room", "roomID", id)
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.rooms, id)
}

func (h *Hub) GetAllUsersWithRooms() []model.UserWithRoom {
	h.mu.RLock()
	defer h.mu.RUnlock()

	usersWithRooms := make([]model.UserWithRoom, 0)
	for roomID, room := range h.rooms {
		users := room.GetUsers()
		for _, user := range users {
			usersWithRooms = append(usersWithRooms, model.UserWithRoom{
				User:   user,
				RoomID: roomID,
			})
		}
	}
	return usersWithRooms
}
