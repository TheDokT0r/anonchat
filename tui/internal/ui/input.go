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
	m.textInput.Width = w - 4
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
