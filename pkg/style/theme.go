package style

import "charm.land/lipgloss/v2"

var (
	TitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("212"))

	SubtleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	// Colors for CLI output
	BlueStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("33")) // Blue

	GreenStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("34")) // Green

	YellowStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("220")) // Yellow

	RedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")) // Red
)
