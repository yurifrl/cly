package lg_tree_toggle

import (
	"fmt"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/tree"
)

// Dir represents a directory node with open/closed state.
type Dir struct {
	name     string
	open     bool
	children []any
}

// String renders the directory with ▼ (open) or ▶ (closed) prefix.
func (d Dir) String() string {
	if d.open {
		return "▼ " + d.name
	}
	return "▶ " + d.name
}

// File represents a file node.
type File struct {
	name string
}

// String renders the file name with a document prefix.
func (f File) String() string {
	return "📄 " + f.name
}

// renderToggleTree creates trees showing directory toggle states with custom
// fmt.Stringer types, plus a purple background block around the output.
func renderToggleTree() string {
	rootStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("230"))

	itemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("230"))

	enumStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("99"))

	// Expanded (open) tree.
	openTree := tree.Root(Dir{name: "project", open: true}).
		Child(
			tree.Root(Dir{name: "src", open: true}).
				Child(
					File{name: "main.go"},
					File{name: "handler.go"},
					File{name: "config.go"},
				),
			tree.Root(Dir{name: "test", open: true}).
				Child(
					File{name: "main_test.go"},
					File{name: "handler_test.go"},
				),
			tree.Root(Dir{name: "docs", open: false}).
				Child().Hide(true),
			File{name: "go.mod"},
			File{name: "README.md"},
		).
		RootStyle(rootStyle).
		ItemStyle(itemStyle).
		EnumeratorStyle(enumStyle).
		Enumerator(tree.RoundedEnumerator)

	// Collapsed tree.
	closedTree := tree.Root(Dir{name: "project", open: false}).
		Child(
			Dir{name: "src", open: false},
			Dir{name: "test", open: false},
			Dir{name: "docs", open: false},
			File{name: "go.mod"},
			File{name: "README.md"},
		).
		RootStyle(rootStyle).
		ItemStyle(itemStyle).
		EnumeratorStyle(enumStyle).
		Enumerator(tree.RoundedEnumerator)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("230")).
		MarginBottom(1)

	content := headerStyle.Render("Expanded:") + "\n" +
		openTree.String() + "\n\n" +
		headerStyle.Render("Collapsed:") + "\n" +
		closedTree.String()

	// Purple background block wrapping the entire output.
	block := lipgloss.NewStyle().
		Background(lipgloss.Color("53")).
		Foreground(lipgloss.Color("230")).
		Padding(1, 2)

	return fmt.Sprint(block.Render(content))
}
