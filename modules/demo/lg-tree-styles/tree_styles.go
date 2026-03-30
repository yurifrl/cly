package lg_tree_styles

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/tree"
)

// renderStyledTree creates a tree with per-subtree styling.
// Each subtree uses a different enumerator color to visually distinguish branches.
func renderStyledTree() string {
	rootStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF69B4"))

	// Subtree 1: green-themed enumerator.
	greenEnum := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF7F"))
	greenItems := lipgloss.NewStyle().Foreground(lipgloss.Color("#98FB98"))

	frontend := tree.Root("Frontend").
		Child("React", "Vue", "Svelte").
		Enumerator(tree.RoundedEnumerator).
		EnumeratorStyle(greenEnum).
		ItemStyle(greenItems)

	// Subtree 2: blue-themed enumerator.
	blueEnum := lipgloss.NewStyle().Foreground(lipgloss.Color("#00BFFF"))
	blueItems := lipgloss.NewStyle().Foreground(lipgloss.Color("#87CEFA"))

	backend := tree.Root("Backend").
		Child("Go", "Rust", "Python").
		Enumerator(tree.RoundedEnumerator).
		EnumeratorStyle(blueEnum).
		ItemStyle(blueItems)

	// Subtree 3: orange-themed enumerator.
	orangeEnum := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8C00"))
	orangeItems := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFDAB9"))

	devops := tree.Root("DevOps").
		Child("Docker", "Kubernetes", "Terraform").
		Enumerator(tree.RoundedEnumerator).
		EnumeratorStyle(orangeEnum).
		ItemStyle(orangeItems)

	// Subtree 4: purple-themed enumerator.
	purpleEnum := lipgloss.NewStyle().Foreground(lipgloss.Color("#BA55D3"))
	purpleItems := lipgloss.NewStyle().Foreground(lipgloss.Color("#DDA0DD"))

	databases := tree.Root("Databases").
		Child("PostgreSQL", "Redis", "MongoDB").
		Enumerator(tree.RoundedEnumerator).
		EnumeratorStyle(purpleEnum).
		ItemStyle(purpleItems)

	t := tree.Root("🛠  Tech Stack").
		Child(frontend, backend, devops, databases).
		RootStyle(rootStyle).
		Enumerator(tree.RoundedEnumerator)

	return t.String()
}
