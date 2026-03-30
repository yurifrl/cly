package lg_compat

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
	"github.com/spf13/cobra"
)

var (
	frameColor      = compat.AdaptiveColor{Light: lipgloss.Color("#C5ADF9"), Dark: lipgloss.Color("#864EFF")}
	textColor       = compat.AdaptiveColor{Light: lipgloss.Color("#696969"), Dark: lipgloss.Color("#bdbdbd")}
	keywordColor    = compat.AdaptiveColor{Light: lipgloss.Color("#37CD96"), Dark: lipgloss.Color("#22C78A")}
	inactiveBgColor = compat.AdaptiveColor{Light: lipgloss.Color("#988F95"), Dark: lipgloss.Color("#978692")}
	inactiveFgColor = compat.AdaptiveColor{Light: lipgloss.Color("#FDFCE3"), Dark: lipgloss.Color("#FBFAE7")}
)

func run(cmd *cobra.Command, args []string) error {
	frameStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(frameColor).
		Padding(1, 3).
		Margin(1, 3)
	paragraphStyle := lipgloss.NewStyle().
		Width(40).
		MarginBottom(1).
		Align(lipgloss.Center)
	textStyle := lipgloss.NewStyle().
		Foreground(textColor)
	keywordStyle := lipgloss.NewStyle().
		Foreground(keywordColor).
		Bold(true)

	activeButton := lipgloss.NewStyle().
		Padding(0, 3).
		Background(lipgloss.Color("#FF6AD2")).
		Foreground(lipgloss.Color("#FFFCC2"))
	inactiveButton := activeButton.
		Background(inactiveBgColor).
		Foreground(inactiveFgColor)

	text := paragraphStyle.Render(
		textStyle.Render("Are you sure you want to eat that ") +
			keywordStyle.Render("moderatly ripe") +
			textStyle.Render(" banana?"),
	)
	buttons := activeButton.Render("Yes") + "  " + inactiveButton.Render("No")
	block := frameStyle.Render(
		lipgloss.JoinVertical(lipgloss.Center, text, buttons),
	)

	lipgloss.Println(block)
	return nil
}
