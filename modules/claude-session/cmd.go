package claudesession

import (
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/pkg/session"
)

func execClaude(entry *Entry) error {
	return session.ExecClaude([]string{"-r", entry.ID})
}

func Register(parent *cobra.Command) {
	var flagID, flagDesc, flagName string
	var flagSave bool

	cmd := &cobra.Command{
		Use:     "claude-sessions",
		Aliases: []string{"cs"},
		Short:   "Manage Claude sessions",
		Args:    cobra.MaximumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			sessions, err := Load(FilePath())
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
			name := flagName
			if name == "" && len(args) > 0 {
				name = args[0]
			}
			if name == "" {
				return cmd.Help()
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}

			filePath := FilePath()
			sessions, err := Load(filePath)
			if err != nil {
				return err
			}

			entry := FindByName(sessions, name)
			freshSession := entry == nil && flagID == ""

			if entry == nil {
				id := flagID
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
				if flagID != "" {
					entry.ID = flagID
				}
				if flagDesc != "" {
					entry.Description = flagDesc
				}
				if flagSave {
					entry.Path = cwd
				}
			}

			sessions[name] = *entry
			if err := Save(filePath, sessions); err != nil {
				return err
			}

			if flagSave {
				fmt.Printf("Saved session %q (id=%s)\n", name, entry.ID)
				return nil
			}

			if err := os.Chdir(entry.Path); err != nil {
				return fmt.Errorf("chdir %s: %w", entry.Path, err)
			}

			if freshSession {
				return session.ExecClaude([]string{"--session-id", entry.ID})
			}
			return execClaude(entry)
		},
	}

	cmd.Flags().StringVarP(&flagName, "name", "n", "", "Session name")
	cmd.RegisterFlagCompletionFunc("name", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		sessions, err := Load(FilePath())
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		names := make([]string, 0, len(sessions))
		for _, e := range sessions {
			names = append(names, e.Name)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().StringVarP(&flagID, "id", "i", "", "Session ID")
	cmd.Flags().StringVarP(&flagDesc, "description", "d", "", "Session description")
	cmd.Flags().BoolVar(&flagSave, "save", false, "Save without opening")

	cmd.AddCommand(lsCmd())
	cmd.AddCommand(rmCmd())

	parent.AddCommand(cmd)
}
