package focusblur

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "focus-blur",
		Short: "Focus and blur event demo",
		Long:  "Interactive demonstration of handling terminal focus and blur events",
		RunE:  run,
	}

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	p := tea.NewProgram(initialModel(), tea.WithReportFocus())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
