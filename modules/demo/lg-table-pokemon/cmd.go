package lg_table_pokemon

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Register adds the lg-table-pokemon command to the parent command.
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "lg-table-pokemon",
		Short: "Lipgloss table Pokémon demo",
		Long:  "A colorful Pokédex table with 28 Pokémon, dual types, Japanese names, and type-based cell coloring using lipgloss/table",
		RunE:  run,
	}

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	fmt.Println(renderTable())
	return nil
}
