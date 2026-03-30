package vanish

import (
	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
)

type model bool

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tea.KeyPressMsg); ok {
		m = true
		return m, tea.Quit
	}
	return m, nil
}

func (m model) View() tea.View {
	if m {
		return tea.NewView("")
	}
	return tea.NewView("Press any key to quit.\n(When this program quits, it will vanish without a trace.)")
}

func run(cmd *cobra.Command, args []string) error {
	p := tea.NewProgram(model(false))
	_, err := p.Run()
	return err
}
