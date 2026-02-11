package claudesession

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func saveCmd() *cobra.Command {
	var id, name, description string

	cmd := &cobra.Command{
		Use:   "save",
		Short: "Save a Claude session",
		Long:  "Save a session: save -i <id> -n <name> [-d <description>] or save <id> <name> [description]",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Accept positional args as fallback
			if id == "" && len(args) > 0 {
				id = args[0]
			}
			if name == "" && len(args) > 1 {
				name = args[1]
			}
			if description == "" && len(args) > 2 {
				description = args[2]
			}

			if id == "" || name == "" {
				return fmt.Errorf("id and name are required")
			}

			path, _ := os.Getwd()
			filePath := FilePath()

			sessions, err := Load(filePath)
			if err != nil {
				return err
			}

			key := MakeKey(path, id)
			sessions[key] = Entry{
				ID:          id,
				Name:        name,
				Path:        path,
				Description: description,
			}

			if err := Save(filePath, sessions); err != nil {
				return err
			}

			fmt.Printf("Saved session %q (id=%s)\n", name, id)
			return nil
		},
	}

	cmd.Flags().StringVarP(&id, "id", "i", "", "Session ID")
	cmd.Flags().StringVarP(&name, "name", "n", "", "Session name")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Session description")

	return cmd
}
