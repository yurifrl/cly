package lg_table_mindy

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Register adds the lg-table-mindy command to the parent command.
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "lg-table-mindy",
		Short: "Lipgloss table 256 ANSI color swatch demo",
		Long:  "A 256-color ANSI swatch grid rendered as a hidden-border table using lipgloss/table",
		RunE:  run,
	}

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	fmt.Println(renderTable())
	return nil
}
