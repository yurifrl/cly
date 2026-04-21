package agentsession

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func cleanupCmd() *cobra.Command {
	var flagDryRun bool

	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Permanently remove all soft-deleted sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := filePathFn()
			allSessions, err := Load(filePath)
			if err != nil {
				return err
			}

			deleted := filterOnlyDeleted(allSessions)
			count := len(deleted)

			if flagDryRun {
				names := make([]string, 0, count)
				for _, e := range deleted {
					names = append(names, e.Name)
				}
				return jsonOut(cmd, map[string]interface{}{
					"would_cleanup": names,
				})
			}

			if count == 0 {
				return jsonOut(cmd, map[string]interface{}{
					"cleaned_up": 0,
					"message":    "no soft-deleted sessions to clean up",
				})
			}

			fmt.Fprintf(cmd.ErrOrStderr(), "Permanently remove %d soft-deleted session(s)? [y/N] ", count)
			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			if strings.TrimSpace(strings.ToLower(answer)) != "y" {
				return jsonOut(cmd, map[string]interface{}{
					"cancelled": true,
				})
			}

			allSessions = CleanupDeleted(allSessions)
			if err := Save(filePath, allSessions); err != nil {
				return err
			}
			return jsonOut(cmd, map[string]interface{}{
				"cleaned_up": count,
			})
		},
	}

	cmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Show what would be cleaned up without removing")

	return cmd
}
