package claudesession

import (
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:     "claude-sessions",
		Aliases: []string{"cs"},
		Short:   "Manage Claude sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(saveCmd())
	cmd.AddCommand(restoreCmd())
	cmd.AddCommand(listCmd())
	cmd.AddCommand(deleteCmd())

	parent.AddCommand(cmd)
}
