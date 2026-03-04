package lg_list_simple

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Register adds the lg-list-simple command to the parent command.
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "lg-list-simple",
		Short: "Lipgloss list simple demo",
		Long:  "A simple nested list with Roman numeral sub-enumerator using lipgloss/list",
		RunE:  run,
	}

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	fmt.Println(renderList())
	return nil
}
