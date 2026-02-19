package claudesession

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/pkg/session"
)

func execClaude(entry *Entry) error {
	return session.ExecClaude([]string{"-r", entry.ID})
}

func restoreCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restore [name]",
		Short: "Restore a saved session (interactive if no name given)",
		Args:  cobra.RangeArgs(0, 1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			sessions, err := Load(FilePath())
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			var names []string
			for _, e := range sessions {
				names = append(names, e.Name)
			}
			return names, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			sessions, err := Load(FilePath())
			if err != nil {
				return err
			}

			var entry *Entry
			if len(args) == 1 {
				entry = FindByName(sessions, args[0])
				if entry == nil {
					return fmt.Errorf("session %q not found", args[0])
				}
				fmt.Printf("Resuming: %s\n", entry.Name)
			} else {
				entry, err = runPicker(sessions, "Sessions", SortDate)
				if err != nil {
					return err
				}
				if entry == nil {
					return nil
				}
			}

			if err := os.Chdir(entry.Path); err != nil {
				return fmt.Errorf("chdir %s: %w", entry.Path, err)
			}

			return execClaude(entry)
		},
	}
}
