package lg_tree_makeup

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
)

// renderMakeupTree creates a tree of makeup brands and their products.
func renderMakeupTree() string {
	rootStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF69B4"))

	enumeratorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#C9A0DC"))

	itemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FADADD"))

	t := tree.Root("⁜ Makeup").
		Child(
			"Glossier",
			tree.Root("Fenty Beauty").
				Child(
					"Gloss Bomb Universal Lip Luminizer",
					"Hot Cheeks Velour Blushlighter",
				),
			"Nyx",
			tree.Root("Mac").
				Child(
					"Lipstick",
					"Foundation",
					"Mascara",
				),
			"Milk",
		).
		Enumerator(tree.RoundedEnumerator).
		RootStyle(rootStyle).
		EnumeratorStyle(enumeratorStyle).
		ItemStyle(itemStyle)

	return t.String()
}
