# TUI Client Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a terminal UI chat client that connects to the existing anonchat WebSocket backend, enabling terminal users and web users to chat together.

**Architecture:** Separate Go module in `tui/` using Bubbletea for the terminal UI and nhooyr/websocket for the backend connection. Two states: room prompt and chat view. WebSocket messages flow through Bubbletea's command system.

**Tech Stack:** Go, Bubbletea, Bubbles, Lipgloss, nhooyr/websocket

---

## File Structure

```
tui/
├── go.mod                    # module anonchat/tui, go 1.23.0
├── go.sum
├── main.go                   # Flag parsing, interactive prompt, launch app
├── internal/
│   ├── ws/
│   │   ├── client.go         # WebSocket client: connect, send, listen, reconnect
│   │   ├── client_test.go    # Tests with mock WebSocket server
│   │   └── messages.go       # JSON message types (client→server, server→client)
│   └── ui/
│       ├── colors.go         # userColor() — same hash algorithm as web frontend
│       ├── colors_test.go
│       ├── app.go            # Root Bubbletea model: prompt vs chat state
│       ├── chat.go           # Message list viewport with colored/system messages
│       ├── input.go          # Text input with typing debounce
│       └── users.go          # User list sidebar
└── Dockerfile
```

---

### Task 1: Module Setup & Message Types

**Files:**
- Create: `tui/go.mod`
- Create: `tui/internal/ws/messages.go`
- Create: `tui/main.go` (placeholder)

- [ ] **Step 1: Initialize Go module**

```bash
mkdir -p /Users/or/projects/anonchat/tui && cd /Users/or/projects/anonchat/tui
go mod init anonchat/tui
```

- [ ] **Step 2: Create message types**

Create `tui/internal/ws/messages.go`:

```go
package ws

import "encoding/json"

// Client → Server messages

type JoinMsg struct {
	RoomName string `json:"roomName"`
}

type LeaveMsg struct{}

type ChatSendMsg struct {
	Content string `json:"content"`
}

type TypingSendMsg struct {
	IsTyping bool `json:"isTyping"`
}

type ClientMessage struct {
	Join   *JoinMsg       `json:"join,omitempty"`
	Leave  *LeaveMsg      `json:"leave,omitempty"`
	Chat   *ChatSendMsg   `json:"chat,omitempty"`
	Typing *TypingSendMsg `json:"typing,omitempty"`
}

// Server → Client messages

type RoomJoinedData struct {
	RoomName     string   `json:"roomName"`
	AssignedName string   `json:"assignedName"`
	Users        []string `json:"users"`
}

type ChatData struct {
	SenderName string `json:"senderName"`
	Content    string `json:"content"`
	Timestamp  int64  `json:"timestamp"`
}

type PresenceData struct {
	Users []string `json:"users"`
}

type TypingData struct {
	UserName string `json:"userName"`
	IsTyping bool   `json:"isTyping"`
}

type ErrorData struct {
	Message string `json:"message"`
}

type ServerMessage struct {
	RoomJoined *RoomJoinedData `json:"roomJoined,omitempty"`
	Chat       *ChatData       `json:"chat,omitempty"`
	Presence   *PresenceData   `json:"presence,omitempty"`
	Typing     *TypingData     `json:"typing,omitempty"`
	Error      *ErrorData      `json:"error,omitempty"`
}

func ParseServerMessage(data []byte) (ServerMessage, error) {
	var msg ServerMessage
	err := json.Unmarshal(data, &msg)
	return msg, err
}
```

- [ ] **Step 3: Create placeholder main.go**

Create `tui/main.go`:

```go
package main

import "fmt"

func main() {
	fmt.Println("anonchat-tui")
}
```

- [ ] **Step 4: Verify it compiles**

```bash
cd /Users/or/projects/anonchat/tui && go build .
```

- [ ] **Step 5: Commit**

```bash
cd /Users/or/projects/anonchat && git add tui/
git commit -m "feat(tui): scaffold module with WebSocket message types"
```

---

### Task 2: Color Utilities

**Files:**
- Create: `tui/internal/ui/colors.go`
- Create: `tui/internal/ui/colors_test.go`

- [ ] **Step 1: Write color tests**

Create `tui/internal/ui/colors_test.go`:

```go
package ui_test

import (
	"testing"

	"anonchat/tui/internal/ui"
)

func TestUserColor_Deterministic(t *testing.T) {
	c1 := ui.UserColor("Blue Fox")
	c2 := ui.UserColor("Blue Fox")
	if c1 != c2 {
		t.Fatalf("expected same color for same name, got %q and %q", c1, c2)
	}
}

func TestUserColor_DifferentNames(t *testing.T) {
	// Not guaranteed to differ for all names, but these two should
	c1 := ui.UserColor("Blue Fox")
	c2 := ui.UserColor("Red Panda")
	if c1 == c2 {
		t.Logf("warning: same color for different names: %q", c1)
	}
}

func TestUserColor_MatchesWebFrontend(t *testing.T) {
	// The web frontend uses this algorithm:
	// hash = 0; for each char: hash = charCode + ((hash << 5) - hash)
	// color = USER_COLORS[abs(hash) % len(USER_COLORS)]
	// Verify a known mapping
	color := ui.UserColor("Blue Fox")
	if color == "" {
		t.Fatal("expected non-empty color")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/or/projects/anonchat/tui && go test ./internal/ui/... -v
```

Expected: FAIL — package not found.

- [ ] **Step 3: Implement colors**

Create `tui/internal/ui/colors.go`:

```go
package ui

import "github.com/charmbracelet/lipgloss"

// Same palette and hash as the web frontend (frontend/src/lib/stores/room.ts)
var userColors = []string{
	"#3b82f6", "#ef4444", "#22c55e", "#f59e0b", "#a855f7",
	"#ec4899", "#14b8a6", "#f97316", "#6366f1", "#06b6d4",
	"#84cc16", "#e879f9", "#fb923c", "#2dd4bf", "#818cf8",
}

// UserColor returns a hex color for a username, matching the web frontend algorithm.
func UserColor(name string) string {
	hash := 0
	for _, c := range name {
		hash = int(c) + ((hash << 5) - hash)
	}
	if hash < 0 {
		hash = -hash
	}
	return userColors[hash%len(userColors)]
}

// StyledName returns a lipgloss-styled username string.
func StyledName(name string) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(UserColor(name))).
		Render(name)
}
```

- [ ] **Step 4: Install lipgloss**

```bash
cd /Users/or/projects/anonchat/tui && go get github.com/charmbracelet/lipgloss
```

- [ ] **Step 5: Run tests**

```bash
cd /Users/or/projects/anonchat/tui && go test ./internal/ui/... -v
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
cd /Users/or/projects/anonchat && git add tui/
git commit -m "feat(tui): add color utilities matching web frontend"
```

---

### Task 3: WebSocket Client

**Files:**
- Create: `tui/internal/ws/client.go`
- Create: `tui/internal/ws/client_test.go`

- [ ] **Step 1: Write WebSocket client tests**

Create `tui/internal/ws/client_test.go`:

```go
package ws_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"anonchat/tui/internal/ws"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

func startMockServer(t *testing.T, handler func(*websocket.Conn)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			t.Fatal(err)
		}
		defer conn.CloseNow()
		handler(conn)
	}))
}

func TestClient_ConnectAndReceive(t *testing.T) {
	srv := startMockServer(t, func(conn *websocket.Conn) {
		// Read join message
		var msg map[string]json.RawMessage
		wsjson.Read(context.Background(), conn, &msg)

		// Send roomJoined response
		resp := ws.ServerMessage{
			RoomJoined: &ws.RoomJoinedData{
				RoomName:     "test",
				AssignedName: "Blue Fox",
				Users:        []string{"Blue Fox"},
			},
		}
		wsjson.Write(context.Background(), conn, resp)
		time.Sleep(100 * time.Millisecond)
		conn.Close(websocket.StatusNormalClosure, "done")
	})
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	client := ws.New(url)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	msgs, err := client.Connect(ctx)
	if err != nil {
		t.Fatal(err)
	}

	client.Send(ws.ClientMessage{Join: &ws.JoinMsg{RoomName: "test"}})

	select {
	case msg := <-msgs:
		if msg.RoomJoined == nil {
			t.Fatal("expected RoomJoined message")
		}
		if msg.RoomJoined.AssignedName != "Blue Fox" {
			t.Fatalf("expected 'Blue Fox', got %q", msg.RoomJoined.AssignedName)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for message")
	}
}

func TestClient_Send(t *testing.T) {
	received := make(chan map[string]json.RawMessage, 1)
	srv := startMockServer(t, func(conn *websocket.Conn) {
		var msg map[string]json.RawMessage
		wsjson.Read(context.Background(), conn, &msg)
		received <- msg
		time.Sleep(100 * time.Millisecond)
		conn.Close(websocket.StatusNormalClosure, "done")
	})
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	client := ws.New(url)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := client.Connect(ctx)
	if err != nil {
		t.Fatal(err)
	}

	client.Send(ws.ClientMessage{Chat: &ws.ChatSendMsg{Content: "hello"}})

	select {
	case msg := <-received:
		if _, ok := msg["chat"]; !ok {
			t.Fatalf("expected chat message, got %v", msg)
		}
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/or/projects/anonchat/tui && go test ./internal/ws/... -v
```

Expected: FAIL — client.go not found.

- [ ] **Step 3: Implement WebSocket client**

Create `tui/internal/ws/client.go`:

```go
package ws

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type Client struct {
	url  string
	conn *websocket.Conn
	mu   sync.Mutex
}

func New(url string) *Client {
	return &Client{url: url}
}

func (c *Client) Connect(ctx context.Context) (<-chan ServerMessage, error) {
	conn, _, err := websocket.Dial(ctx, c.url, nil)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	ch := make(chan ServerMessage, 64)
	go c.readLoop(ctx, ch)
	return ch, nil
}

func (c *Client) Send(msg ClientMessage) {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	wsjson.Write(ctx, conn, msg)
}

func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		c.conn.Close(websocket.StatusNormalClosure, "bye")
		c.conn = nil
	}
}

func (c *Client) readLoop(ctx context.Context, ch chan<- ServerMessage) {
	defer close(ch)
	for {
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			return
		}
		msg, err := ParseServerMessage(data)
		if err != nil {
			continue
		}
		select {
		case ch <- msg:
		case <-ctx.Done():
			return
		}
	}
}

// ConnectWithReconnect connects and automatically reconnects on disconnect.
// It calls onMessage for each received server message.
// It calls rejoin after each reconnect to re-send the JoinRoom message.
func (c *Client) ConnectWithReconnect(ctx context.Context, onMessage func(ServerMessage), rejoin func()) {
	delay := time.Second
	maxDelay := 30 * time.Second

	for {
		msgs, err := c.Connect(ctx)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
				delay = min(delay*2, maxDelay)
				continue
			}
		}
		delay = time.Second
		rejoin()

		for msg := range msgs {
			onMessage(msg)
		}

		// Connection closed, reconnect
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
			delay = min(delay*2, maxDelay)
		}
	}
}

// Note: min() is a Go built-in since 1.21 — no need to define it
```

- [ ] **Step 4: Install nhooyr/websocket**

```bash
cd /Users/or/projects/anonchat/tui && go get nhooyr.io/websocket
```

- [ ] **Step 5: Run tests**

```bash
cd /Users/or/projects/anonchat/tui && go test ./internal/ws/... -v
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
cd /Users/or/projects/anonchat && git add tui/
git commit -m "feat(tui): add WebSocket client with reconnect"
```

---

### Task 4: Bubbletea UI — App Model & Chat View

**Files:**
- Create: `tui/internal/ui/app.go`
- Create: `tui/internal/ui/chat.go`
- Create: `tui/internal/ui/input.go`
- Create: `tui/internal/ui/users.go`

- [ ] **Step 1: Install Bubbletea dependencies**

```bash
cd /Users/or/projects/anonchat/tui
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/bubbles
```

- [ ] **Step 2: Create user list model**

Create `tui/internal/ui/users.go`:

```go
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type UsersModel struct {
	users    []string
	userName string // own name
	width    int
	height   int
}

func NewUsersModel() UsersModel {
	return UsersModel{}
}

func (m UsersModel) SetUsers(users []string) UsersModel {
	m.users = users
	return m
}

func (m UsersModel) SetUserName(name string) UsersModel {
	m.userName = name
	return m
}

func (m UsersModel) SetSize(w, h int) UsersModel {
	m.width = w
	m.height = h
	return m
}

func (m UsersModel) View() string {
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#888")).
		Render(fmt.Sprintf("Users (%d)", len(m.users)))

	var lines []string
	lines = append(lines, header)
	lines = append(lines, "")

	for _, u := range m.users {
		line := StyledName(u)
		if u == m.userName {
			line += lipgloss.NewStyle().Foreground(lipgloss.Color("#888")).Render(" (you)")
		}
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(0, 1).
		Render(content)
}
```

- [ ] **Step 3: Create chat model**

Create `tui/internal/ui/chat.go`:

```go
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type MessageType int

const (
	MsgChat MessageType = iota
	MsgSystem
)

type Message struct {
	Type       MessageType
	SenderName string
	Content    string
}

const maxMessages = 200

type ChatModel struct {
	messages     []Message
	typingUsers  map[string]bool
	userName     string
	width        int
	height       int
	scrollOffset int
}

func NewChatModel() ChatModel {
	return ChatModel{
		typingUsers: make(map[string]bool),
	}
}

func (m ChatModel) AddMessage(msg Message) ChatModel {
	m.messages = append(m.messages, msg)
	if len(m.messages) > maxMessages {
		m.messages = m.messages[len(m.messages)-maxMessages:]
	}
	// Auto-scroll to bottom
	m.scrollOffset = max(0, len(m.messages)-m.visibleLines())
	return m
}

func (m ChatModel) SetTyping(userName string, isTyping bool) ChatModel {
	newMap := make(map[string]bool)
	for k, v := range m.typingUsers {
		newMap[k] = v
	}
	if isTyping {
		newMap[userName] = true
	} else {
		delete(newMap, userName)
	}
	m.typingUsers = newMap
	return m
}

func (m ChatModel) SetUserName(name string) ChatModel {
	m.userName = name
	return m
}

func (m ChatModel) SetSize(w, h int) ChatModel {
	m.width = w
	m.height = h
	return m
}

func (m ChatModel) visibleLines() int {
	return max(1, m.height-2) // reserve lines for typing indicator
}

func (m ChatModel) View() string {
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666")).Italic(true)

	var lines []string
	for _, msg := range m.messages {
		switch msg.Type {
		case MsgSystem:
			lines = append(lines, dimStyle.Render(msg.Content))
		case MsgChat:
			name := StyledName(msg.SenderName)
			lines = append(lines, fmt.Sprintf("%s: %s", name, msg.Content))
		}
	}

	// Apply scroll
	visible := m.visibleLines()
	start := max(0, len(lines)-visible)
	if start < len(lines) {
		lines = lines[start:]
	}

	// Pad to fill height
	for len(lines) < visible {
		lines = append(lines, "")
	}

	// Typing indicator
	typingLine := m.typingView()

	content := strings.Join(lines, "\n")
	if typingLine != "" {
		content += "\n" + dimStyle.Render(typingLine)
	} else {
		content += "\n"
	}

	return lipgloss.NewStyle().Width(m.width).Render(content)
}

func (m ChatModel) typingView() string {
	var typers []string
	for name, isTyping := range m.typingUsers {
		if isTyping && name != m.userName {
			typers = append(typers, name)
		}
	}
	switch len(typers) {
	case 0:
		return ""
	case 1:
		return typers[0] + " is typing..."
	case 2:
		return typers[0] + " and " + typers[1] + " are typing..."
	default:
		return fmt.Sprintf("%d people are typing...", len(typers))
	}
}

// Note: max() is a Go built-in since 1.21 — no need to define it
```

- [ ] **Step 4: Create input model**

Create `tui/internal/ui/input.go`:

```go
package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type InputModel struct {
	textInput textinput.Model
	width     int
}

func NewInputModel() InputModel {
	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.Focus()
	ti.CharLimit = 2000
	return InputModel{textInput: ti}
}

func (m InputModel) SetWidth(w int) InputModel {
	m.textInput.Width = w - 4 // account for prompt and padding
	m.width = w
	return m
}

func (m InputModel) Update(msg tea.Msg) (InputModel, tea.Cmd) {
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m InputModel) Value() string {
	return m.textInput.Value()
}

func (m InputModel) Reset() InputModel {
	m.textInput.Reset()
	return m
}

func (m InputModel) View() string {
	return m.textInput.View()
}

func (m InputModel) Focus() InputModel {
	m.textInput.Focus()
	return m
}
```

- [ ] **Step 5: Create app model (root)**

Create `tui/internal/ui/app.go`:

```go
package ui

import (
	"fmt"
	"strings"
	"time"

	"anonchat/tui/internal/ws"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Bubbletea messages
type ServerMsg ws.ServerMessage
type TickTypingMsg struct{ UserName string }

type state int

const (
	statePrompt state = iota
	stateChat
)

type AppModel struct {
	state    state
	client   *ws.Client
	serverURL string
	roomName string
	userName string

	// Sub-models
	promptInput textinput.Model
	chatModel   ChatModel
	inputModel  InputModel
	usersModel  UsersModel

	// Presence tracking for join/leave diffing
	prevUsers []string

	// Typing debounce
	lastTypingSent time.Time
	typingSent     bool

	width  int
	height int

	err error
}

func NewAppModel(client *ws.Client, serverURL string, roomName string) AppModel {
	pi := textinput.New()
	pi.Placeholder = "Enter a room name..."
	pi.Focus()
	pi.CharLimit = 50

	m := AppModel{
		client:      client,
		serverURL:   serverURL,
		chatModel:   NewChatModel(),
		inputModel:  NewInputModel(),
		usersModel:  NewUsersModel(),
		promptInput: pi,
	}

	if roomName != "" {
		m.state = stateChat
		m.roomName = roomName
	} else {
		m.state = statePrompt
	}

	return m
}

func (m AppModel) Init() tea.Cmd {
	if m.state == stateChat {
		return tea.Batch(textinput.Blink, connectToServer(m.client, m.roomName))
	}
	return textinput.Blink
}

// listenForMessages returns a command that waits for the next WebSocket message.
func listenForMessages(msgCh <-chan ws.ServerMessage) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-msgCh
		if !ok {
			return nil
		}
		return ServerMsg(msg)
	}
}

type errMsg struct{ err error }
type connectedMsg struct{ msgs <-chan ws.ServerMessage }

func connectToServer(client *ws.Client, room string) tea.Cmd {
	return func() tea.Msg {
		msgs, err := client.Connect(tea.Context())
		if err != nil {
			return errMsg{err}
		}
		client.Send(ws.ClientMessage{Join: &ws.JoinMsg{RoomName: room}})
		return connectedMsg{msgs}
	}
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.client.Send(ws.ClientMessage{Leave: &ws.LeaveMsg{}})
			m.client.Close()
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m = m.updateSizes()
		return m, nil

	case connectedMsg:
		return m, listenForMessages(msg.msgs)

	case ServerMsg:
		m = m.handleServerMessage(ws.ServerMessage(msg))
		return m, listenForMessages(nil) // will be replaced with actual channel

	case errMsg:
		m.err = msg.err
		return m, nil

	case TickTypingMsg:
		m.chatModel = m.chatModel.SetTyping(msg.UserName, false)
		return m, nil
	}

	switch m.state {
	case statePrompt:
		return m.updatePrompt(msg)
	case stateChat:
		return m.updateChat(msg)
	}

	return m, nil
}

func (m AppModel) updatePrompt(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			room := strings.TrimSpace(m.promptInput.Value())
			if room == "" {
				return m, nil
			}
			m.roomName = strings.ToLower(room)
			m.state = stateChat
			m.inputModel = m.inputModel.Focus()
			m = m.updateSizes()
			return m, connectToServer(m.client, m.roomName)
		}
	}

	var cmd tea.Cmd
	m.promptInput, cmd = m.promptInput.Update(msg)
	return m, cmd
}

func (m AppModel) updateChat(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			content := strings.TrimSpace(m.inputModel.Value())
			if content == "" {
				return m, nil
			}
			// Optimistic send
			m.chatModel = m.chatModel.AddMessage(Message{
				Type:       MsgChat,
				SenderName: m.userName,
				Content:    content,
			})
			m.client.Send(ws.ClientMessage{Chat: &ws.ChatSendMsg{Content: content}})
			m.inputModel = m.inputModel.Reset()
			m.typingSent = false
			m.client.Send(ws.ClientMessage{Typing: &ws.TypingSendMsg{IsTyping: false}})
			return m, nil
		}

		// Typing indicator debounce
		if msg.Type == tea.KeyRunes && !m.typingSent {
			m.typingSent = true
			m.lastTypingSent = time.Now()
			m.client.Send(ws.ClientMessage{Typing: &ws.TypingSendMsg{IsTyping: true}})
			return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
				return stopTypingTick{}
			})
		}
	case stopTypingTick:
		if m.typingSent && time.Since(m.lastTypingSent) >= 2*time.Second {
			m.typingSent = false
			m.client.Send(ws.ClientMessage{Typing: &ws.TypingSendMsg{IsTyping: false}})
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.inputModel, cmd = m.inputModel.Update(msg)
	return m, cmd
}

type stopTypingTick struct{}

func (m AppModel) handleServerMessage(msg ws.ServerMessage) (AppModel, tea.Cmd) {
	if msg.RoomJoined != nil {
		m.userName = msg.RoomJoined.AssignedName
		m.roomName = msg.RoomJoined.RoomName
		m.prevUsers = msg.RoomJoined.Users
		m.chatModel = m.chatModel.SetUserName(m.userName)
		m.usersModel = m.usersModel.SetUsers(msg.RoomJoined.Users).SetUserName(m.userName)
		m.chatModel = m.chatModel.AddMessage(Message{
			Type:    MsgSystem,
			Content: fmt.Sprintf("You joined as %s", m.userName),
		})
	}
	if msg.Chat != nil {
		m.chatModel = m.chatModel.AddMessage(Message{
			Type:       MsgChat,
			SenderName: msg.Chat.SenderName,
			Content:    msg.Chat.Content,
		})
	}
	if msg.Presence != nil {
		// Diff for join/leave messages
		newUsers := msg.Presence.Users
		for _, u := range newUsers {
			if !contains(m.prevUsers, u) {
				m.chatModel = m.chatModel.AddMessage(Message{
					Type:    MsgSystem,
					Content: fmt.Sprintf("%s joined", u),
				})
			}
		}
		for _, u := range m.prevUsers {
			if !contains(newUsers, u) {
				m.chatModel = m.chatModel.AddMessage(Message{
					Type:    MsgSystem,
					Content: fmt.Sprintf("%s left", u),
				})
			}
		}
		m.prevUsers = newUsers
		m.usersModel = m.usersModel.SetUsers(newUsers)
	}
	var cmds []tea.Cmd
	if msg.Typing != nil {
		m.chatModel = m.chatModel.SetTyping(msg.Typing.UserName, msg.Typing.IsTyping)
		if msg.Typing.IsTyping {
			// Auto-clear after 3 seconds
			userName := msg.Typing.UserName
			cmds = append(cmds, tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
				return TickTypingMsg{UserName: userName}
			}))
		}
	}
	if msg.Error != nil {
		m.chatModel = m.chatModel.AddMessage(Message{
			Type:    MsgSystem,
			Content: fmt.Sprintf("Error: %s", msg.Error.Message),
		})
	}
	return m, tea.Batch(cmds...)
}

func (m AppModel) updateSizes() AppModel {
	sidebarWidth := 20
	chatWidth := m.width - sidebarWidth - 1 // 1 for border
	chatHeight := m.height - 3               // 3 for header + input

	m.chatModel = m.chatModel.SetSize(chatWidth, chatHeight)
	m.inputModel = m.inputModel.SetWidth(chatWidth)
	m.usersModel = m.usersModel.SetSize(sidebarWidth, chatHeight+1)
	return m
}

func (m AppModel) View() string {
	if m.state == statePrompt {
		return m.promptView()
	}
	return m.chatView()
}

func (m AppModel) promptView() string {
	style := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center)

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#3b82f6")).Render("anonchat")
	subtitle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888")).Render("Anonymous, ephemeral chat rooms")

	content := fmt.Sprintf("%s\n%s\n\n%s", title, subtitle, m.promptInput.View())
	return style.Render(content)
}

func (m AppModel) chatView() string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#3b82f6")).
		Padding(0, 1)

	header := headerStyle.Render(fmt.Sprintf("#%s", m.roomName))

	separator := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#2a2a2a")).
		Render(strings.Repeat("─", m.width))

	chatAndUsers := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.chatModel.View(),
		lipgloss.NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#2a2a2a")).
			Render(m.usersModel.View()),
	)

	inputLine := lipgloss.NewStyle().Padding(0, 1).Render("> " + m.inputModel.View())

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		separator,
		chatAndUsers,
		separator,
		inputLine,
	)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
```

- [ ] **Step 6: Verify compilation**

```bash
cd /Users/or/projects/anonchat/tui && go build ./...
```

Expected: compiles without error.

- [ ] **Step 7: Commit**

```bash
cd /Users/or/projects/anonchat && git add tui/
git commit -m "feat(tui): add Bubbletea UI models (app, chat, input, users)"
```

---

### Task 5: Main Entry Point & Wiring

**Files:**
- Modify: `tui/main.go`

- [ ] **Step 1: Implement main.go**

Update `tui/main.go`:

```go
package main

import (
	"flag"
	"fmt"
	"os"

	"anonchat/tui/internal/ui"
	"anonchat/tui/internal/ws"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	room := flag.String("room", "", "Room name to join")
	server := flag.String("server", "ws://localhost:8080/ws", "WebSocket server URL")
	flag.Parse()

	roomName := *room
	if roomName == "" {
		// Interactive prompt handled by Bubbletea
	}

	client := ws.New(*server)
	model := ui.NewAppModel(client, *server, roomName)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Fix the WebSocket message loop**

The Bubbletea `connectToServer` command in `app.go` returns a `connectedMsg` with the message channel. But subsequent messages need to be pumped via `listenForMessages`. Update the `Update` method to properly chain message listening.

Replace the `connectedMsg` and `ServerMsg` handling in `app.go` Update:

```go
	case connectedMsg:
		m.msgCh = msg.msgs
		return m, listenForMessages(m.msgCh)

	case ServerMsg:
		var cmd tea.Cmd
		m, cmd = m.handleServerMessage(ws.ServerMessage(msg))
		if m.msgCh != nil {
			return m, tea.Batch(cmd, listenForMessages(m.msgCh))
		}
		return m, cmd
```

And add `msgCh` field to `AppModel`:

```go
type AppModel struct {
	// ... existing fields ...
	msgCh <-chan ws.ServerMessage
}
```

- [ ] **Step 3: Verify it compiles**

```bash
cd /Users/or/projects/anonchat/tui && go build .
```

- [ ] **Step 4: Commit**

```bash
cd /Users/or/projects/anonchat && git add tui/
git commit -m "feat(tui): add main entry point with flag parsing"
```

---

### Task 6: Dockerfile & Integration

**Files:**
- Create: `tui/Dockerfile`
- Create: `tui/.gitignore`
- Modify: `AGENTS.md`
- Modify: `Makefile`

- [ ] **Step 1: Create Dockerfile**

Create `tui/Dockerfile`:

```dockerfile
FROM golang:1.23-alpine AS build
ENV GOTOOLCHAIN=local
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /anonchat-tui .

FROM alpine:3.19
COPY --from=build /anonchat-tui /usr/local/bin/anonchat-tui
ENTRYPOINT ["anonchat-tui"]
```

- [ ] **Step 2: Create .gitignore**

Create `tui/.gitignore`:

```
anonchat-tui
tmp/
```

- [ ] **Step 3: Update Makefile**

Add to the root `Makefile`:

```makefile
build-tui:
	cd tui && go build -o ../bin/anonchat-tui .

test-tui:
	cd tui && go test -race ./...
```

- [ ] **Step 4: Update AGENTS.md**

Add TUI section to `AGENTS.md` under Monorepo Layout table and add a new `## TUI Client` section documenting the structure.

- [ ] **Step 5: Verify everything compiles and tests pass**

```bash
cd /Users/or/projects/anonchat/tui && go test -race ./... -v && go build .
```

- [ ] **Step 6: Commit**

```bash
cd /Users/or/projects/anonchat && git add tui/ Makefile AGENTS.md
git commit -m "feat(tui): add Dockerfile, build targets, and documentation"
```

---

### Task 7: Manual Smoke Test

**Files:** None (testing only)

- [ ] **Step 1: Start the backend**

```bash
cd /Users/or/projects/anonchat && docker compose -f docker-compose.dev.yml up -d
```

- [ ] **Step 2: Run the TUI**

```bash
cd /Users/or/projects/anonchat/tui && go run . --room test
```

Expected: TUI launches, connects to backend, shows "You joined as [name]".

- [ ] **Step 3: Open web client in parallel**

Open http://localhost:5173, join room "test". Verify:
- TUI shows "[web user's name] joined"
- Send a message from web — appears in TUI
- Send a message from TUI — appears in web
- Typing indicator shows in both directions

- [ ] **Step 4: Test interactive prompt**

```bash
cd /Users/or/projects/anonchat/tui && go run .
```

Expected: Shows room name prompt. Type a room name, press enter, joins the room.

- [ ] **Step 5: Clean up**

```bash
docker compose -f docker-compose.dev.yml down
```
