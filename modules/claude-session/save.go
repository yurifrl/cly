package claudesession

import (
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func saveCmd() *cobra.Command {
	var flagDesc string
	var flagID string
	cmd := &cobra.Command{
		Use:   "save <name> [id]",
		Short: "Save or update a session",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}

			// Resolve id from positional arg or flag
			var id string
			if len(args) > 1 {
				id = args[1]
			}
			if flagID != "" {
				id = flagID
			}

			filePath := filePathFn()
			sessions, err := Load(filePath)
			if err != nil {
				return err
			}

			// Find existing: by id first (if provided), then by name
			var entry *Entry
			if id != "" {
				entry = FindByID(sessions, id)
			}
			if entry == nil {
				entry = FindByName(sessions, name)
			}

			if entry == nil {
				// Create new
				if id == "" {
					id = uuid.New().String()
				}
				e := Entry{
					Name:        name,
					ID:          id,
					Path:        cwd,
					Description: flagDesc,
					SavedAt:     time.Now(),
				}
				entry = &e
			} else {
				// Update existing — remove old key if name changed
				if entry.Name != name {
					delete(sessions, entry.Name)
					entry.Name = name
				}
				if id != "" {
					entry.ID = id
				}
				if flagDesc != "" {
					entry.Description = flagDesc
				}
				entry.Path = cwd
				entry.SavedAt = time.Now()
			}

			sessions[name] = *entry
			if err := Save(filePath, sessions); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Saved session %q (id=%s)\n", name, entry.ID)
			return nil
		},
	}
	cmd.Flags().StringVarP(&flagDesc, "description", "d", "", "Session description")
	cmd.Flags().StringVar(&flagID, "id", "", "Session ID to find/update (looks up by ID before name)")
	return cmd
}
