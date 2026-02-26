package claudesession

import (
	"fmt"

	"github.com/spf13/cobra"
)

func rmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <name>",
		Short: "Delete a saved session",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			sessions, err := Load(filePathFn())
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			names := make([]string, 0, len(sessions))
			for _, e := range sessions {
				names = append(names, e.Name)
			}
			return names, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			filePath := filePathFn()

			sessions, err := Load(filePath)
			if err != nil {
				return err
			}

			if FindByName(sessions, name) == nil {
				return fmt.Errorf("session %q not found", name)
			}

			sessions = Remove(sessions, name)

			if err := Save(filePath, sessions); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Deleted session %q\n", name)
			return nil
		},
	}
}
