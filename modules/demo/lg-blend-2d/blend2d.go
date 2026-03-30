package lg_blend_2d

import (
	"fmt"
	"image/color"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
)

func run(cmd *cobra.Command, args []string) error {
	hasDarkBG := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
	lightDark := lipgloss.LightDark(hasDarkBG)

	gradients := []struct {
		name  string
		stops []color.Color
		angle float64
	}{
		{
			name:  "Sunset Diagonal",
			stops: []color.Color{lipgloss.Color("#FF6B6B"), lipgloss.Color("#FFB74D"), lipgloss.Color("#FFDFBA")},
			angle: 45,
		},
		{
			name:  "Ocean Wave",
			stops: []color.Color{lipgloss.Color("#0077B6"), lipgloss.Color("#48CAE4"), lipgloss.Color("#ADE8F4")},
			angle: 90,
		},
		{
			name:  "Forest Mist",
			stops: []color.Color{lipgloss.Color("#228B22"), lipgloss.Color("#90EE90"), lipgloss.Color("#FFFFE0")},
			angle: 135,
		},
		{
			name:  "Purple Dream",
			stops: []color.Color{lipgloss.Color("#9370DB"), lipgloss.Color("#DDA0DD"), lipgloss.Color("#FFB6C1")},
			angle: 180,
		},
		{
			name:  "Fire Gradient",
			stops: []color.Color{lipgloss.Color("#FF0000"), lipgloss.Color("#FFA500"), lipgloss.Color("#FFFF00")},
			angle: 225,
		},
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lightDark(lipgloss.Color("#2D3748"), lipgloss.Color("#E2E8F0"))).
		MarginBottom(1).
		Align(lipgloss.Center)

	gradientStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lightDark(lipgloss.Color("#718096"), lipgloss.Color("#A0AEC0"))).
		MarginBottom(1)

	gradientNameStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lightDark(lipgloss.Color("#4A5568"), lipgloss.Color("#CBD5E0"))).
		MarginBottom(1)

	var content strings.Builder

	content.WriteString(titleStyle.Render("2D Color Gradient Examples with Blend2D"))
	content.WriteString("\n\n")

	for _, gradient := range gradients {
		width, height := 30, 12
		blendedColors := lipgloss.Blend2D(width, height, gradient.angle, gradient.stops...)

		var gradientBox strings.Builder
		for y := range height {
			for x := range width {
				index := y*width + x
				gradientBox.WriteString(
					lipgloss.NewStyle().
						Foreground(blendedColors[index]).
						Render("\u2588"),
				)
			}
			if y < height-1 {
				gradientBox.WriteString("\n")
			}
		}

		content.WriteString(gradientNameStyle.Render(fmt.Sprintf("%s (Angle: %.0f\u00b0)", gradient.name, gradient.angle)))
		content.WriteString("\n")
		content.WriteString(gradientStyle.Render(gradientBox.String()))
		content.WriteString("\n")
	}

	lipgloss.Println(content.String())
	return nil
}
