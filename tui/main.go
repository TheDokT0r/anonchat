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

	client := ws.New(*server)
	model := ui.NewAppModel(client, *server, *room)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
