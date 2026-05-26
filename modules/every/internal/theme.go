// Package internal holds module-private styling constants so that
// modules/every does not depend on pkg/style. Values mirror pkg/style/theme.go.
package internal

import "charm.land/lipgloss/v2"

var (
	SubtleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

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
