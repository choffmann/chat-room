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
| `BASE_URL` | Host for Swagger UI (e.g. `example.com:8080`) | _(auto)_ |
| `LEGACY_ROUTES` | Enable unversioned legacy routes | `true` |

## API Overview

All endpoints are under `/api/v1`. Full request/response documentation is available via the **Swagger UI** at `/api/v1/swagger/`.

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

### Message Types

The `type` field accepts any string value, allowing clients to define custom message types without server-side changes. If omitted, the type defaults to `"message"`.

| Type | Stored | Description |
|---|---|---|
| `system` | Yes (< 2 MiB) | Join/leave notifications (server-generated, not sendable by clients) |
| `message` | Yes (< 2 MiB) | Text messages (default if `type` is omitted) |
| `image` | No | Base64-encoded images, broadcast only |
| _custom_ | Yes (< 2 MiB) | Any other string (e.g. `"poll"`, `"reaction"`, `"file"`) |

## `additionalInfo`

Most entities (rooms, messages, users) support an `additionalInfo` field. This is a free-form JSON object that the server stores and returns as-is, without validation or schema enforcement. It allows clients to attach arbitrary metadata without requiring server-side changes.

**Examples by entity:**

| Entity | Example use cases |
|---|---|
| **Room** | `{"title": "Lecture 5", "topic": "REST APIs", "courseId": 42}` |
| **Message** | `{"replyTo": "<msgID>", "edited": true, "reactions": {"thumbsUp": 3}}` |
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

On shutdown, all clients are disconnected and room data is discarded.

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
