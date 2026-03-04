package lg_tree_toggle

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Register adds the lg-tree-toggle command to the parent command.
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "lg-tree-toggle",
		Short: "Lipgloss tree hide/show toggle demo",
		Long:  "A tree demonstrating hidden nodes using Hide() and SetHidden() from lipgloss/tree",
		RunE:  run,
	}

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	fmt.Println(renderToggleTree())
	return nil
}
