package claudesession

import (
	"fmt"

	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:     "claude-sessions",
		Aliases: []string{"cs"},
		Short:   "Manage Claude sessions (TODO)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("TODO: not implemented yet")
			return nil
		},
	}

	parent.AddCommand(cmd)
}
