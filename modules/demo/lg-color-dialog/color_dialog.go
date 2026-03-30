package lg_color_dialog

import (
	"charm.land/lipgloss/v2"
)

// renderDialog creates a banana-themed dialog box with a purple rounded border,
// paragraph text, and Yes/No buttons using JoinHorizontal, all centered with JoinVertical.
func renderDialog() string {
	// Purple color for the border
	purple := lipgloss.Color("99")

	// Dialog box style with rounded border in purple
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(purple).
		Padding(1, 2)

	// Paragraph text
	paragraph := lipgloss.NewStyle().
		Width(40).
		Align(lipgloss.Center).
		Render("Are you sure you want to eat that banana?\n\nIt's the last one in the bunch!")

	// Active button style (Yes - highlighted)
	activeButtonStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("212")).
		Foreground(lipgloss.Color("0")).
		Padding(0, 3).
		MarginRight(1)

	// Inactive button style (No - dimmed)
	inactiveButtonStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("240")).
		Foreground(lipgloss.Color("0")).
		Padding(0, 3)

	// Render buttons
	yesBtn := activeButtonStyle.Render("Yes")
	noBtn := inactiveButtonStyle.Render("No")

	// Join buttons horizontally, centered
	buttons := lipgloss.JoinHorizontal(lipgloss.Center, yesBtn, noBtn)

	// Join paragraph and buttons vertically, centered
	content := lipgloss.JoinVertical(lipgloss.Center, paragraph, "", buttons)

	// Wrap in the dialog box
	return dialogStyle.Render(content)
}
