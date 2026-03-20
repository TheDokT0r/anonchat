# Agents Guide — anonchat

Anonymous, ephemeral chat app. Go backend + SvelteKit frontend + Redis. Monorepo.

## Architecture

```
Browser → SvelteKit (static) → WebSocket → Go backend → Redis (pub/sub + room state)
```

Rooms are ephemeral — created on first join, deleted when empty. No database, no persistence. Redis is the only state store.

## Monorepo Layout

| Directory | What | Language |
|---|---|---|
| `proto/` | Protobuf message definitions (shared contract) | Protobuf |
| `backend/` | WebSocket server, room management, rate limiting | Go |
| `frontend/` | SPA chat UI | SvelteKit 5 (TypeScript) |
| `tui/` | Terminal UI chat client | Go (Bubbletea) |
| `k8s/` | Kubernetes deployment manifests | YAML |

## Backend (`backend/`)

Go module: `anonchat/backend`

### Package Structure

- `cmd/server/main.go` — Entry point. Wires Redis, room manager, rate limiters, handler. Graceful shutdown on SIGTERM.
- `internal/handler/ws.go` — WebSocket handler. Upgrades HTTP, manages per-connection read/write loops. Tracks all connections for graceful shutdown cleanup. Health endpoints at `/healthz` and `/readyz`.
- `internal/room/manager.go` — Room lifecycle. Join/leave, random name assignment, room capacity enforcement. All state delegated to Redis via `redisclient.Store` interface. Also exposes `Publish`/`Subscribe` for message broadcasting.
- `internal/redisclient/client.go` — Redis wrapper. Defines `Store` and `PubSub` interfaces. Room state stored in Redis hashes (`room:{name}:users`), messages broadcast via Redis pub/sub channels (`room:{name}:channel`).
- `internal/ratelimit/ratelimit.go` — Two limiters: `MessageLimiter` (token bucket per connection, 10 msg/s) and `IPRoomLimiter` (sliding window per IP, 10 rooms/min).
- `internal/names/names.go` — Random name generator. "Adjective Animal" format, 400 unique combinations. Avoids duplicates within a room.
- `internal/config/config.go` — Env-based config: `PORT`, `REDIS_ADDR`, `ALLOWED_ORIGINS`.

### Key Patterns

- **All room state lives in Redis** — the backend is stateless. Any instance can serve any connection.
- **`_sender` field in pub/sub messages** — internal field used to suppress echo. Stripped before forwarding to clients.
- **Connection registry** (`Handler.conns`) — tracks active WebSocket connections so graceful shutdown can clean up Redis state.
- **Room name normalization** — lowercase, alphanumeric + hyphens, max 50 chars. Validated in `room.normalizeRoomName()`.

### Testing

- Unit tests mock Redis via `redisclient.Store` interface (see `room/manager_test.go` for mock examples).
- Integration tests use real Redis on `localhost:6379` DB 15, skip if unavailable.
- Run: `cd backend && go test -race ./...`

### WebSocket Protocol

Messages are JSON over WebSocket (not binary protobuf). The proto definitions in `proto/chat/v1/chat.proto` define the schema contract but generated code is not imported at runtime.

**Client → Server:** `{ "join": { "roomName": "..." } }`, `{ "leave": {} }`, `{ "chat": { "content": "..." } }`, `{ "typing": { "isTyping": true } }`

**Server → Client:** `{ "roomJoined": { "roomName": "...", "assignedName": "...", "users": [...] } }`, `{ "chat": { "senderName": "...", "content": "...", "timestamp": ... } }`, `{ "presence": { "users": [...] } }`, `{ "typing": { "userName": "...", "isTyping": true } }`, `{ "error": { "message": "..." } }`

## Frontend (`frontend/`)

SvelteKit 5 with `adapter-static` (SPA, no SSR).

### Structure

- `src/routes/+page.svelte` — Single page. Shows `RoomEntry` or `ChatRoom` based on room state. Creates `ChatClient`, wires callbacks to stores.
- `src/lib/ws/client.ts` — WebSocket client class. Auto-reconnect with exponential backoff (1s → 30s max). Re-joins room on reconnect.
- `src/lib/stores/room.ts` — Svelte stores: `roomState`, `messages` (capped at 200), `users`, `typingUsers` (auto-clear after 3s). Also has `userColor()` for per-user name colors and `addSystemMessage()` for join/leave events.
- `src/lib/components/` — `RoomEntry.svelte` (room name input + validation), `ChatRoom.svelte` (layout shell), `MessageList.svelte` (auto-scroll, colored names, italic system messages), `MessageInput.svelte` (typing indicator debounce), `UserList.svelte` (colored names), `TypingIndicator.svelte`.

### Key Patterns

- **Svelte 5 runes** — components use `$props()`, `$state()`, not `createEventDispatcher`.
- **Optimistic chat** — sender's own messages are added to the store immediately (server won't echo them back due to `_sender` filtering).
- **Presence diffing** — `updatePresence()` compares previous and new user lists to generate join/leave system messages.
- **Vite proxy** — dev server proxies `/ws` to backend (target configurable via `BACKEND_URL` env var).

## Infrastructure

### Local Development

```bash
docker compose -f docker-compose.dev.yml up -d
```

- Frontend: http://localhost:5173 (Vite hot reload)
- Backend: http://localhost:8080 (CompileDaemon hot reload)
- Redis: localhost:6379

### Production

```bash
docker compose up --build -d
```

- Frontend: http://localhost:3000 (nginx serves static build, proxies `/ws` to backend)
- Backend: http://localhost:8080

### Kubernetes

Raw manifests in `k8s/`. Backend runs 2 replicas (stateless). Redis is a single StatefulSet with no persistence. Nginx ingress routes `/ws` to backend, everything else to frontend.

## TUI Client (`tui/`)

Go module: `anonchat/tui`. Bubbletea-based terminal chat client.

### Structure

- `main.go` — Entry point. `--room` and `--server` flags, interactive prompt if no room given.
- `internal/ws/messages.go` — JSON message types matching the WebSocket protocol.
- `internal/ws/client.go` — WebSocket client with connect, send, listen.
- `internal/ui/app.go` — Root Bubbletea model. Two states: room prompt and chat view. Routes WebSocket messages via Bubbletea command system.
- `internal/ui/chat.go` — Message list with colored names and italic system messages. Capped at 200 messages.
- `internal/ui/input.go` — Text input with typing indicator debounce (2s).
- `internal/ui/users.go` — User list sidebar with colored names.
- `internal/ui/colors.go` — Same hash-to-color algorithm as the web frontend.

### Key Patterns

- **Optimistic send** — own messages added to view immediately (server suppresses echo).
- **Presence diffing** — compares previous/new user lists to generate join/leave system messages.
- **Typing auto-clear** — 3s timeout on received typing indicators.
- **Bubbletea commands** — WebSocket messages flow through `listenForMessages` → `ServerMsg` → `handleServerMessage`.

## Common Tasks

| Task | Command |
|---|---|
| Run backend tests | `cd backend && go test -race ./...` |
| Run TUI tests | `cd tui && go test -race ./...` |
| Build backend | `cd backend && go build ./cmd/server` |
| Build TUI | `cd tui && go build -o bin/anonchat-tui .` |
| Build frontend | `cd frontend && bun run build` |
| Generate protobuf | `buf generate` (requires `buf` CLI) |
| Lint proto | `buf lint` |
| Start dev env | `docker compose -f docker-compose.dev.yml up -d` |
| Start prod env | `docker compose up --build -d` |
| Run TUI | `cd tui && go run . --room test` |
