package lg_table_chess

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Register adds the lg-table-chess command to the parent command.
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "lg-table-chess",
		Short: "Lipgloss table chess board demo",
		Long:  "A chess board rendered as a styled table with file/rank labels using lipgloss/table and JoinHorizontal/JoinVertical",
		RunE:  run,
	}

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	fmt.Println(renderTable())
	return nil
}
