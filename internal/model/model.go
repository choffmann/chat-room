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
	ID             uuid.UUID      `json:"id" example:"9a6e58a5-4d47-4c86-8b3f-9ea373cbdb0c"`
	FirstName      string         `json:"firstName,omitempty" example:"John"`
	LastName       string         `json:"lastName,omitempty" example:"Doe"`
	Name           string         `json:"name,omitempty" example:"johndoe"`
	AdditionalInfo AdditionalInfo `json:"additionalInfo,omitempty" swaggertype:"object"`
}

type OutgoingMessage struct {
	ID             uuid.UUID      `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	MessageType    MessageType    `json:"type" example:"message"`
	Message        string         `json:"message" example:"Hello everyone!"`
	Timestamp      time.Time      `json:"timestamp" example:"2024-04-09T12:35:10.123456789Z"`
	User           User           `json:"user"`
	AdditionalInfo AdditionalInfo `json:"additionalInfo" swaggertype:"object"`
}

type IncomingMessage struct {
	MessageType    MessageType    `json:"type"`
	Message        string         `json:"message"`
	AdditionalInfo AdditionalInfo `json:"additionalInfo,omitempty"`
}

type RoomResponse struct {
	ID             uint           `json:"id" example:"1"`
	UserCount      int            `json:"onlineUser" example:"3"`
	AdditionalInfo AdditionalInfo `json:"additionalInfo,omitempty" swaggertype:"object"`
}

type CreateUserRequest struct {
	FirstName      string         `json:"firstName,omitempty" example:"John"`
	LastName       string         `json:"lastName,omitempty" example:"Doe"`
	Name           string         `json:"name,omitempty" example:"johndoe"`
	AdditionalInfo AdditionalInfo `json:"additionalInfo,omitempty" swaggertype:"object"`
}

type UpdateUserRequest struct {
	FirstName      string         `json:"firstName,omitempty" example:"Jane"`
	LastName       string         `json:"lastName,omitempty" example:"Smith"`
	Name           string         `json:"name,omitempty" example:"janesmith"`
	AdditionalInfo AdditionalInfo `json:"additionalInfo,omitempty" swaggertype:"object"`
}

type UserWithRoom struct {
	User   User `json:"user"`
	RoomID uint `json:"roomId" example:"1"`
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

// Non-storable types are transient or too large to keep in memory.
var nonStorableTypes = map[MessageType]struct{}{
	ImageMessage: {},
}

func ShouldStoreMessage(msgType MessageType) bool {
	_, excluded := nonStorableTypes[msgType]
	return msgType != "" && !excluded
}
