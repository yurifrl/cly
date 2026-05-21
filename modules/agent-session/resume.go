package agentsession

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
			sessions, err := loadScopedSessions(cmd)
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
			provider, err := providerFromCmd(cmd)
			if err != nil {
				return err
			}

			query := args[0]
			sessions, err := loadScopedSessions(cmd)
			if err != nil {
				return err
			}

			entry := findEntry(sessions, provider.Name, query)
			if entry == nil {
				return fmt.Errorf("%s session %q not found", provider.Name, query)
			}

			return resumeEntry(entry, provider, false)
		},
	}
}

func resumeEntry(entry *Entry, provider Provider, yolo bool) error {
	if err := ensureProviderSupportsYolo(provider, yolo); err != nil {
		return err
	}

	entry.Provider = provider.Name
	entry.SavedAt = time.Now()
	sessions, err := Load(filePathFn())
	if err != nil {
		return err
	}
	upsertEntry(sessions, *entry)
	if err := Save(filePathFn(), sessions); err != nil {
		return err
	}

	if entry.Path == "" {
		return fmt.Errorf("session %s has no recorded path; run `cly as save` from the project dir or `cly as search --refresh` to re-index", entry.ID)
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
	return execProvider(entry, provider, yolo)
}
