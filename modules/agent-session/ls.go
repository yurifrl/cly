package agentsession

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func lsCmd() *cobra.Command {
	var flagAll bool
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List sessions (current dir by default)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLS(cmd, flagAll)
		},
	}
	cmd.Flags().BoolVarP(&flagAll, "all", "a", false, "Show all sessions")
	return cmd
}

func runLS(cmd *cobra.Command, all bool) error {
	provider, err := providerFromCmd(cmd)
	if err != nil {
		return err
	}

	sessions, err := Load(filePathFn())
	if err != nil {
		return err
	}
	sessions = filterByProvider(sessions, provider.Name)

	if !all {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		sessions = filterByPath(sessions, cwd)
	}

	if len(sessions) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "No saved %s sessions\n", provider.Name)
		return nil
	}

	entry, yolo, err := runPicker(sessions, providerSupportsYolo(provider))
	if err != nil {
		return err
	}
	if entry == nil {
		return nil
	}

	return resumeEntry(entry, provider, yolo)
}
