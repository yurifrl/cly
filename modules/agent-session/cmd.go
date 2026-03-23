package agentsession

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const providerFlag = "provider"

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:     "agent-session",
		Aliases: []string{"as", "ags", "agents-sessions"},
		Short:   "Manage saved agent sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI(cmd)
		},
	}

	cmd.PersistentFlags().StringP(providerFlag, "p", "all", "Agent provider filter (e.g., claude, pi, all)")
	cmd.PersistentFlags().BoolP("all", "a", false, "Show all sessions (not just current directory)")
	cmd.PersistentFlags().String("directory", "", "Filter sessions by directory path")
	_ = cmd.RegisterFlagCompletionFunc(providerFlag, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return availableProviders(), cobra.ShellCompDirectiveNoFileComp
	})

	cmd.AddCommand(lsCmd())
	cmd.AddCommand(rmCmd())
	cmd.AddCommand(upsertCmd())
	cmd.AddCommand(resumeCmd())
	cmd.AddCommand(editCmd())
	cmd.AddCommand(tuiCmd())

	parent.AddCommand(cmd)
}

// loadScopedSessions loads sessions and applies provider + directory scoping.
// By default, scopes to current directory unless --all is set or --directory overrides.
func loadScopedSessions(cmd *cobra.Command) (Sessions, error) {
	sessions, err := Load(filePathFn())
	if err != nil {
		return nil, err
	}

	// Provider filter
	providerFilter := providerFilterFromCmd(cmd)
	if providerFilter != "" && providerFilter != "all" {
		sessions = filterByProvider(sessions, providerFilter)
	}

	// Directory scoping
	all, _ := cmd.Flags().GetBool("all")
	directory, _ := cmd.Flags().GetString("directory")

	if directory != "" {
		sessions = filterByPath(sessions, directory)
	} else if !all {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		sessions = filterByPath(sessions, cwd)
	}

	return sessions, nil
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

// providerFilterFromCmd returns the raw provider filter string (may be "all").
func providerFilterFromCmd(cmd *cobra.Command) string {
	providerName, err := cmd.Flags().GetString(providerFlag)
	if err != nil {
		return "all"
	}
	return normalizeProvider(providerName)
}

func ensureProviderSupportsYolo(provider Provider, yolo bool) error {
	if yolo && !providerSupportsYolo(provider) {
		return fmt.Errorf("provider %q does not support yolo", provider.Name)
	}
	return nil
}

// jsonOut marshals v as indented JSON and prints to cmd stdout.
func jsonOut(cmd *cobra.Command, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}
