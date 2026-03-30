package setwindowtitle

import (
	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "set-window-title",
		Short: "Set a window title",
		Long:  "Demonstrates how to set a window title",
		RunE:  run,
	}

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	if _, err := tea.NewProgram(initialModel()).Run(); err != nil {
		return err
	}
	return nil
}
