package omp

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	"github.com/yurifrl/cly/pkg/cmux"
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

			// Ship the sidebar with the binary: install/refresh it before the
			// loop so a user machine needs only the binary, not this repo.
			// Non-fatal: cards still push without a fresh sidebar file.
			if err := ompsummary.EnsureSidebar(); err != nil {
				cmd.PrintErrf("sidebar install: %v\n", err)
			}

			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			// Make the product visible: point the right sidebar at the card
			// feed and say one line, so a foreground run never looks dead.
			if err := cmux.RightSidebarSet(ctx, "omp-cards"); err != nil {
				cmd.PrintErrf("omp summary: right sidebar: %v\n", err)
			}
			if once {
				t := eng.Tick(ctx) // shows the cached card immediately, enqueues staleness
				go sum.Start(ctx)
				for !sum.Idle() {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(100 * time.Millisecond):
					}
				}
				cancel()
				if t == nil {
					cmd.PrintErrln("omp summary: nothing to show (no live omp session near the focused surface)")
				} else {
					cmd.Printf("omp summary: card pushed for %q (workspace %s)\n", t.Title, t.Workspace)
				}
				return nil
			}
			cmd.PrintErrln("omp summary: watching — card renders in the right sidebar (ctrl-c to stop)")
			go sum.Start(ctx)
			eng.Run(ctx) // blocks until the context ends
			return nil
		},
	}
	cmd.Flags().BoolVar(&once, "once", false, "one discovery + summary pass, then exit")
	return cmd
}
