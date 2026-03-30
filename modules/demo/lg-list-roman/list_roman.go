package lg_list_roman

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/list"
)

// renderRomanList builds a Roman numeral enumerated list of makeup brands
// with styled enumerator and items.
func renderRomanList() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF69B4")).
		MarginBottom(1)

	enumeratorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#C9A0DC")).
		PaddingRight(1)

	itemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FADADD"))

	l := list.New(
		"MAC Cosmetics",
		"NARS",
		"Charlotte Tilbury",
		"Fenty Beauty",
		"Urban Decay",
		"Too Faced",
		"Rare Beauty",
		"Pat McGrath Labs",
	).
		Enumerator(list.Roman).
		EnumeratorStyle(enumeratorStyle).
		ItemStyle(itemStyle)

	return titleStyle.Render("💄 Makeup Brands") + "\n" + l.String()
}
