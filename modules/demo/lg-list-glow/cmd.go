package lg_list_glow

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Register adds the lg-list-glow command to the parent command.
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "lg-list-glow",
		Short: "Lipgloss list with selection highlight and multi-line items",
		Long:  "Demonstrates a document-style list with multi-line items, selection highlighting via background color, and a custom enumerator showing │ for the selected item.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(renderGlowList())
			return nil
		},
	}
	parent.AddCommand(cmd)
}
