package lg_list_duckduckgoose

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Register adds the lg-list-duckduckgoose command to the parent command.
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "lg-list-duckduckgoose",
		Short: "Lipgloss list: duck duck goose with custom enumerator",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(renderDuckDuckGoose())
			return nil
		},
	}
	parent.AddCommand(cmd)
}
