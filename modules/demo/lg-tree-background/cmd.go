package lg_tree_background

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Register adds the lg-tree-background command to the parent command.
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "lg-tree-background",
		Short: "Lipgloss tree background color demo",
		Long:  "A tree with background-colored items and styled enumerators using lipgloss/tree",
		RunE:  run,
	}

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	fmt.Println(renderBackgroundTree())
	return nil
}
