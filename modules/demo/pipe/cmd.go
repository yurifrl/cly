package pipe

import (
	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "pipe",
		Short: "Piped input demo",
		Long:  "Demo showing how to handle piped input in Bubble Tea",
		RunE:  run,
	}

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	m := newModel("")
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
