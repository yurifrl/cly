package lg_table_ansi

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Register adds the lg-table-ansi command to the parent command.
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "lg-table-ansi",
		Short: "Lipgloss table with ANSI colors demo",
		Long:  "A simple 3-row table with a dim second column using lipgloss/table",
		RunE:  run,
	}

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	fmt.Println(renderTable())
	return nil
}
