package sequence

import (
	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "sequence",
		Short: "Run a series of commands in order",
		Long:  "Demonstrates how to run a series of commands in order using tea.Sequence()",
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
