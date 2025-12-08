package realtime

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "realtime",
		Short: "Send activity to Bubble Tea in real-time",
		Long:  "Demonstrates sending activity to Bubble Tea in real-time through a channel",
		RunE:  run,
	}

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	p := tea.NewProgram(initialModel())

	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
