package style

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Adaptive colors for light and dark modes
var (
	// Primary colors
	Primary = lipgloss.AdaptiveColor{
		Light: purple600,
		Dark:  purple400,
	}

	// Status colors
	Success = lipgloss.AdaptiveColor{
		Light: green600,
		Dark:  green400,
	}

	Warning = lipgloss.AdaptiveColor{
		Light: orange500,
		Dark:  orange200,
	}

	Error = lipgloss.AdaptiveColor{
		Light: red600,
		Dark:  red400,
	}

	Info = lipgloss.AdaptiveColor{
		Light: blue600,
		Dark:  blue400,
	}

	// Text colors
	Text = lipgloss.AdaptiveColor{
		Light: gray900,
		Dark:  gray100,
	}

	TextMuted = lipgloss.AdaptiveColor{
		Light: gray600,
		Dark:  gray400,
	}

	TextDisabled = lipgloss.AdaptiveColor{
		Light: gray400,
		Dark:  gray600,
	}

	// UI colors
	Border = lipgloss.AdaptiveColor{
		Light: gray300,
		Dark:  gray700,
	}

	Background = lipgloss.AdaptiveColor{
		Light: white,
		Dark:  gray900,
	}

	Highlight = lipgloss.AdaptiveColor{
		Light: purple100,
		Dark:  purple800,
	}

	Focus = lipgloss.AdaptiveColor{
		Light: purple600,
		Dark:  purple400,
	}

	Deprioritized = lipgloss.AdaptiveColor{
		Light: gray600,
		Dark:  gray200,
	}

	ColorWarningDeprioritized = lipgloss.AdaptiveColor{
		Light: blue600,
		Dark:  blue200,
	}
)

// Predefined styles
var (
	// Title styles
	Title = lipgloss.NewStyle().
		Foreground(Primary).
		Bold(true)

	TitleBlock = lipgloss.NewStyle().
			Background(Primary).
			Foreground(lipgloss.Color(white)).
			Bold(true).
			Padding(0, 1)

	// Text styles
	Label = lipgloss.NewStyle().
		Foreground(Focus).
		Bold(true)

	Status = lipgloss.NewStyle().
		Bold(true)

	// Command styles
	CommandTitle = lipgloss.NewStyle().
			Foreground(Info).
			Bold(true)

	CommandKey = lipgloss.NewStyle().
			Foreground(Info)

	// Highlight styles
	HighlightBlock = lipgloss.NewStyle().
			Background(Highlight).
			Foreground(Text).
			Padding(0, 1)

	// Table styles
	TableHeader = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true).
			Align(lipgloss.Center)

	TableBorder = lipgloss.NewStyle().
			Foreground(Border)

	TableRowEven = lipgloss.NewStyle().
			Foreground(Text)

	TableRowOdd = lipgloss.NewStyle().
			Foreground(TextMuted)

	// Status-specific styles
	SuccessStyle = lipgloss.NewStyle().
			Foreground(Success).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(Warning).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(Error).
			Bold(true)

	InfoStyle = lipgloss.NewStyle().
			Foreground(Info).
			Bold(true)

	// Utility styles
	Bold = lipgloss.NewStyle().
		Bold(true)

	Italic = lipgloss.NewStyle().
		Italic(true)

	Underline = lipgloss.NewStyle().
			Underline(true)

	Dimmed = lipgloss.NewStyle().
		Foreground(TextMuted)
)

// Utility functions

// FormatKeyValue formats a key-value pair with consistent styling
func FormatKeyValue(key, value string) string {
	return fmt.Sprintf("%s: %s", Label.Render(key), value)
}

// FormatStatus formats status text with appropriate color
func FormatStatus(status, text string) string {
	switch status {
	case "success", "ok", "completed":
		return SuccessStyle.Render(text)
	case "warning", "pending":
		return WarningStyle.Render(text)
	case "error", "failed":
		return ErrorStyle.Render(text)
	case "info":
		return InfoStyle.Render(text)
	default:
		return text
	}
}

// FormatDuration formats duration with consistent styling
func FormatDuration(duration string) string {
	return CommandKey.Render(duration)
}

// FormatCommand formats command text with consistent styling
func FormatCommand(command string) string {
	return CommandTitle.Render(command)
}

// FormatHighlight formats highlighted text
func FormatHighlight(text string) string {
	return HighlightBlock.Render(text)
}

// FormatTable formats table elements with consistent styling
func FormatTableHeader(text string) string {
	return TableHeader.Render(text)
}

func FormatTableRow(text string, isEven bool) string {
	if isEven {
		return TableRowEven.Render(text)
	}
	return TableRowOdd.Render(text)
}

// Theme-aware border styles
func GetBorderStyle() lipgloss.Border {
	return lipgloss.ThickBorder()
}

func GetBorderColor() lipgloss.TerminalColor {
	return Border
}
