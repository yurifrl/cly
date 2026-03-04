package lg_tree_simple

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Register adds the lg-tree-simple command to the parent command.
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "lg-tree-simple",
		Short: "Lipgloss tree simple demo",
		Long:  "A simple tree with nested children using lipgloss/tree",
		RunE:  run,
	}

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	fmt.Println(renderTree())
	return nil
}
