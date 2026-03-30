package uuid

import (
	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "uuid",
		Short: "Generate UUIDs",
		Long:  "Interactive UUID generator with multiple format options",
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
