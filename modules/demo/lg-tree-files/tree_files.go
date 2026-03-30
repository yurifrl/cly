package lg_tree_files

import (
	"os"
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/tree"
)

// renderFileTree walks the current working directory recursively and builds a
// styled tree. Entries whose names start with "." are skipped.
func renderFileTree() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "error: " + err.Error()
	}

	enumeratorStyle := lipgloss.NewStyle().
		Faint(true)

	itemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("99"))

	root := buildTree(cwd, 0)
	root.
		EnumeratorStyle(enumeratorStyle).
		ItemStyle(itemStyle)

	return root.String()
}

// buildTree reads the directory at path and returns a tree node. maxDepth
// limits recursion to keep output manageable.
func buildTree(path string, depth int) *tree.Tree {
	const maxDepth = 3

	name := filepath.Base(path)
	t := tree.Root(name)

	if depth >= maxDepth {
		return t
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return t
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if entry.IsDir() {
			child := buildTree(filepath.Join(path, entry.Name()), depth+1)
			t.Child(child)
		} else {
			t.Child(entry.Name())
		}
	}

	return t
}
