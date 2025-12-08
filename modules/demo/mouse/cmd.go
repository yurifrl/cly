package mouse

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "mouse",
		Short: "Mouse tracking demo",
		Long:  "Interactive demonstration of mouse tracking and click handling",
		RunE:  run,
	}

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	p := tea.NewProgram(initialModel(), tea.WithMouseAllMotion())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
