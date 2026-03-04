package lg_tree_files

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Register adds the lg-tree-files command to the parent command.
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "lg-tree-files",
		Short: "Lipgloss tree file explorer demo",
		Long:  "A tree displaying a file system hierarchy using lipgloss/tree",
		RunE:  run,
	}

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	fmt.Println(renderFileTree())
	return nil
}
