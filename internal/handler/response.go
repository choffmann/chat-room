package handler

import "github.com/google/uuid"

// -- Swagger documentation types ------------------------------------------------
// These types mirror the actual runtime types but replace map[string]any
// (AdditionalInfo) with a concrete struct so Swagger UI renders useful examples.
// They are referenced only in swag annotations, never in handler code.

type RoomAdditionalInfoDoc struct {
	Theme       string `json:"theme,omitempty" example:"dark"`
	Description string `json:"description,omitempty" example:"General chat room"`
} // @name RoomAdditionalInfo

type UserAdditionalInfoDoc struct {
	Avatar string `json:"avatar,omitempty" example:"https://example.com/avatar.png"`
	Bio    string `json:"bio,omitempty" example:"Software developer"`
} // @name UserAdditionalInfo

type MessageAdditionalInfoDoc struct {
	Modified bool   `json:"modified,omitempty" example:"true"`
	Format   string `json:"format,omitempty" example:"markdown"`
} // @name MessageAdditionalInfo

// -- Request types --------------------------------------------------------------

type CreateRoomRequestDoc struct {
	Theme       string `json:"theme,omitempty" example:"dark"`
	Description string `json:"description,omitempty" example:"General chat room"`
} // @name CreateRoomRequest

type PatchRoomRequestDoc struct {
	Theme string `json:"theme,omitempty" example:"light"`
} // @name PatchRoomRequest

type PutRoomRequestDoc struct {
	Theme       string `json:"theme,omitempty" example:"dark"`
	Description string `json:"description,omitempty" example:"Updated chat room"`
} // @name PutRoomRequest

type MessagePatchRequestDoc struct {
	Message        *string                   `json:"message,omitempty" example:"Hello everyone! (edited)"`
	AdditionalInfo *MessageAdditionalInfoDoc `json:"additionalInfo,omitempty"`
} // @name MessagePatchRequest

type MessagePutRequestDoc struct {
	Message        string                    `json:"message" example:"Completely new message content"`
	AdditionalInfo *MessageAdditionalInfoDoc `json:"additionalInfo,omitempty"`
} // @name MessagePutRequest

type CreateUserRequestDoc struct {
	FirstName      string                 `json:"firstName,omitempty" example:"John"`
	LastName       string                 `json:"lastName,omitempty" example:"Doe"`
	Name           string                 `json:"name,omitempty" example:"johndoe"`
	AdditionalInfo *UserAdditionalInfoDoc `json:"additionalInfo,omitempty"`
} // @name CreateUserRequest

type UpdateUserRequestDoc struct {
	FirstName      string                 `json:"firstName,omitempty" example:"Jane"`
	LastName       string                 `json:"lastName,omitempty" example:"Smith"`
	Name           string                 `json:"name,omitempty" example:"janesmith"`
	AdditionalInfo *UserAdditionalInfoDoc `json:"additionalInfo,omitempty"`
} // @name UpdateUserRequest

type PatchUserRequestDoc struct {
	FirstName      string                 `json:"firstName,omitempty" example:"Jane"`
	Name           string                 `json:"name,omitempty" example:"janesmith"`
	AdditionalInfo *UserAdditionalInfoDoc `json:"additionalInfo,omitempty"`
} // @name PatchUserRequest

// -- Response types -------------------------------------------------------------

type CreateRoomResponse struct {
	RoomID uint `json:"roomID" example:"1"`
} // @name CreateRoomResponse

type UserDoc struct {
	ID             uuid.UUID              `json:"id" example:"9a6e58a5-4d47-4c86-8b3f-9ea373cbdb0c"`
	FirstName      string                 `json:"firstName,omitempty" example:"John"`
	LastName       string                 `json:"lastName,omitempty" example:"Doe"`
	Name           string                 `json:"name,omitempty" example:"johndoe"`
	AdditionalInfo *UserAdditionalInfoDoc `json:"additionalInfo,omitempty"`
} // @name User

type OutgoingMessageDoc struct {
	ID             uuid.UUID                 `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	MessageType    string                    `json:"type" example:"message"`
	Message        string                    `json:"message" example:"Hello everyone!"`
	Timestamp      string                    `json:"timestamp" example:"2024-04-09T12:35:10.123456789Z"`
	User           UserDoc                   `json:"user"`
	AdditionalInfo *MessageAdditionalInfoDoc `json:"additionalInfo"`
} // @name OutgoingMessage

type RoomResponseDoc struct {
	ID             uint                   `json:"id" example:"1"`
	UserCount      int                    `json:"onlineUser" example:"3"`
	AdditionalInfo *RoomAdditionalInfoDoc `json:"additionalInfo,omitempty"`
} // @name RoomResponse

type UserWithRoomDoc struct {
	User   UserDoc `json:"user"`
	RoomID uint    `json:"roomId" example:"1"`
} // @name UserWithRoom

type RoomsListResponse struct {
	Rooms []RoomResponseDoc `json:"rooms"`
} // @name RoomsListResponse

type MessagesListResponse struct {
	Messages []OutgoingMessageDoc `json:"messages"`
} // @name MessagesListResponse

type UsersListResponse struct {
	Users []UserDoc `json:"users"`
} // @name UsersListResponse

type UsersWithRoomListResponse struct {
	Users []UserWithRoomDoc `json:"users"`
} // @name UsersWithRoomListResponse
