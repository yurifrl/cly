package memwatch

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/pkg/cmux"
)

// newNotifyCmd builds the `cly memwatch notify` command group.
// Each subcommand exercises a single notification path so you can verify
// cmux notify / cmux sidebar / desktop / zellij routing end-to-end.
func newNotifyCmd() *cobra.Command {
	notify := &cobra.Command{
		Use:   "notify",
		Short: "Fire test notifications (exercises every alert path)",
		Long: `Fire test notifications for each memwatch alert path.

Examples:
  cly memwatch notify warn         # warn-level alert + orange sidebar badge
  cly memwatch notify critical     # critical alert + red sidebar badge
  cly memwatch notify proc         # per-process outlier alert (uses real top procs)
  cly memwatch notify status       # set the cmux sidebar badge only (no popup)
  cly memwatch notify desktop      # force-send a macOS desktop notification
  cly memwatch notify all          # fire every path in sequence`,
	}

	notify.AddCommand(
		&cobra.Command{
			Use:   "warn",
			Short: "Fire a warn-level system memory alert",
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg := loadConfig()
				procs, _ := TopProcesses(cmd.Context(), cfg.TopN)
				s := &Sample{FreePercent: cfg.ThresholdPercent, PressureLvl: "warn"}
				syncCmuxStatus(cmd.Context(), cfg, s, true)
				return sendAlert(cmd.Context(), cfg, s, procs, true)
			},
		},
		&cobra.Command{
			Use:   "critical",
			Short: "Fire a critical-level system memory alert",
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg := loadConfig()
				procs, _ := TopProcesses(cmd.Context(), cfg.TopN)
				s := &Sample{FreePercent: cfg.CriticalPercent, PressureLvl: "critical"}
				syncCmuxStatus(cmd.Context(), cfg, s, true)
				return sendAlert(cmd.Context(), cfg, s, procs, true)
			},
		},
		&cobra.Command{
			Use:   "proc",
			Short: "Fire a per-process outlier alert using the current top process",
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg := loadConfig()
				procs, err := TopProcesses(cmd.Context(), cfg.TopN)
				if err != nil {
					return err
				}
				if len(procs) == 0 {
					return fmt.Errorf("no processes found")
				}
				s := &Sample{FreePercent: -1, PressureLvl: "warn"}
				return sendProcAlert(cmd.Context(), cfg, s, procs[:1], procs)
			},
		},
		&cobra.Command{
			Use:   "status",
			Short: "Set the cmux sidebar badge only (no popup)",
			RunE: func(cmd *cobra.Command, args []string) error {
				if !cmux.Available() {
					fmt.Fprintln(os.Stderr, "memwatch: not inside a cmux session; skipping sidebar test")
					return nil
				}
				cfg := loadConfig()
				pct, _ := cmd.Flags().GetInt("percent")
				s := &Sample{FreePercent: pct}
				if pct <= cfg.CriticalPercent {
					s.PressureLvl = "critical"
				} else {
					s.PressureLvl = "warn"
				}
				syncCmuxStatus(cmd.Context(), cfg, s, true)
				fmt.Fprintf(os.Stderr, "memwatch: sidebar badge set to RAM %d%% — run `cly memwatch clear` to remove\n", pct)
				return nil
			},
		},
		&cobra.Command{
			Use:   "desktop",
			Short: "Force a macOS desktop notification (bypasses cmux even if available)",
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg := loadConfig()
				cfg.UseCmux = false // force desktop path
				cfg.UseDesktop = true
				s := &Sample{FreePercent: 15, PressureLvl: "warn"}
				return dispatch(cmd.Context(), cfg, s, "🧪 memwatch test", "Desktop notification path — if you can read this, beeep works.")
			},
		},
		&cobra.Command{
			Use:   "all",
			Short: "Fire every notification path in sequence (with short pauses)",
			RunE: func(cmd *cobra.Command, args []string) error {
				for _, sub := range []string{"warn", "critical", "proc", "desktop"} {
					c, _, err := cmd.Root().Find([]string{"memwatch", "notify", sub})
					if err != nil {
						return err
					}
					fmt.Fprintf(os.Stderr, "\n=== memwatch notify %s ===\n", sub)
					if err := c.RunE(c, nil); err != nil {
						fmt.Fprintf(os.Stderr, "memwatch: %s failed: %v\n", sub, err)
					}
				}
				return nil
			},
		},
	)

	// Flag on `status` subcommand.
	for _, c := range notify.Commands() {
		if c.Use == "status" {
			c.Flags().Int("percent", 15, "free memory percent to display on the badge")
		}
	}

	return notify
}
