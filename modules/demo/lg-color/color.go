package lg_color

import (
	"os"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
)

func run(cmd *cobra.Command, args []string) error {
	hasDarkBG := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
	lightDark := lipgloss.LightDark(hasDarkBG)

	frameStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lightDark(lipgloss.Color("#C5ADF9"), lipgloss.Color("#864EFF"))).
		Padding(1, 3).
		Margin(1, 3)
	paragraphStyle := lipgloss.NewStyle().
		Width(40).
		MarginBottom(1).
		Align(lipgloss.Center)
	textStyle := lipgloss.NewStyle().
		Foreground(lightDark(lipgloss.Color("#696969"), lipgloss.Color("#bdbdbd")))
	keywordStyle := lipgloss.NewStyle().
		Foreground(lightDark(lipgloss.Color("#37CD96"), lipgloss.Color("#22C78A"))).
		Bold(true)

	activeButton := lipgloss.NewStyle().
		Padding(0, 3).
		Background(lipgloss.Color("#FF6AD2")).
		Foreground(lipgloss.Color("#FFFCC2"))
	inactiveButton := activeButton.
		Background(lightDark(lipgloss.Color("#988F95"), lipgloss.Color("#978692"))).
		Foreground(lightDark(lipgloss.Color("#FDFCE3"), lipgloss.Color("#FBFAE7")))

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
