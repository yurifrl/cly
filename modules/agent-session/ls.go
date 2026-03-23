package agentsession

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func lsCmd() *cobra.Command {
	var flagFilter string
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List sessions as JSON (current dir by default, -a for all)",
		RunE: func(cmd *cobra.Command, args []string) error {
			sessions, err := loadScopedSessions(cmd)
			if err != nil {
				return err
			}

			if flagFilter != "" {
				sessions = filterByName(sessions, flagFilter)
			}

			if len(sessions) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "[]")
				return nil
			}

			entries := sortedEntries(sessions, SortDateDesc)
			rows := make([]lsRow, 0, len(entries))
			for _, e := range entries {
				rows = append(rows, lsRow{
					Name:        e.Name,
					Provider:    effectiveProvider(e),
					SavedAt:     formatSavedAt(e.SavedAt),
					ID:          e.ID,
					Path:        e.Path,
					Description: e.Description,
					Meta:        e.Meta,
				})
			}

			data, err := json.MarshalIndent(rows, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		},
	}
	cmd.Flags().StringVarP(&flagFilter, "filter", "f", "", "Filter sessions by name (case-insensitive substring)")
	return cmd
}

type lsRow struct {
	Name        string            `json:"name,omitempty"`
	Provider    string            `json:"provider"`
	SavedAt     string            `json:"saved_at,omitempty"`
	ID          string            `json:"id"`
	Path        string            `json:"path"`
	Description string            `json:"description,omitempty"`
	Meta        map[string]string `json:"meta,omitempty"`
}

func formatSavedAt(t interface{ IsZero() bool; Format(string) string }) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04")
}
