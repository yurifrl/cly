package lg_list_simple

import (
	"github.com/charmbracelet/lipgloss/list"
)

// renderList creates a simple nested list with a Roman numeral sub-enumerator.
func renderList() string {
	l := list.New(
		"A",
		"B",
		"C",
		list.New("D", "E", "F").Enumerator(list.Roman),
		"G",
	)
	return l.String()
}
