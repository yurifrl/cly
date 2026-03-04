package lg_tree_styles

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Register adds the lg-tree-styles command to the parent command.
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "lg-tree-styles",
		Short: "Lipgloss tree conditional styling demo",
		Long:  "A tree with conditional styling using ItemStyleFunc and EnumeratorStyleFunc from lipgloss/tree",
		RunE:  run,
	}

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	fmt.Println(renderStyledTree())
	return nil
}
