package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type UsersModel struct {
	users    []string
	userName string
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
