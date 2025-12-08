package style

import "github.com/charmbracelet/lipgloss"

var (
	TitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("212"))

	SubtleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
)
