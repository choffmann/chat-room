package chat

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/choffmann/chat-room/internal/model"
	"github.com/google/uuid"
)

const (
	RoomTimeout         = 3 * time.Hour
	RoomTimeoutInterval = 25 * time.Second
)

// timeNow is a variable for testing purposes
var timeNow = time.Now

type Room struct {
	id             uint
	hub            *Hub
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
	additionalInfo model.AdditionalInfo
	messagesMu     sync.RWMutex
	messages       []model.OutgoingMessage
	logger         *slog.Logger
}

func (r *Room) ID() uint                { return r.id }
func (r *Room) Shutdown() chan struct{} { return r.shutdown }
func (r *Room) Closed() chan struct{}   { return r.closed }

func (r *Room) ShutdownOnce(f func()) {
	r.shutdownOnce.Do(f)
}

func (r *Room) UpdateActivityNow() {
	r.activityMu.Lock()
	defer r.activityMu.Unlock()
	r.lastActivity = time.Now()
}

func (r *Room) UpdateAdditionalInfo(newInfo model.AdditionalInfo) {
	r.activityMu.Lock()
	defer r.activityMu.Unlock()
	r.additionalInfo = newInfo
}

func (r *Room) PatchAdditionalInfo(updates model.AdditionalInfo) {
	r.activityMu.Lock()
	defer r.activityMu.Unlock()
	if r.additionalInfo == nil {
		r.additionalInfo = make(model.AdditionalInfo)
	}
	for key, value := range updates {
		r.additionalInfo[key] = value
	}
}

func (r *Room) GetAdditionalInfo() model.AdditionalInfo {
	r.activityMu.RLock()
	defer r.activityMu.RUnlock()
	info := make(model.AdditionalInfo, len(r.additionalInfo))
	for k, v := range r.additionalInfo {
		info[k] = v
	}
	return info
}

func (r *Room) DisconnectAllClients() {
	r.clientsMu.Lock()
	defer r.clientsMu.Unlock()
	for c := range r.clients {
		c.CloseSend()
	}
}

func (r *Room) GetClientCount() int {
	r.clientsMu.RLock()
	defer r.clientsMu.RUnlock()
	return len(r.clients)
}

func (r *Room) GetUsers() []model.User {
	r.clientsMu.RLock()
	defer r.clientsMu.RUnlock()

	users := make([]model.User, 0, len(r.clients))
	for client := range r.clients {
		users = append(users, client.user)
	}
	return users
}

func (r *Room) TryBroadcast(msg []byte) bool {
	select {
	case r.broadcast <- msg:
		return true
	case <-r.shutdown:
		return false
	}
}

func (r *Room) TryRegister(c *Client) bool {
	select {
	case r.register <- c:
		return true
	case <-r.shutdown:
		return false
	}
}

func (r *Room) TryUnregister(c *Client) bool {
	select {
	case r.unregister <- c:
		return true
	case <-r.shutdown:
		return false
	}
}

func (r *Room) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		close(r.closed)
		cancel()
	}()

	go r.deleteRoomWithNoActivity(ctx)

	for {
		select {
		case <-r.shutdown:
			r.logger.Info("room shutdown signal received", "roomID", r.id)
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
				c.CloseSend()
			}
			r.clientsMu.Unlock()

		case msg := <-r.broadcast:
			r.UpdateActivityNow()
			r.clientsMu.RLock()
			clientsList := make([]*Client, 0, len(r.clients))
			for c := range r.clients {
				clientsList = append(clientsList, c)
			}
			r.clientsMu.RUnlock()

			failedClients := make([]*Client, 0)
			for _, c := range clientsList {
				select {
				case c.send <- msg:
				default:
					failedClients = append(failedClients, c)
				}
			}

			if len(failedClients) > 0 {
				r.clientsMu.Lock()
				for _, c := range failedClients {
					delete(r.clients, c)
					c.CloseSend()
				}
				r.clientsMu.Unlock()
			}
		}
	}
}

func (r *Room) deleteRoomWithNoActivity(ctx context.Context) {
	ticker := time.NewTicker(RoomTimeoutInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.activityMu.RLock()
			timeSinceActivity := time.Since(r.lastActivity)
			r.activityMu.RUnlock()

			if timeSinceActivity > RoomTimeout {
				r.shutdownOnce.Do(func() {
					close(r.shutdown)
				})
				r.DisconnectAllClients()
				r.hub.DeleteRoom(r.id)
				r.logger.Info("remove room due to timeout activity", "roomID", r.id)
				return
			}

		case <-ctx.Done():
			r.logger.Debug("stopping delete room scheduler")
			return
		}
	}
}

func (r *Room) StoreMessage(msg model.OutgoingMessage) {
	r.messagesMu.Lock()
	defer r.messagesMu.Unlock()
	if msg.AdditionalInfo == nil {
		msg.AdditionalInfo = make(model.AdditionalInfo)
	}
	r.messages = append(r.messages, msg)
}

func (r *Room) GetMessages() []model.OutgoingMessage {
	r.messagesMu.RLock()
	defer r.messagesMu.RUnlock()
	messages := make([]model.OutgoingMessage, len(r.messages))
	copy(messages, r.messages)
	return messages
}

func (r *Room) GetMessage(messageID uuid.UUID) (*model.OutgoingMessage, bool) {
	r.messagesMu.RLock()
	defer r.messagesMu.RUnlock()
	for _, msg := range r.messages {
		if msg.ID == messageID {
			return &msg, true
		}
	}
	return nil, false
}

func (r *Room) UpdateMessage(messageID uuid.UUID, newContent string, newAdditionalInfo model.AdditionalInfo) bool {
	r.messagesMu.Lock()
	defer r.messagesMu.Unlock()
	for i := range r.messages {
		if r.messages[i].ID == messageID {
			if r.messages[i].MessageType == model.SystemMessage {
				return false
			}

			r.messages[i].Message = newContent
			if newAdditionalInfo != nil {
				r.messages[i].AdditionalInfo = newAdditionalInfo
			}

			r.messages[i].AdditionalInfo["modified"] = true
			return true
		}
	}
	return false
}

func (r *Room) PatchMessage(messageID uuid.UUID, newContent *string, newAdditionalInfo model.AdditionalInfo) bool {
	r.messagesMu.Lock()
	defer r.messagesMu.Unlock()
	for i := range r.messages {
		if r.messages[i].ID == messageID {
			if r.messages[i].MessageType == model.SystemMessage {
				return false
			}
			if newContent != nil {
				r.messages[i].Message = *newContent
			}
			if newAdditionalInfo != nil {
				r.messages[i].AdditionalInfo = newAdditionalInfo
			}

			r.messages[i].AdditionalInfo["modified"] = true
			return true
		}
	}
	return false
}
