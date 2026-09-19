package omp

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	ompsummary "github.com/yurifrl/cly/modules/omp/summary"
)

// Register attaches the `extensions` command to parent.
func Register(parent *cobra.Command) {
	ext := &cobra.Command{
		Use:   "extensions",
		Short: "Manage omp extensions shipped by cly",
	}
	ext.AddCommand(installCmd())

	parent.AddCommand(ext)
	parent.AddCommand(summaryCmd())
}

// summaryCmd is the live sidebar TUI over omp session summaries.
func summaryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "summary",
		Short: "Live AI summaries of omp sessions in this workspace (sidebar TUI)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := ompsummary.LoadConfig()
			sum := ompsummary.NewSummarizer(cfg)
			eng := ompsummary.NewEngine(cfg, sum)

			p := tea.NewProgram(ompsummary.NewModel(eng, sum))
			wake := func() { p.Send(ompsummary.Wake{}) }
			eng.SetSend(wake)
			sum.SetNotify(wake)

			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()
			go eng.Run(ctx)
			go sum.Start(ctx)

			_, err := p.Run()
			return err
		},
	}
}
