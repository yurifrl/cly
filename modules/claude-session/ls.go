package claudesession

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
	sessions, err := Load(filePathFn())
	if err != nil {
		return err
	}

	if !all {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		sessions = filterByPath(sessions, cwd)
	}

	if len(sessions) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No saved sessions")
		return nil
	}

	entry, yolo, err := runPicker(sessions, "Sessions")
	if err != nil {
		return err
	}
	if entry == nil {
		return nil
	}

	return resumeEntry(entry, yolo)
}
