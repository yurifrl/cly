package lg_list_grocery

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Register adds the lg-list-grocery command to the parent command.
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "lg-list-grocery",
		Short: "Lipgloss list: grocery list with checkmarks and strikethrough",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(renderGroceryList())
			return nil
		},
	}
	parent.AddCommand(cmd)
}
