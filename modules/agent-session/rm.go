package agentsession

import (
	"fmt"

	"github.com/spf13/cobra"
)

func rmCmd() *cobra.Command {
	var flagFilter string
	var flagDryRun bool

	cmd := &cobra.Command{
		Use:   "rm [name]",
		Short: "Delete a saved session (JSON output)",
		Args:  cobra.MaximumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			sessions, err := loadScopedSessions(cmd)
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
			hasName := len(args) > 0
			hasFilter := flagFilter != ""

			if hasName && hasFilter {
				return fmt.Errorf("cannot use both <name> argument and --filter flag")
			}
			if !hasName && !hasFilter {
				return fmt.Errorf("provide a session name or use --filter")
			}

			// Load scoped sessions for matching
			sessions, err := loadScopedSessions(cmd)
			if err != nil {
				return err
			}

			// Load all sessions for actual deletion
			filePath := filePathFn()
			allSessions, err := Load(filePath)
			if err != nil {
				return err
			}

			// Find entries to delete
			type deleteTarget struct {
				Name     string `json:"name"`
				Provider string `json:"provider"`
				ID       string `json:"id"`
			}

			var targets []deleteTarget

			if hasName {
				name := args[0]
				for _, e := range sessions {
					if e.Name == name || e.ID == name {
						targets = append(targets, deleteTarget{
							Name:     e.Name,
							Provider: effectiveProvider(e),
							ID:       e.ID,
						})
						break
					}
				}
				if len(targets) == 0 {
					return fmt.Errorf("session %q not found", name)
				}
			} else {
				filtered := filterByName(sessions, flagFilter)
				for _, e := range filtered {
					targets = append(targets, deleteTarget{
						Name:     e.Name,
						Provider: effectiveProvider(e),
						ID:       e.ID,
					})
				}
				if len(targets) == 0 {
					return fmt.Errorf("no sessions matching %q", flagFilter)
				}
			}

			if flagDryRun {
				return jsonOut(cmd, map[string]interface{}{
					"would_delete": targets,
				})
			}

			// Perform deletion
			for _, t := range targets {
				allSessions = RemoveForProvider(allSessions, t.Provider, t.Name)
			}

			if err := Save(filePath, allSessions); err != nil {
				return err
			}

			return jsonOut(cmd, map[string]interface{}{
				"deleted": targets,
			})
		},
	}

	cmd.Flags().StringVarP(&flagFilter, "filter", "f", "", "Delete sessions matching name (case-insensitive substring)")
	cmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Show what would be deleted without deleting")

	return cmd
}
