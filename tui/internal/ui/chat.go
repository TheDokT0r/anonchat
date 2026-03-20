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
	messages    []Message
	typingUsers map[string]bool
	userName    string
	width       int
	height      int
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
	return max(1, m.height-2)
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

	visible := m.visibleLines()
	start := max(0, len(lines)-visible)
	if start < len(lines) {
		lines = lines[start:]
	}

	for len(lines) < visible {
		lines = append(lines, "")
	}

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
