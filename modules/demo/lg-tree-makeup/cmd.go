package lg_tree_makeup

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Register adds the lg-tree-makeup command to the parent command.
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "lg-tree-makeup",
		Short: "Lipgloss tree makeup brands demo",
		Long:  "A tree of makeup brands with nested product lines using lipgloss/tree",
		RunE:  run,
	}

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	fmt.Println(renderMakeupTree())
	return nil
}
