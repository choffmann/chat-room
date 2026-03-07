package chat

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/choffmann/chat-room/internal/model"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	B   = 1
	KiB = 1024 * B
	MiB = 1024 * KiB
)

type Client struct {
	room         *Room
	conn         *websocket.Conn
	user         model.User
	send         chan []byte
	closeMu      sync.Mutex
	closed       bool
	disconnected sync.Once
	systemUser   model.User
	logger       *slog.Logger
}

func NewClient(room *Room, conn *websocket.Conn, user model.User, systemUser model.User, logger *slog.Logger) *Client {
	return &Client{
		room:       room,
		conn:       conn,
		user:       user,
		send:       make(chan []byte, 256),
		systemUser: systemUser,
		logger:     logger,
	}
}

func (c *Client) User() model.User  { return c.user }
func (c *Client) Send() chan []byte { return c.send }
func (c *Client) Room() *Room       { return c.room }

func (c *Client) CloseSend() {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()
	if !c.closed {
		close(c.send)
		c.closed = true
	}
}

func (c *Client) Disconnect() {
	c.disconnected.Do(func() {
		displayName := model.GetDisplayName(c.user)
		timestamp := time.Now()

		leaveMsg := model.OutgoingMessage{
			ID:          uuid.New(),
			MessageType: model.SystemMessage,
			Message:     fmt.Sprintf("%s left room %d", displayName, c.room.id),
			Timestamp:   timestamp,
			User:        c.systemUser,
		}

		c.room.StoreMessage(leaveMsg)

		b, _ := json.Marshal(leaveMsg)
		if !c.room.TryBroadcast(b) {
			c.logger.Debug("failed to broadcast leave message, room may be closing", "roomID", c.room.id)
		}

		if !c.room.TryUnregister(c) {
			c.logger.Debug("failed to unregister client, room may be closing", "roomID", c.room.id, "userID", c.user.ID)
		}
	})
}

func (c *Client) ReadPump() {
	defer func() {
		c.Disconnect()
		c.conn.Close()
	}()

	c.conn.SetReadLimit(10 * MiB)
	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var message model.IncomingMessage
		if err := c.conn.ReadJSON(&message); err != nil {
			if !websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) && !strings.Contains(err.Error(), "use of closed network connection") {
				c.logger.Warn("websocket read failed", "roomID", c.room.id, "userID", c.user.ID, "error", err)
			}
			break
		}

		if message.MessageType == "" {
			message.MessageType = model.UserMessage
		}

		timestamp := time.Now()

		payload := model.OutgoingMessage{
			ID:             uuid.New(),
			MessageType:    message.MessageType,
			Message:        message.Message,
			Timestamp:      timestamp,
			User:           c.user,
			AdditionalInfo: message.AdditionalInfo,
		}

		b, _ := json.Marshal(payload)
		if !c.room.TryBroadcast(b) {
			c.logger.Warn("failed to broadcast message, room may be closing", "roomID", c.room.id, "userID", c.user.ID)
			break
		}

		if model.ShouldStoreMessage(message.MessageType) && len(b) < 2*MiB && len(b) > 0 {
			c.room.StoreMessage(payload)
		}

		c.logger.Info("new message received", "roomID", c.room.id, "userID", c.user.ID, "messageID", payload.ID, "messageType", payload.MessageType)
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Disconnect()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				c.logger.Warn("failed to write websocket message", "roomID", c.room.id, "userID", c.user.ID, "error", err)
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.logger.Warn("failed to send websocket ping", "roomID", c.room.id, "userID", c.user.ID, "error", err)
				return
			}
		}
	}
}
