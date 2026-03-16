package agentsession

import (
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func saveCmd() *cobra.Command {
	var flagDesc string
	cmd := &cobra.Command{
		Use:   "save <name> [id]",
		Short: "Save or update a session",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			provider, err := providerFromCmd(cmd)
			if err != nil {
				return err
			}

			name := args[0]
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}

			filePath := filePathFn()
			sessions, err := Load(filePath)
			if err != nil {
				return err
			}

			entry := FindByNameForProvider(sessions, provider.Name, name)
			if entry == nil {
				id := uuid.New().String()
				if len(args) > 1 {
					id = args[1]
				}
				e := Entry{
					Name:        name,
					Provider:    provider.Name,
					ID:          id,
					Path:        cwd,
					Description: flagDesc,
					SavedAt:     time.Now(),
				}
				entry = &e
			} else {
				if len(args) > 1 {
					entry.ID = args[1]
				}
				if flagDesc != "" {
					entry.Description = flagDesc
				}
				entry.Provider = provider.Name
				entry.Path = cwd
				entry.SavedAt = time.Now()
			}

			upsertEntry(sessions, *entry)
			if err := Save(filePath, sessions); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Saved %s session %q (id=%s)\n", provider.Name, name, entry.ID)
			return nil
		},
	}
	cmd.Flags().StringVarP(&flagDesc, "description", "d", "", "Session description")
	return cmd
}
