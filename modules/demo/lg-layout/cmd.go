package lg_layout

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Register adds the lg-layout command to the parent command.
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "lg-layout",
		Short: "Lipgloss layout demo",
		Long:  "A full layout showcase with tabs, gradient title, description, dialog, 3-column lists, history, and status bar",
		RunE:  run,
	}

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	fmt.Println(renderLayout())
	return nil
}
