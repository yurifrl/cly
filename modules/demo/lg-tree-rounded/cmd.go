package lg_tree_rounded

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Register adds the lg-tree-rounded command to the parent command.
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "lg-tree-rounded",
		Short: "Lipgloss tree rounded enumerator demo",
		Long:  "A tree using the rounded enumerator for softer branch indicators using lipgloss/tree",
		RunE:  run,
	}

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	fmt.Println(renderRoundedTree())
	return nil
}
