package lg_tree_background

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
)

// renderBackgroundTree creates a dark background table of contents with pink items.
func renderBackgroundTree() string {
	pink := lipgloss.Color("#FF69B4")
	lightPink := lipgloss.Color("#FFB6C1")
	darkBg := lipgloss.Color("#1a1a2e")
	hotPink := lipgloss.Color("#FF1493")

	rootStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(hotPink)

	enumStyle := lipgloss.NewStyle().
		Foreground(lightPink)

	itemStyle := lipgloss.NewStyle().
		Foreground(pink)

	t := tree.Root("📖 Table of Contents").
		Child(
			tree.Root("1. Introduction").
				Child(
					"1.1 Getting Started",
					"1.2 Prerequisites",
					"1.3 Installation",
				),
			tree.Root("2. Core Concepts").
				Child(
					"2.1 Architecture",
					"2.2 Data Flow",
					"2.3 Configuration",
				),
			tree.Root("3. Advanced Topics").
				Child(
					"3.1 Performance Tuning",
					"3.2 Security",
					"3.3 Deployment",
				),
			tree.Root("4. Reference").
				Child(
					"4.1 API Documentation",
					"4.2 CLI Reference",
					"4.3 Troubleshooting",
				),
		).
		RootStyle(rootStyle).
		EnumeratorStyle(enumStyle).
		ItemStyle(itemStyle).
		Enumerator(tree.RoundedEnumerator)

	// Wrap the tree in a dark background block.
	block := lipgloss.NewStyle().
		Background(darkBg).
		Foreground(pink).
		Padding(1, 3).
		Bold(true)

	return block.Render(t.String())
}
