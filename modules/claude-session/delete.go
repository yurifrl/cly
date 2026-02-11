package claudesession

import (
	"fmt"

	"github.com/spf13/cobra"
)

func deleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a saved session",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			sessions, err := Load(FilePath())
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			var names []string
			for _, e := range sessions {
				names = append(names, e.Name)
			}
			return names, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			filePath := FilePath()

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

			fmt.Printf("Deleted session %q\n", name)
			return nil
		},
	}
}
