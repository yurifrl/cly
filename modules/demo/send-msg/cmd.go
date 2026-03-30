package sendmsg

import (
	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "send-msg",
		Short: "Send messages to a running program",
		Long:  "Demonstrates sending messages to a Bubble Tea program from outside using Program.Send()",
		RunE:  run,
	}

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	p := tea.NewProgram(initialModel())

	// Simulate activity
	go sendMessages(p)

	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
