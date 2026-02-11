package claudesession

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/pkg/session"
)

func restoreCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restore <name>",
		Short: "Restore a saved session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			sessions, err := Load(FilePath())
			if err != nil {
				return err
			}

			entry := FindByName(sessions, name)
			if entry == nil {
				return fmt.Errorf("session %q not found", name)
			}

			fmt.Printf("Resuming session: %s\n", name)

			if err := os.Chdir(entry.Path); err != nil {
				return fmt.Errorf("chdir %s: %w", entry.Path, err)
			}

			return session.ExecClaude([]string{"-r", entry.ID})
		},
	}
}
