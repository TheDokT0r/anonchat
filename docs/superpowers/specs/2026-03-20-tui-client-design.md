# TUI Client — Design Spec

## Overview

A terminal UI client for anonchat built with Go + Bubbletea. Connects to the same WebSocket backend as the web client, allowing terminal and web users to chat together in the same rooms.

## Architecture

Lives in `tui/` at the monorepo root with its own Go module (`anonchat/tui`).

```
tui/
├── go.mod
├── go.sum
├── main.go             # Entry point, flag parsing, interactive prompt fallback
├── internal/
│   ├── ws/
│   │   └── client.go   # WebSocket client (connect, send, receive, reconnect)
│   └── ui/
│       ├── app.go      # Root Bubbletea model, wires sub-models
│       ├── chat.go     # Message list view (colored names, italic system msgs)
│       ├── input.go    # Text input with typing indicator debounce
│       └── users.go    # User list sidebar
└── Dockerfile
```

## Connection

Connects to the same `/ws` endpoint as the web client. Same JSON-over-WebSocket protocol — no backend changes needed.

**Client → Server:** `{"join":{"roomName":"..."}}`, `{"leave":{}}`, `{"chat":{"content":"..."}}`, `{"typing":{"isTyping":true}}`

**Server → Client:** `{"roomJoined":{...}}`, `{"chat":{...}}`, `{"presence":{...}}`, `{"typing":{...}}`, `{"error":{...}}`

## UI Layout

```
┌─────────────────────────────────────┬──────────────┐
│ #general                            │ Users (3)    │
├─────────────────────────────────────┤ Blue Fox     │
│ Blue Fox joined                     │ Red Panda    │
│ Red Panda: hey everyone             │ Bold Tiger   │
│ Blue Fox: hello!                    │              │
│ Bold Tiger joined                   │              │
│                                     │              │
├─────────────────────────────────────┤              │
│ Red Panda is typing...              │              │
│ > type a message...                 │              │
└─────────────────────────────────────┴──────────────┘
```

## Features

- **Full parity with web client:** chat messages, presence (user list sidebar), typing indicators, join/leave system messages, per-user colored usernames
- **Color scheme:** same hash-to-color approach as the web client — consistent colors for the same username across both clients
- **System messages:** join/leave events derived by diffing previous and new user lists on each `presence` event (no dedicated join/leave server message exists). Displayed in dim/italic style
- **Optimistic send:** sender's own messages are added to the local view immediately after sending (server suppresses echo via `_sender` field)
- **Message buffer:** capped at 200 messages, matching the web client
- **Typing indicator receive timeout:** auto-clear after 3s of no update from a user, matching the web client
- **Auto-reconnect:** exponential backoff (1s → 30s max), re-joins room on reconnect
- **CLI args:** `--room` and `--server` flags. Interactive prompt if `--room` not provided. Default server: `ws://localhost:8080/ws`
- **Keyboard:** Enter to send, Ctrl+C or Esc to quit

## CLI Usage

```bash
# Interactive — prompts for room name
anonchat-tui

# Direct join
anonchat-tui --room general

# Custom server
anonchat-tui --room general --server ws://example.com/ws
```

## Dependencies

- `charmbracelet/bubbletea` — TUI framework (Elm architecture)
- `charmbracelet/bubbles` — text input component
- `charmbracelet/lipgloss` — terminal styling (colors, borders, layout)
- `nhooyr.io/websocket` — WebSocket client (same as backend)
- `flag` — CLI arg parsing (no cobra, single command)

## WebSocket Client (`internal/ws/`)

Thin wrapper around nhooyr/websocket:
- `Connect(url string)` — dial and return client
- `Send(msg interface{})` — JSON marshal and send
- `Listen(ctx context.Context) <-chan ServerMessage` — returns channel of parsed server messages
- Auto-reconnect with exponential backoff in background
- Reconnect re-sends `JoinRoom` for the current room
- New random name assigned on reconnect (no identity persistence)

## Bubbletea Models

### `app` (root model)
- Manages the two states: room prompt vs. chat view
- Holds the WebSocket client
- Routes incoming WebSocket messages to sub-models via Bubbletea commands

### `chat`
- Scrollable message list (viewport from bubbles)
- Two message types: chat messages (colored sender name + content) and system messages (dim italic)
- Auto-scrolls to bottom on new messages
- Typing indicator line above input

### `input`
- Text input (from bubbles)
- Sends typing indicator on keypress, debounced (stops after 2s idle)
- Enter sends message, clears input

### `users`
- Right sidebar panel
- Lists current users with colored names
- Shows "(you)" next to own name
- Header shows count

## Docker

```dockerfile
FROM golang:1.24-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /anonchat-tui .

FROM alpine:3.19
COPY --from=build /anonchat-tui /usr/local/bin/anonchat-tui
ENTRYPOINT ["anonchat-tui"]
```

## Testing

- **WebSocket client:** unit tests with mock WebSocket server (httptest)
- **UI models:** Bubbletea models are pure functions — test by sending messages and asserting on model state
- No integration tests with real backend in initial scope
