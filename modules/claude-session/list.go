package claudesession

import (
	"fmt"

	"github.com/spf13/cobra"
)

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List saved sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			sessions, err := Load(FilePath())
			if err != nil {
				return err
			}

			if len(sessions) == 0 {
				fmt.Println("No saved sessions")
				return nil
			}

			for _, e := range sessions {
				line := fmt.Sprintf("%s  id=%s  path=%s", e.Name, e.ID, e.Path)
				if e.Description != "" {
					line += fmt.Sprintf("  (%s)", e.Description)
				}
				fmt.Println(line)
			}
			return nil
		},
	}
}
