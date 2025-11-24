# Chat Room - Real-time Chat Backend

A lightweight real-time chat backend written in Go. The server manages ephemeral chat rooms where clients connect via WebSocket to exchange messages. Rooms are created on-demand and automatically deleted after 3 hours of inactivity.

## REST Endpoints

### Create a Room — `POST /rooms`

Creates a new chat room. The request body is optional and can carry additional metadata that will be echoed back when the room is queried.

#### Request

```http
POST /rooms HTTP/1.1
Content-Type: application/json

{
  "title": "Mobile Computing Lecture",
  "topic": "Android Architecture Components"
}
```

- Body: arbitrary JSON object (may be empty). Values are stored as-is under `additionalInfo`.
- If the JSON payload cannot be decoded, an empty `additionalInfo` is used instead.

#### Successful Response — `200 OK`

```json
{
  "roomID": 1
}
```

### List Rooms — `GET /rooms`

Retrieves all currently active rooms.

#### Successful Response — `200 OK`

```json
{
  "rooms": [
    {
      "id": 1,
      "onlineUser": 3,
      "additionalInfo": {
        "title": "Mobile Computing Lecture",
        "topic": "Android Architecture Components"
      }
    }
  ]
}
```

### Get Room Details — `GET /rooms/{roomID}`

Returns metadata for a specific room.

- `roomID`: numeric path parameter.

#### Successful Response — `200 OK`

```json
{
  "id": 1,
  "additionalInfo": {
    "title": "Mobile Computing Lecture",
    "topic": "Android Architecture Components"
  }
}
```

#### Error Responses

- `400 Bad Request` if `roomID` is not a positive integer.
- `404 Not Found` if the room does not exist (either never created or already removed).

### Update Room Metadata (Partial) — `PATCH /rooms/{roomID}`

Partially updates room metadata. The provided fields are merged with existing `additionalInfo`, preserving fields not included in the request.

- `roomID`: numeric path parameter.

#### Request

```http
PATCH /rooms/1 HTTP/1.1
Content-Type: application/json

{
  "topic": "iOS Frameworks & SwiftUI"
}
```

#### Successful Response — `200 OK`

```json
{
  "id": 1,
  "additionalInfo": {
    "title": "Mobile Computing Lecture",
    "topic": "iOS Frameworks & SwiftUI"
  }
}
```

#### Error Responses

- `400 Bad Request` if `roomID` is invalid or JSON payload is malformed.
- `404 Not Found` if the room does not exist.

### Replace Room Metadata — `PUT /rooms/{roomID}`

Replaces all room metadata. This completely overwrites the existing `additionalInfo`.

- `roomID`: numeric path parameter.

#### Request

```http
PUT /rooms/1 HTTP/1.1
Content-Type: application/json

{
  "title": "Cross-Platform Development",
  "topic": "Flutter & React Native Comparison",
  "lecturer": "Prof. Dr. Smith"
}
```

#### Successful Response — `200 OK`

```json
{
  "id": 1,
  "additionalInfo": {
    "title": "Cross-Platform Development",
    "topic": "Flutter & React Native Comparison",
    "lecturer": "Prof. Dr. Smith"
  }
}
```

#### Error Responses

- `400 Bad Request` if `roomID` is invalid or JSON payload is malformed.
- `404 Not Found` if the room does not exist.

### Get Room Messages — `GET /rooms/{roomID}/messages`

Returns all messages that have been sent in a specific room. Messages are stored in memory and include system messages (joins/leaves) as well as user messages.

**Note:** Only messages smaller than 2 MiB (after JSON serialization) are stored. Larger messages are broadcast in real-time but not persisted. If a Message is empty, it will also not be stored

- `roomID`: numeric path parameter.

#### Successful Response — `200 OK`

```json
{
  "messages": [
    {
      "id": 1,
      "type": "system",
      "message": "Alice joined room 1",
      "timestamp": "2024-04-09T12:34:56.789012345Z",
      "user": {
        "id": "dd0a6c0c-7b01-47d4-8b3a-296774a0930c",
        "name": "system"
      }
    },
    {
      "id": 2,
      "type": "message",
      "message": "Hello everyone!",
      "timestamp": "2024-04-09T12:35:10.123456789Z",
      "user": {
        "id": "9a6e58a5-4d47-4c86-8b3f-9ea373cbdb0c",
        "name": "Alice"
      },
      "additionalInfo": {
        "replyTo": "msg-456"
      }
    }
  ]
}
```

#### Error Responses

- `400 Bad Request` if `roomID` is not a positive integer.
- `404 Not Found` if the room does not exist.

### Get Single Message — `GET /rooms/{roomID}/messages/{messageID}`

Retrieves a specific message from a room by its ID.

- `roomID`: numeric path parameter (room identifier).
- `messageID`: UUID path parameter (message identifier).

#### Successful Response — `200 OK`

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "type": "message",
  "message": "Hello everyone!",
  "timestamp": "2024-04-09T12:35:10.123456789Z",
  "user": {
    "id": "9a6e58a5-4d47-4c86-8b3f-9ea373cbdb0c",
    "name": "Alice"
  },
  "additionalInfo": {
    "replyTo": "msg-456"
  }
}
```

#### Error Responses

- `400 Bad Request` if `roomID` or `messageID` is invalid.
- `404 Not Found` if the room or message does not exist.

### Update Message (Partial) — `PATCH /rooms/{roomID}/messages/{messageID}`

Partially updates a specific message in a room. You can update the message text, additionalInfo, or both. Only provided fields are updated; others remain unchanged.

**Note:** The server automatically sets `modified: true` in the message's `additionalInfo`.

- `roomID`: numeric path parameter (room identifier).
- `messageID`: UUID path parameter (message identifier).

#### Request Examples

**Update only the message text:**

```http
PATCH /rooms/1/messages/550e8400-e29b-41d4-a716-446655440000 HTTP/1.1
Content-Type: application/json

{
  "message": "Hello everyone! (edited)"
}
```

**Update only additionalInfo:**

```http
PATCH /rooms/1/messages/550e8400-e29b-41d4-a716-446655440000 HTTP/1.1
Content-Type: application/json

{
  "additionalInfo": {
    "edited": true,
    "editTimestamp": "2024-04-09T12:36:00.000000000Z"
  }
}
```

**Update both message and additionalInfo:**

```http
PATCH /rooms/1/messages/550e8400-e29b-41d4-a716-446655440000 HTTP/1.1
Content-Type: application/json

{
  "message": "Hello everyone! (edited)",
  "additionalInfo": {
    "editReason": "Fixed typo"
  }
}
```

#### Successful Response — `200 OK`

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "type": "message",
  "message": "Hello everyone! (edited)",
  "timestamp": "2024-04-09T12:35:10.123456789Z",
  "user": {
    "id": "9a6e58a5-4d47-4c86-8b3f-9ea373cbdb0c",
    "name": "Alice"
  },
  "additionalInfo": {
    "modified": true,
    "editReason": "Fixed typo"
  }
}
```

#### Error Responses

- `400 Bad Request` if:
  - `roomID` or `messageID` is invalid
  - No fields are provided in the request body
  - `message` field is provided but empty
  - JSON payload is malformed
- `404 Not Found` if the room or message does not exist.

### Replace Message — `PUT /rooms/{roomID}/messages/{messageID}`

Completely replaces a message in a room. Unlike PATCH, this requires all fields and replaces the entire message content.

**Note:** The server automatically sets `modified: true` in the message's `additionalInfo`.

- `roomID`: numeric path parameter (room identifier).
- `messageID`: UUID path parameter (message identifier).

#### Request

```http
PUT /rooms/1/messages/550e8400-e29b-41d4-a716-446655440000 HTTP/1.1
Content-Type: application/json

{
  "message": "Completely new message content",
  "additionalInfo": {
    "version": 2
  }
}
```

- `message`: Required. New message content.
- `additionalInfo`: Optional. New metadata (replaces existing).

#### Successful Response — `200 OK`

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "type": "message",
  "message": "Completely new message content",
  "timestamp": "2024-04-09T12:35:10.123456789Z",
  "user": {
    "id": "9a6e58a5-4d47-4c86-8b3f-9ea373cbdb0c",
    "name": "Alice"
  },
  "additionalInfo": {
    "modified": true,
    "version": 2
  }
}
```

The updated message is automatically broadcast to all connected clients.

#### Error Responses

- `400 Bad Request` if `roomID` or `messageID` is invalid or JSON payload is malformed.
- `404 Not Found` if the room or message does not exist.

### Delete Message — `DELETE /rooms/{roomID}/messages/{messageID}`

Marks a message as deleted. The message is not actually removed but its content is replaced with "deleted" and a deleted flag is added to additionalInfo.

- `roomID`: numeric path parameter (room identifier).
- `messageID`: UUID path parameter (message identifier).

#### Successful Response — `200 OK`

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "type": "message",
  "message": "deleted",
  "timestamp": "2024-04-09T12:35:10.123456789Z",
  "user": {
    "id": "9a6e58a5-4d47-4c86-8b3f-9ea373cbdb0c",
    "name": "Alice"
  },
  "additionalInfo": {
    "deleted": true
  }
}
```

The deleted message is automatically broadcast to all connected clients.

#### Error Responses

- `400 Bad Request` if `roomID` or `messageID` is invalid.
- `404 Not Found` if the room or message does not exist.

---

## User Management Endpoints

### Create User — `POST /users`

Creates a new user in the user registry. This allows pre-registering users before they join rooms.

#### Request

```http
POST /users HTTP/1.1
Content-Type: application/json

{
  "firstName": "John",
  "lastName": "Doe",
  "name": "johndoe",
  "additionalInfo": {
    "avatar": "https://example.com/avatar.jpg",
    "role": "student",
    "semester": 5
  }
}
```

- All fields are optional
- `additionalInfo`: arbitrary JSON metadata

#### Successful Response — `201 Created`

```json
{
  "id": "9a6e58a5-4d47-4c86-8b3f-9ea373cbdb0c",
  "firstName": "John",
  "lastName": "Doe",
  "name": "johndoe",
  "additionalInfo": {
    "avatar": "https://example.com/avatar.jpg",
    "role": "student",
    "semester": 5
  }
}
```

#### Error Responses

- `400 Bad Request` if the JSON payload is malformed.

### Get All Users — `GET /users`

Returns all users registered in the user registry.

#### Successful Response — `200 OK`

```json
[
  {
    "id": "9a6e58a5-4d47-4c86-8b3f-9ea373cbdb0c",
    "firstName": "John",
    "lastName": "Doe",
    "name": "johndoe",
    "additionalInfo": {
      "avatar": "https://example.com/avatar.jpg",
      "role": "student",
      "semester": 5
    }
  },
  {
    "id": "dd0a6c0c-7b01-47d4-8b3a-296774a0930c",
    "firstName": "Jane",
    "lastName": "Smith",
    "name": "janesmith"
  }
]
```

- Returns an empty array `[]` if no users are registered.

### Update User (Full Replace) — `PUT /users/{userID}`

Completely replaces all user information. Fields not included will be cleared.

- `userID`: UUID path parameter.

#### Request

```http
PUT /users/9a6e58a5-4d47-4c86-8b3f-9ea373cbdb0c HTTP/1.1
Content-Type: application/json

{
  "firstName": "Jane",
  "lastName": "Smith",
  "name": "janesmith"
}
```

#### Successful Response — `200 OK`

```json
{
  "id": "9a6e58a5-4d47-4c86-8b3f-9ea373cbdb0c",
  "firstName": "Jane",
  "lastName": "Smith",
  "name": "janesmith"
}
```

#### Error Responses

- `400 Bad Request` if `userID` is not a valid UUID or JSON payload is malformed.
- `404 Not Found` if the user does not exist.

### Update User (Partial) — `PATCH /users/{userID}`

Partially updates user information. Only provided fields are updated, others remain unchanged.

- `userID`: UUID path parameter.

#### Request

```http
PATCH /users/9a6e58a5-4d47-4c86-8b3f-9ea373cbdb0c HTTP/1.1
Content-Type: application/json

{
  "name": "john_doe_updated"
}
```

#### Successful Response — `200 OK`

```json
{
  "id": "9a6e58a5-4d47-4c86-8b3f-9ea373cbdb0c",
  "firstName": "John",
  "lastName": "Doe",
  "name": "john_doe_updated",
  "additionalInfo": {
    "avatar": "https://example.com/avatar.jpg",
    "role": "student",
    "semester": 5
  }
}
```

#### Error Responses

- `400 Bad Request` if `userID` is not a valid UUID or JSON payload is malformed.
- `404 Not Found` if the user does not exist.

### Delete User — `DELETE /users/{userID}`

Deletes a user from the user registry.

- `userID`: UUID path parameter.

#### Successful Response — `204 No Content`

No response body.

#### Error Responses

- `400 Bad Request` if `userID` is not a valid UUID.
- `404 Not Found` if the user does not exist.

### Get Room Users — `GET /rooms/{roomID}/users`

Returns all users currently connected to a specific room.

- `roomID`: numeric path parameter.

#### Successful Response — `200 OK`

```json
{
  "users": [
    {
      "id": "9a6e58a5-4d47-4c86-8b3f-9ea373cbdb0c",
      "firstName": "John",
      "lastName": "Doe",
      "name": "johndoe",
      "additionalInfo": {
        "avatar": "https://example.com/avatar.jpg"
      }
    },
    {
      "id": "dd0a6c0c-7b01-47d4-8b3a-296774a0930c",
      "name": "Alice"
    }
  ]
}
```

#### Error Responses

- `400 Bad Request` if `roomID` is not a positive integer.
- `404 Not Found` if the room does not exist.

### Get All Users in Rooms — `GET /rooms/users`

Returns all users currently connected to any room, along with their room IDs.

#### Successful Response — `200 OK`

```json
{
  "users": [
    {
      "user": {
        "id": "9a6e58a5-4d47-4c86-8b3f-9ea373cbdb0c",
        "firstName": "John",
        "lastName": "Doe",
        "name": "johndoe"
      },
      "roomId": 1
    },
    {
      "user": {
        "id": "dd0a6c0c-7b01-47d4-8b3a-296774a0930c",
        "name": "Alice"
      },
      "roomId": 2
    }
  ]
}
```

---

### Join Room via WebSocket — `GET /join/{roomID}`

Upgrades the HTTP connection to WebSocket and joins the requested room.

- `roomID`: numeric path parameter.

#### Query Parameters

**Option 1: Join as registered user**

- `userId`: UUID of a user registered via `POST /users`. If provided, the server loads the full user profile from the registry.

**Option 2: Join as ephemeral user (fallback)**

- `user`: Display name for an ephemeral (temporary) user. If omitted, the server assigns a random display name.

**Priority:** If `userId` is provided, it takes precedence over `user`. The `user` parameter is only used when `userId` is absent.

#### Examples

```
GET /join/1?userId=9a6e58a5-4d47-4c86-8b3f-9ea373cbdb0c
```

Joins room 1 using registered user profile.

```
GET /join/1?user=Alice
```

Joins room 1 as ephemeral user "Alice".

```
GET /join/1
```

Joins room 1 with random display name (e.g., "Toni Tester", "Harald Hüftschmerz").

#### WebSocket Message Flow

**1. System Join Notification**

Upon joining, the server broadcasts a system message to all connected clients:

```json
{
  "type": "system",
  "message": "Alice joined room 1",
  "timestamp": "2024-04-09T12:34:56.789012345Z",
  "user": {
    "id": "dd0a6c0c-7b01-47d4-8b3a-296774a0930c",
    "name": "system"
  }
}
```

**2. Sending Messages (Client → Server)**

Clients send chat messages as JSON:

```json
{
  "type": "message",
  "message": "Can someone explain the difference between Jetpack Compose and XML layouts?",
  "additionalInfo": {
    "replyTo": "msg-456",
    "category": "question"
  }
}
```

- `type`: Message type (`"message"` or `"image"`)
- `message`: Message content (text or image URL)
- `additionalInfo`: Optional metadata (arbitrary JSON)

**3. Receiving Messages (Server → Client)**

The server wraps messages with timestamp and user info, then broadcasts to all participants:

```json
{
  "type": "message",
  "message": "Can someone explain the difference between Jetpack Compose and XML layouts?",
  "timestamp": "2024-04-09T12:35:10.123456789Z",
  "user": {
    "id": "9a6e58a5-4d47-4c86-8b3f-9ea373cbdb0c",
    "name": "Alice"
  },
  "additionalInfo": {
    "replyTo": "msg-456",
    "category": "question"
  }
}
```

**4. System Leave Notification**

When a client disconnects, the server broadcasts:

```json
{
  "type": "system",
  "message": "Alice left room 1",
  "timestamp": "2024-04-09T12:40:00.123456789Z",
  "user": {
    "id": "dd0a6c0c-7b01-47d4-8b3a-296774a0930c",
    "name": "system"
  }
}
```

**Connection Management**

- Server sends ping frames every 30 seconds
- Client must respond with pong within 60 seconds
- Maximum message size: 10 MiB
- Write timeout: 10 seconds

**Message Persistence**

- Messages are broadcast to all connected clients in real-time regardless of size
- Only `system` and `message` type messages are stored in message history
- Messages larger than 2 MiB (after JSON serialization) are NOT stored, but still broadcast
- Other message types (`image`, custom events) are ephemeral and never stored

#### Error Responses

- `400 Bad Request` if:
  - `roomID` cannot be parsed
  - `userId` is provided but not a valid UUID
- `404 Not Found` if:
  - The room does not exist
  - `userId` is provided but user not found in registry
- Standard WebSocket close frames for protocol errors or disconnects.

### Build Information — `GET /info`

Exposes metadata about the running binary.

#### Successful Response — `200 OK`

```json
{
  "version": "unknown",
  "commit": "unknown",
  "branch": "unknown",
  "repository": "unknown",
  "build_time": "2024-04-09T12:45:00Z"
}
```

- Field values are populated at build time; when unavailable, they default to `"unknown"`.

### Health Check — `GET /healthz`

Simple liveness probe.

#### Successful Response — `200 OK`

Plain-text body: `OK`

## Message Schema Reference

### Outgoing Messages (Client → Server)

| Field            | Type   | Required | Description                                                      |
| ---------------- | ------ | -------- | ---------------------------------------------------------------- |
| `type`           | string | Yes      | Message type: `"message"`, `"image"`                             |
| `message`        | string | Yes      | Message content or image as base64                               |
| `additionalInfo` | object | No       | Arbitrary metadata (client-defined structure)                    |

### Incoming Messages (Server → Client)

| Field            | Type    | Description                                                      |
| ---------------- | ------- | ---------------------------------------------------------------- |
| `id`             | UUID    | Unique message identifier                                        |
| `type`           | string  | Message or event type (see below)                                |
| `message`        | string  | Message body or system notification text                         |
| `timestamp`      | RFC3339 | Server-generated timestamp                                       |
| `user.id`        | UUID    | Unique user identifier for this connection                       |
| `user.name`      | string  | Display name (or random default if not provided)                 |
| `additionalInfo` | object  | Optional metadata passed through from client messages            |

### Message Types

The server defines three message types:

**Persistent Types** (stored in message history if < 2 MiB):

- `"system"` - System notifications (user joined/left)
- `"message"` - Regular text messages

**Ephemeral Types** (broadcast only, not stored):

- `"image"` - Image messages (Base64-encoded)

## Message Behavior

### Storage Rules

- **System messages** (`"system"`): Stored if < 2 MiB
- **User messages** (`"message"`): Stored if < 2 MiB
- **Image messages** (`"image"`): Never stored, only broadcast in real-time

### Broadcasting

All message types are broadcast to all connected clients in real-time, regardless of whether they are stored or not (up to 10 MiB size limit).

## Room Lifecycle

### Creation

- Rooms are created via `POST /rooms`
- Each room gets an auto-incrementing numeric ID
- Optional metadata can be attached at creation

### Activity Tracking

Room activity is updated on:

- Client joins the room
- Message is broadcast

### Auto-Deletion

Rooms are automatically deleted when:

- **3 hours of inactivity** (no joins or messages)

### Shutdown Behavior

When a room shuts down:

1. Broadcasting stops immediately
2. All connected clients are disconnected
3. Room is removed from the global registry
4. All room data and metadata is discarded
