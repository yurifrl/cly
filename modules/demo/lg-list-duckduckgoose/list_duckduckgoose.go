package lg_list_duckduckgoose

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/list"
)

// renderDuckDuckGoose builds and returns a duck duck goose list with a custom
// enumerator that shows "Honk →" for Goose items and "•" for Duck items.
func renderDuckDuckGoose() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("220")).
		MarginBottom(1)

	duckColor := lipgloss.Color("178")  // warm yellow/orange
	gooseColor := lipgloss.Color("197") // bright magenta/red

	// Custom enumerator: "Honk →" for Goose, bullet for everything else.
	enumerator := func(items list.Items, index int) string {
		if items.At(index).Value() == "Goose" {
			return "Honk →"
		}
		return "•"
	}

	// Enumerator style: highlight Goose enumerators differently.
	enumeratorStyleFunc := func(items list.Items, index int) lipgloss.Style {
		if items.At(index).Value() == "Goose" {
			return lipgloss.NewStyle().
				Foreground(gooseColor).
				Bold(true).
				PaddingRight(1)
		}
		return lipgloss.NewStyle().
			Foreground(duckColor).
			PaddingRight(1)
	}

	// Item style: bold + bright for Goose, normal for Duck.
	itemStyleFunc := func(items list.Items, index int) lipgloss.Style {
		if items.At(index).Value() == "Goose" {
			return lipgloss.NewStyle().
				Foreground(gooseColor).
				Bold(true)
		}
		return lipgloss.NewStyle().
			Foreground(duckColor)
	}

	l := list.New(
		"Duck",
		"Duck",
		"Duck",
		"Goose",
		"Duck",
		"Duck",
		"Goose",
		"Duck",
	).
		Enumerator(enumerator).
		EnumeratorStyleFunc(enumeratorStyleFunc).
		ItemStyleFunc(itemStyleFunc)

	return titleStyle.Render("🦆 Duck Duck Goose") + "\n" + l.String()
}
