package claudesession

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func lsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List and open a session interactively",
		RunE: func(cmd *cobra.Command, args []string) error {
			sessions, err := Load(FilePath())
			if err != nil {
				return err
			}

			if len(sessions) == 0 {
				fmt.Println("No saved sessions")
				return nil
			}

			entry, err := runPicker(sessions, "Sessions")
			if err != nil {
				return err
			}
			if entry == nil {
				return nil
			}

			fmt.Printf("Resuming: %s\n", entry.Name)
			if err := os.Chdir(entry.Path); err != nil {
				return fmt.Errorf("chdir %s: %w", entry.Path, err)
			}

			return execClaude(entry)
		},
	}
}
