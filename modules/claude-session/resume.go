package claudesession

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

func resumeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resume <name|id>",
		Short: "Resume a saved session by name or ID",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			sessions, err := Load(filePathFn())
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
			query := args[0]
			sessions, err := Load(filePathFn())
			if err != nil {
				return err
			}

			entry := findEntry(sessions, query)
			if entry == nil {
				return fmt.Errorf("session %q not found", query)
			}

			return resumeEntry(entry)
		},
	}
}

func resumeEntry(entry *Entry) error {
	entry.SavedAt = time.Now()
	sessions, err := Load(filePathFn())
	if err != nil {
		return err
	}
	sessions[entry.Name] = *entry
	if err := Save(filePathFn(), sessions); err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	if cwd != entry.Path {
		if err := os.Chdir(entry.Path); err != nil {
			return fmt.Errorf("chdir %s: %w", entry.Path, err)
		}
	}
	return execClaude(entry)
}
