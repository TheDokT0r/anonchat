package ui

import (
	"fmt"
	"strings"
	"time"

	"context"

	"anonchat/tui/internal/ws"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Bubbletea messages
type ServerMsg ws.ServerMessage
type TickTypingMsg struct{ UserName string }
type connectedMsg struct{ msgs <-chan ws.ServerMessage }
type errMsg struct{ err error }
type stopTypingTick struct{}

type state int

const (
	statePrompt state = iota
	stateChat
)

type AppModel struct {
	state     state
	client    *ws.Client
	serverURL string
	roomName  string
	userName  string

	promptInput textinput.Model
	chatModel   ChatModel
	inputModel  InputModel
	usersModel  UsersModel

	prevUsers []string
	msgCh     <-chan ws.ServerMessage

	lastTypingSent time.Time
	typingSent     bool

	width  int
	height int
	err    error
}

func NewAppModel(client *ws.Client, serverURL string, roomName string) AppModel {
	pi := textinput.New()
	pi.Placeholder = "Enter a room name..."
	pi.Focus()
	pi.CharLimit = 50

	m := AppModel{
		client:    client,
		serverURL: serverURL,
		chatModel: NewChatModel(),
		inputModel: NewInputModel(),
		usersModel: NewUsersModel(),
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

func connectToServer(client *ws.Client, room string) tea.Cmd {
	return func() tea.Msg {
		msgs, err := client.Connect(context.Background())
		if err != nil {
			return errMsg{err}
		}
		client.Send(ws.ClientMessage{Join: &ws.JoinMsg{RoomName: room}})
		return connectedMsg{msgs}
	}
}

func listenForMessages(msgCh <-chan ws.ServerMessage) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-msgCh
		if !ok {
			return nil
		}
		return ServerMsg(msg)
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
		m.msgCh = msg.msgs
		return m, listenForMessages(m.msgCh)

	case ServerMsg:
		var cmd tea.Cmd
		m, cmd = m.handleServerMessage(ws.ServerMessage(msg))
		if m.msgCh != nil {
			return m, tea.Batch(cmd, listenForMessages(m.msgCh))
		}
		return m, cmd

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

	case stopTypingTick:
		if m.typingSent && time.Since(m.lastTypingSent) >= 2*time.Second {
			m.typingSent = false
			m.client.Send(ws.ClientMessage{Typing: &ws.TypingSendMsg{IsTyping: false}})
		}
		return m, nil
	}

	// Always pass through to text input first
	var cmd tea.Cmd
	m.inputModel, cmd = m.inputModel.Update(msg)

	// Then handle typing indicator (after input has the keypress)
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.Type == tea.KeyRunes && !m.typingSent {
		m.typingSent = true
		m.lastTypingSent = time.Now()
		m.client.Send(ws.ClientMessage{Typing: &ws.TypingSendMsg{IsTyping: true}})
		return m, tea.Batch(cmd, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return stopTypingTick{}
		}))
	}

	return m, cmd
}

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
	chatWidth := m.width - sidebarWidth - 1
	chatHeight := m.height - 3

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
