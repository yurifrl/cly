package lg_tree_simple

import (
	"charm.land/lipgloss/v2/tree"
)

// renderTree creates a simple nested tree with OS families as children.
func renderTree() string {
	t := tree.Root("Operating Systems").
		Child(
			"macOS",
			tree.Root("Linux").
				Child(
					"Ubuntu",
					"Fedora",
					"Arch",
					"NixOS",
				),
			tree.Root("BSD").
				Child(
					"FreeBSD",
					"OpenBSD",
					"NetBSD",
				),
		)
	return t.String()
}
