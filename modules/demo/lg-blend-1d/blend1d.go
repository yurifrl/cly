package lg_blend_1d

import (
	"image/color"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
)

var gradients = [][]color.Color{
	{
		lipgloss.Color("#FF6B6B"),
		lipgloss.Color("#FFB74D"),
		lipgloss.Color("#FFDFBA"),
	},
	{
		lipgloss.Color("#0077B6"),
		lipgloss.Color("#48CAE4"),
		lipgloss.Color("#ADE8F4"),
	},
	{
		lipgloss.Color("#228B22"),
		lipgloss.Color("#90EE90"),
		lipgloss.Color("#FFFFE0"),
	},
	{
		lipgloss.Color("#9370DB"),
		lipgloss.Color("#DDA0DD"),
		lipgloss.Color("#FFB6C1"),
	},
	{
		lipgloss.Color("#9900FF"),
		lipgloss.Color("#00FA68"),
		lipgloss.Color("#ED5353"),
	},
}

func run(cmd *cobra.Command, args []string) error {
	hasDarkBG := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
	lightDark := lipgloss.LightDark(hasDarkBG)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lightDark(lipgloss.Color("#2D3748"), lipgloss.Color("#E2E8F0"))).
		MarginBottom(1).
		Align(lipgloss.Center)

	gradientStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lightDark(lipgloss.Color("#718096"), lipgloss.Color("#A0AEC0")))

	var content strings.Builder

	content.WriteString(titleStyle.Render("Color Gradient Examples with Blend1D"))
	content.WriteString("\n")

	for _, gradient := range gradients {
		blendedColors := lipgloss.Blend1D(40, gradient...)

		var gradientBar strings.Builder
		for _, c := range blendedColors {
			blockStyle := lipgloss.NewStyle().Foreground(c)
			gradientBar.WriteString(blockStyle.Render("\u2588"))
		}

		content.WriteString(gradientStyle.Render(gradientBar.String()))
		content.WriteString("\n")
	}

	lipgloss.Println(content.String())
	return nil
}
