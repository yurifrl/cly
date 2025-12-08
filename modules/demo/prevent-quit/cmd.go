package preventquit

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func filter(teaModel tea.Model, msg tea.Msg) tea.Msg {
	if _, ok := msg.(tea.QuitMsg); !ok {
		return msg
	}

	m := teaModel.(model)
	if m.hasChanges {
		return nil
	}

	return msg
}

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "prevent-quit",
		Short: "Prevent quit with unsaved changes demo",
		Long:  "Interactive demonstration of preventing application quit when there are unsaved changes",
		RunE:  run,
	}

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	p := tea.NewProgram(initialModel(), tea.WithFilter(filter))
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
