package lg_table_languages

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Register adds the lg-table-languages command to the parent command.
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "lg-table-languages",
		Short: "Lipgloss table language greetings demo",
		Long:  "A table of multilingual greetings with formal and informal forms, using thick borders and purple styling via lipgloss/table",
		RunE:  run,
	}

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	fmt.Println(renderTable())
	return nil
}
