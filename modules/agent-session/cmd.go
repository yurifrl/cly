package agentsession

import (
	"fmt"

	"github.com/spf13/cobra"
)

const providerFlag = "provider"

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:     "agent-session",
		Aliases: []string{"as", "ags", "agents-sessions"},
		Short:   "Manage saved agent sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLS(cmd, false)
		},
	}

	cmd.PersistentFlags().StringP(providerFlag, "p", defaultProvider, "Agent provider (e.g., claude, pi)")
	_ = cmd.RegisterFlagCompletionFunc(providerFlag, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return availableProviders(), cobra.ShellCompDirectiveNoFileComp
	})

	cmd.AddCommand(lsCmd())
	cmd.AddCommand(rmCmd())
	cmd.AddCommand(saveCmd())
	cmd.AddCommand(resumeCmd())
	cmd.AddCommand(editCmd())

	parent.AddCommand(cmd)
}

func providerFromCmd(cmd *cobra.Command) (Provider, error) {
	providerName, err := cmd.Flags().GetString(providerFlag)
	if err != nil {
		return Provider{}, err
	}
	provider, err := providerByName(providerName)
	if err != nil {
		return Provider{}, err
	}
	return provider, nil
}

func ensureProviderSupportsYolo(provider Provider, yolo bool) error {
	if yolo && !providerSupportsYolo(provider) {
		return fmt.Errorf("provider %q does not support yolo", provider.Name)
	}
	return nil
}
