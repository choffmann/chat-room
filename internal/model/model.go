package model

import (
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type MessageType string

const (
	SystemMessage MessageType = "system"
	UserMessage   MessageType = "message"
	ImageMessage  MessageType = "image"
)

type AdditionalInfo = map[string]any

type User struct {
	ID             uuid.UUID      `json:"id"`
	FirstName      string         `json:"firstName,omitempty"`
	LastName       string         `json:"lastName,omitempty"`
	Name           string         `json:"name,omitempty"`
	AdditionalInfo AdditionalInfo `json:"additionalInfo,omitempty"`
}

type OutgoingMessage struct {
	ID             uuid.UUID      `json:"id"`
	MessageType    MessageType    `json:"type"`
	Message        string         `json:"message"`
	Timestamp      time.Time      `json:"timestamp"`
	User           User           `json:"user"`
	AdditionalInfo AdditionalInfo `json:"additionalInfo"`
}

type IncomingMessage struct {
	MessageType    MessageType    `json:"type"`
	Message        string         `json:"message"`
	AdditionalInfo AdditionalInfo `json:"additionalInfo,omitempty"`
}

type RoomResponse struct {
	ID             uint           `json:"id"`
	UserCount      int            `json:"onlineUser"`
	AdditionalInfo AdditionalInfo `json:"additionalInfo,omitempty"`
}

type CreateUserRequest struct {
	FirstName      string         `json:"firstName,omitempty"`
	LastName       string         `json:"lastName,omitempty"`
	Name           string         `json:"name,omitempty"`
	AdditionalInfo AdditionalInfo `json:"additionalInfo,omitempty"`
}

type UpdateUserRequest struct {
	FirstName      string         `json:"firstName,omitempty"`
	LastName       string         `json:"lastName,omitempty"`
	Name           string         `json:"name,omitempty"`
	AdditionalInfo AdditionalInfo `json:"additionalInfo,omitempty"`
}

type UserWithRoom struct {
	User   User `json:"user"`
	RoomID uint `json:"roomId"`
}

func GetDisplayName(user User) string {
	displayName := user.Name
	if displayName == "" && user.FirstName != "" && user.LastName != "" {
		return fmt.Sprintf("%s %s", user.FirstName, user.LastName)
	} else if displayName == "" && user.FirstName != "" {
		return user.FirstName
	} else if displayName == "" {
		return "Anonymous"
	}
	return displayName
}

func ParseRoomID(roomIDStr string) (uint, error) {
	roomID, err := strconv.ParseUint(roomIDStr, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(roomID), nil
}

func ShouldStoreMessage(msgType MessageType) bool {
	return msgType == SystemMessage || msgType == UserMessage
}
