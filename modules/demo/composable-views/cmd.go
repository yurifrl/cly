package composableviews

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "composable-views",
		Short: "Composable views demo (timer and spinner)",
		Long:  "Demo showcasing composable views with timer and spinner components",
		RunE:  run,
	}

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	p := tea.NewProgram(initialModel(time.Minute))
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
