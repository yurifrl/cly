package omp

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	ompsummary "github.com/yurifrl/cly/modules/omp/summary"
)

// Register attaches the `extensions` command to parent (the `omp y` group).
func Register(parent *cobra.Command) {
	ext := &cobra.Command{
		Use:   "extensions",
		Short: "Manage omp extensions shipped by cly",
	}
	ext.AddCommand(installCmd())

	parent.AddCommand(ext)
}

// SummaryCmd keeps the active surface's summary card fresh: the engine resolves
// the focused surface, enqueues AI updates, and pushes the rendered card into
// the workspace description slot that the omp-summary cmux sidebar reads.
// Attached directly under `cly omp` (not the `y` group) so it runs as
// `cly omp summary`. Daemon by default; --once runs a single pass and exits.
func SummaryCmd() *cobra.Command {
	var once bool
	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Keep the active surface's AI summary card fresh (cmux sidebar feed)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := ompsummary.LoadConfig()
			sum := ompsummary.NewSummarizer(cfg)
			eng := ompsummary.NewEngine(cfg, sum)

			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()
			if once {
				eng.Tick(ctx) // shows the cached card immediately, enqueues staleness
				go sum.Start(ctx)
				for !sum.Idle() {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(100 * time.Millisecond):
					}
				}
				cancel()
				return nil
			}
			go sum.Start(ctx)
			eng.Run(ctx) // blocks until the context ends
			return nil
		},
	}
	cmd.Flags().BoolVar(&once, "once", false, "one discovery + summary pass, then exit")
	return cmd
}
