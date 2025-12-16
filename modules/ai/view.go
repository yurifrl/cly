package ai

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/yurifrl/cly/pkg/style"
)

var (
	userStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(style.BlueStyle.GetForeground()).
			Padding(0, 1).
			MarginBottom(1)

	assistantStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(style.GreenStyle.GetForeground()).
			Padding(0, 1).
			MarginBottom(1)

	headerStyle = style.TitleStyle.
			MarginBottom(1)

	statusStyle = style.SubtleStyle

	errorStyle = style.RedStyle.
			Bold(true)

	spinnerStyle = style.BlueStyle
)

// View renders the UI
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	// Header
	header := headerStyle.Render(
		fmt.Sprintf("AI Chat: %s", m.conversationID),
	)

	// Status line
	statusText := fmt.Sprintf("Model: %s", m.client.model)
	if m.loading {
		statusText = fmt.Sprintf("%s %s", m.spinner.View(), "Thinking...")
	}
	status := statusStyle.Render(statusText)

	// Error display
	errorDisplay := ""
	if m.err != nil {
		errorDisplay = errorStyle.Render(fmt.Sprintf("Error: %s", m.err.Error()))
	}

	// Help text
	help := statusStyle.Render("Ctrl+Enter: send • Esc: quit")

	// Build layout
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		status,
		"",
		m.viewport.View(),
		"",
		m.textarea.View(),
		help,
		errorDisplay,
	)
}

// renderMessages renders all messages for the viewport
func (m Model) renderMessages() string {
	if len(m.messages) == 0 {
		return statusStyle.Render("Start the conversation by typing a message below...")
	}

	var renderedMessages []string

	for _, msg := range m.messages {
		var rendered string

		timeStr := msg.Time.Format("15:04:05")
		roleHeader := fmt.Sprintf("%s (%s)", strings.Title(msg.Role), timeStr)

		if msg.Role == "user" {
			content := lipgloss.NewStyle().Width(m.width - 8).Render(msg.Content)
			rendered = userStyle.Render(
				lipgloss.JoinVertical(
					lipgloss.Left,
					style.BlueStyle.Render(roleHeader),
					"",
					content,
				),
			)
		} else {
			content := lipgloss.NewStyle().Width(m.width - 8).Render(msg.Content)
			rendered = assistantStyle.Render(
				lipgloss.JoinVertical(
					lipgloss.Left,
					style.GreenStyle.Render(roleHeader),
					"",
					content,
				),
			)
		}

		renderedMessages = append(renderedMessages, rendered)
	}

	return lipgloss.JoinVertical(lipgloss.Left, renderedMessages...)
}
