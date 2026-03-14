# Chat Room - Real-time Chat Backend

A lightweight real-time chat backend written in Go. The server manages ephemeral chat rooms where clients connect via WebSocket to exchange messages. Rooms are created on-demand and automatically deleted after 3 hours of inactivity.

## Quick Start

### Local Development

```bash
go run ./cmd/chat-room
```

The server starts on `:8080` by default. API documentation is available at [/api/v1/swagger/](http://localhost:8080/api/v1/swagger/).

### Docker

```bash
docker build -t chat-room .
docker run -p 8080:8080 chat-room
```

## Configuration

| Variable | Description | Default |
|---|---|---|
| `LOG_LEVEL` | Logging level (`debug`, `info`, `warn`, `error`) | `info` |
| `LOG_FORMAT` | Log format (`text`, `json`) | `text` |
| `BASE_URL` | Host for Swagger UI and upload URLs (e.g. `example.com:8080`) | _(auto)_ |
| `LEGACY_ROUTES` | Enable unversioned legacy routes | `true` |
| `UPLOAD_DIR` | Directory for binary file uploads | `./uploads` |

## API Overview

All endpoints are under `/api/v1`. Full request/response documentation is available via the **Swagger UI** at `/api/v1/swagger/`.

> **Note:** The server does not implement authentication or authorization. All endpoints and WebSocket connections are publicly accessible. This is by design — the server focuses on ephemeral, lightweight communication. Rooms are short-lived (auto-deleted after 3 hours of inactivity), and no sensitive data is persisted.

| Area | Endpoints |
|---|---|
| **Rooms** | `POST /rooms`, `GET /rooms`, `GET /rooms/{id}`, `PATCH /rooms/{id}`, `PUT /rooms/{id}` |
| **Messages** | `GET /rooms/{id}/messages`, `GET/PATCH/PUT/DELETE /rooms/{id}/messages/{msgID}` |
| **Users** | `POST /users`, `GET /users`, `GET/PUT/PATCH/DELETE /users/{id}` |
| **Room Users** | `GET /rooms/{id}/users`, `GET /rooms/users` |
| **WebSocket** | `GET /join/{id}?userId=<uuid>` or `?user=<name>` |
| **System** | `GET /info`, `GET /healthz` |

## WebSocket

Connect via `GET /api/v1/join/{roomID}` to join a room. Query parameters:

- `userId=<uuid>` - Join as a registered user
- `user=<name>` - Join as an ephemeral user (random name if omitted)
- `userInfo=true` - Receive a self-addressed join message containing assigned user info

### Message Format

**Client -> Server:**

| Field | Type | Required | Description |
|---|---|---|---|
| `type` | string | No | Any string. Defaults to `"message"` if omitted. |
| `message` | string | Yes | Text content or Base64-encoded image |
| `additionalInfo` | object | No | Arbitrary JSON metadata (see [additionalInfo](#additionalinfo)) |

```json
{
  "type": "message",
  "message": "Hello!",
  "additionalInfo": {
    "replyTo": "550e8400-e29b-41d4-a716-446655440000",
    "priority": "high"
  }
}
```

Custom types work the same way:

```json
{
  "type": "poll",
  "message": "What should we do?",
  "additionalInfo": {
    "options": ["Option A", "Option B"],
    "multiSelect": false
  }
}
```

**Server -> Client:**

The server wraps the message with a unique ID, timestamp, and user info, then broadcasts it to all room participants:

```json
{
  "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "type": "message",
  "message": "Hello!",
  "timestamp": "2024-04-09T12:35:10.123456789Z",
  "user": {
    "id": "9a6e58a5-4d47-4c86-8b3f-9ea373cbdb0c",
    "name": "Alice"
  },
  "additionalInfo": {
    "replyTo": "550e8400-e29b-41d4-a716-446655440000",
    "priority": "high"
  }
}
```

### Binary File Upload

Clients can send binary WebSocket frames to upload files directly. The server saves the file, detects its MIME type, and broadcasts a JSON message with the download URL to all room participants.

- **Max upload size:** 5 MiB
- **Supported types:** Images (`.jpg`, `.png`, `.gif`, `.webp`, `.svg`), documents (`.pdf`), archives (`.zip`, `.gz`), audio (`.mp3`, `.ogg`), video (`.mp4`, `.webm`), and generic binary (`.bin` fallback)
- **Download URL:** Files are served at `/uploads/{roomID}/{uuid}.{ext}`

**Server -> Client (broadcast on upload):**

```json
{
  "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "type": "image",
  "message": "http://localhost:8080/uploads/1/a1b2c3d4-e5f6-7890-abcd-ef1234567890.png",
  "timestamp": "2024-04-09T12:35:10.123456789Z",
  "user": {
    "id": "9a6e58a5-4d47-4c86-8b3f-9ea373cbdb0c",
    "name": "Alice"
  },
  "additionalInfo": {
    "contentType": "image/png",
    "size": 204800,
    "fileName": "a1b2c3d4-e5f6-7890-abcd-ef1234567890.png"
  }
}
```

The `type` is `"image"` for image files and `"file"` for all other types. On errors (uploads disabled, file too large, empty payload, disk error), the server sends an error message only to the sending client with `"error": true` in `additionalInfo`.

Upload files are automatically cleaned up when a room is deleted or the server shuts down.

### Message Types

The `type` field accepts any string value, allowing clients to define custom message types without server-side changes. If omitted, the type defaults to `"message"`.

| Type | Stored | Description |
|---|---|---|
| `system` | Yes (< 2 MiB) | Join/leave notifications (server-generated, not sendable by clients) |
| `message` | Yes (< 2 MiB) | Text messages (default if `type` is omitted) |
| `image` | Yes | Image uploads via binary WebSocket frames |
| `file` | Yes (< 2 MiB) | Non-image binary uploads |
| _custom_ | Yes (< 2 MiB) | Any other string (e.g. `"poll"`, `"reaction"`) |

## `additionalInfo`

Most entities (rooms, messages, users) support an `additionalInfo` field. This is a free-form JSON object that the server stores and returns as-is, without validation or schema enforcement. It allows clients to attach arbitrary metadata without requiring server-side changes.

**Examples by entity:**

| Entity | Example use cases |
|---|---|
| **Room** | `{"title": "Lecture 5", "topic": "REST APIs", "courseId": 42}` |
| **Message** | `{"replyTo": "<msgID>", "reactions": {"thumbsUp": 3}}` |
| **User** | `{"avatar": "https://...", "role": "student", "semester": 5}` |

The server will set certain keys automatically in specific situations:
- **Message edit** (`PATCH`/`PUT`): sets `"modified": true`
- **Message delete** (`DELETE`): sets `"deleted": true` and replaces message text with `"deleted"`
- **WebSocket join** (with `userInfo=true`): the self-addressed join message includes `"self": true`, `"joinedUserId"`, and `"joinedUserName"`

On `PATCH` requests, `additionalInfo` is **merged** with existing data. On `PUT` requests, it is **replaced** entirely.

### Connection

- Ping interval: 30s, pong deadline: 60s
- Max message size: 10 MiB
- Write timeout: 10s

## Room Lifecycle

1. **Created** via `POST /rooms` with optional metadata
2. **Active** while clients join or messages are sent
3. **Deleted** after 3 hours of inactivity (no joins or messages)

On room deletion, uploaded files for that room are removed. On server shutdown, all clients are disconnected and all uploads are cleaned up.

## Build with Version Info

```bash
go build -o chat-room ./cmd/chat-room \
  -ldflags="-X github.com/choffmann/chat-room/internal/config.Version=v1.0.0 \
  -X github.com/choffmann/chat-room/internal/config.GitCommit=$(git rev-parse --short HEAD) \
  -X github.com/choffmann/chat-room/internal/config.GitRepository=https://github.com/choffmann/chat-room \
  -X github.com/choffmann/chat-room/internal/config.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
```

## Testing

```bash
go test ./...
```
