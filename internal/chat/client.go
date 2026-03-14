package chat

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/choffmann/chat-room/internal/model"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type UploadStore interface {
	Save(roomID uint, data []byte) (string, error)
}

const (
	B   = 1
	KiB = 1024 * B
	MiB = 1024 * KiB
)

const maxUploadSize = 5 * MiB

type Client struct {
	room          *Room
	conn          *websocket.Conn
	user          model.User
	send          chan []byte
	closeMu       sync.Mutex
	closed        bool
	disconnected  sync.Once
	systemUser    model.User
	uploadStore   UploadStore
	uploadBaseURL string
	logger        *slog.Logger
}

func NewClient(room *Room, conn *websocket.Conn, user model.User, systemUser model.User, logger *slog.Logger, uploadStore UploadStore, uploadBaseURL string) *Client {
	return &Client{
		room:          room,
		conn:          conn,
		user:          user,
		send:          make(chan []byte, 256),
		systemUser:    systemUser,
		uploadStore:   uploadStore,
		uploadBaseURL: uploadBaseURL,
		logger:        logger,
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
		msgType, data, err := c.conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) && !strings.Contains(err.Error(), "use of closed network connection") {
				c.logger.Warn("websocket read failed", "roomID", c.room.id, "userID", c.user.ID, "error", err)
			}
			break
		}

		switch msgType {
		case websocket.TextMessage:
			if !c.handleTextMessage(data) {
				return
			}
		case websocket.BinaryMessage:
			if !c.handleBinaryMessage(data) {
				return
			}
		}
	}
}

func (c *Client) handleTextMessage(data []byte) bool {
	var message model.IncomingMessage
	if err := json.Unmarshal(data, &message); err != nil {
		c.logger.Warn("invalid JSON from client", "roomID", c.room.id, "userID", c.user.ID, "error", err)
		return true
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
		return false
	}

	if model.ShouldStoreMessage(message.MessageType) && len(b) < 2*MiB && len(b) > 0 {
		c.room.StoreMessage(payload)
	}

	c.logger.Info("new message received", "roomID", c.room.id, "userID", c.user.ID, "messageID", payload.ID, "messageType", payload.MessageType)
	return true
}

func (c *Client) sendError(errMsg string) {
	payload := model.OutgoingMessage{
		ID:          uuid.New(),
		MessageType: model.SystemMessage,
		Message:     errMsg,
		Timestamp:   time.Now(),
		User:        c.systemUser,
		AdditionalInfo: model.AdditionalInfo{
			"error": true,
		},
	}
	b, _ := json.Marshal(payload)
	select {
	case c.send <- b:
	default:
		c.logger.Warn("failed to send error to client, channel full", "roomID", c.room.id, "userID", c.user.ID)
	}
}

func (c *Client) handleBinaryMessage(data []byte) bool {
	if c.uploadStore == nil {
		c.logger.Warn("binary message received but uploads are disabled", "roomID", c.room.id, "userID", c.user.ID)
		c.sendError("uploads are disabled")
		return true
	}

	if len(data) > maxUploadSize {
		c.logger.Warn("upload too large", "roomID", c.room.id, "userID", c.user.ID, "size", len(data), "max", maxUploadSize)
		c.sendError(fmt.Sprintf("upload too large: %d bytes exceeds limit of %d bytes", len(data), maxUploadSize))
		return true
	}

	if len(data) == 0 {
		c.sendError("empty upload")
		return true
	}

	relPath, err := c.uploadStore.Save(c.room.id, data)
	if err != nil {
		c.logger.Error("failed to save upload", "roomID", c.room.id, "userID", c.user.ID, "error", err)
		c.sendError("failed to save upload")
		return true
	}

	contentType := http.DetectContentType(data)
	msgType := model.MessageType("file")
	if strings.HasPrefix(contentType, "image/") {
		msgType = model.ImageMessage
	}

	fileURL := c.uploadBaseURL + "/" + relPath

	payload := model.OutgoingMessage{
		ID:          uuid.New(),
		MessageType: msgType,
		Message:     fileURL,
		Timestamp:   time.Now(),
		User:        c.user,
		AdditionalInfo: model.AdditionalInfo{
			"contentType": contentType,
			"size":        len(data),
			"fileName":    filepath.Base(relPath),
		},
	}

	b, _ := json.Marshal(payload)
	if !c.room.TryBroadcast(b) {
		c.logger.Warn("failed to broadcast upload notification, room may be closing", "roomID", c.room.id, "userID", c.user.ID)
		return false
	}

	c.room.StoreMessage(payload)
	c.logger.Info("binary upload received", "roomID", c.room.id, "userID", c.user.ID, "messageID", payload.ID, "url", fileURL, "contentType", contentType, "size", len(data))
	return true
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
