package agentsession

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
			provider, err := providerFromCmd(cmd)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			sessions, err := Load(filePathFn())
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			names := make([]string, 0, len(sessions))
			for _, e := range filterByProvider(sessions, provider.Name) {
				names = append(names, e.Name)
			}
			return names, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			provider, err := providerFromCmd(cmd)
			if err != nil {
				return err
			}

			name := args[0]
			filePath := filePathFn()

			sessions, err := Load(filePath)
			if err != nil {
				return err
			}

			if FindByNameForProvider(sessions, provider.Name, name) == nil {
				return fmt.Errorf("%s session %q not found", provider.Name, name)
			}

			sessions = RemoveForProvider(sessions, provider.Name, name)

			if err := Save(filePath, sessions); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Deleted %s session %q\n", provider.Name, name)
			return nil
		},
	}
}
