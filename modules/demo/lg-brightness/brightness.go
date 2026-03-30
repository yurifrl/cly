package lg_brightness

import (
	"image/color"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
)

func run(cmd *cobra.Command, args []string) error {
	hasDarkBG := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
	lightDark := lipgloss.LightDark(hasDarkBG)

	baseColors := map[string]color.Color{
		"Red":   lipgloss.Color("#FF0000"),
		"Blue":  lipgloss.Color("#0066FF"),
		"Green": lipgloss.Color("#00FF00"),
		"Gray":  lipgloss.Color("#808080"),
	}

	percentage := 0.05
	steps := 20

	colorNameStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lightDark(lipgloss.Color("#2D3748"), lipgloss.Color("#E2E8F0")))

	var content strings.Builder

	for name, baseColor := range baseColors {
		content.WriteString(colorNameStyle.Render(name))
		content.WriteString("\n")

		var lightenedBox strings.Builder
		lightenedBox.WriteString("Lightened: ")
		for i := range steps {
			lightenedBox.WriteString(
				lipgloss.NewStyle().
					Foreground(lipgloss.Lighten(baseColor, percentage*(float64(i)+1))).
					Render("\u2588\u2588"),
			)
		}
		content.WriteString(lightenedBox.String())
		content.WriteString("\n")

		var darkenedBox strings.Builder
		darkenedBox.WriteString("Darkened:  ")
		for i := range steps {
			darkenedBox.WriteString(
				lipgloss.NewStyle().
					Foreground(lipgloss.Darken(baseColor, percentage*(float64(i)+1))).
					Render("\u2588\u2588"),
			)
		}
		content.WriteString(darkenedBox.String())
		content.WriteString("\n\n")
	}

	lipgloss.Println(content.String())
	return nil
}
