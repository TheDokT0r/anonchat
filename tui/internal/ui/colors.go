package ui

import "github.com/charmbracelet/lipgloss"

var userColors = []string{
	"#3b82f6", "#ef4444", "#22c55e", "#f59e0b", "#a855f7",
	"#ec4899", "#14b8a6", "#f97316", "#6366f1", "#06b6d4",
	"#84cc16", "#e879f9", "#fb923c", "#2dd4bf", "#818cf8",
}

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

func StyledName(name string) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(UserColor(name))).
		Render(name)
}
