package omp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
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
// `cly omp summary`.
//
// Default UX: open the visualization (right sidebar, omp-cards), then return —
// the watcher daemonizes in the background. --foreground keeps the loop in the
// terminal, --once runs a single pass, --stop kills the background daemon.
func SummaryCmd() *cobra.Command {
	var once, foreground, stop bool
	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Open the live omp summary sidebar and keep it fresh in the background",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Ship the sidebar with the binary: install/refresh it before the
			// loop so a user machine needs only the binary, not this repo.
			// Non-fatal: cards still push without a fresh sidebar file.
			if err := ompsummary.EnsureSidebar(); err != nil {
				cmd.PrintErrf("sidebar install: %v\n", err)
			}

			if stop {
				return stopDaemon(cmd)
			}

			// Open the visualization: switch the right sidebar to the card feed
			// and make it visible. Targets the frontmost window explicitly, so
			// this works from any shell. Non-fatal: the card still pushes.
			if err := cmux.RightSidebarSet(ctx, "omp-cards"); err != nil {
				cmd.PrintErrf("omp summary: right sidebar: %v\n", err)
			} else if err := cmux.RightSidebarShow(ctx); err != nil {
				cmd.PrintErrf("omp summary: right sidebar: %v\n", err)
			}

			if once {
				return runOnce(cmd, ctx)
			}
			if foreground {
				return runForeground(cmd, ctx)
			}

			// Daemonize: re-exec self in the background and return the
			// terminal immediately. The child re-enters with --foreground.
			return startDaemon(cmd)
		},
	}
	cmd.Flags().BoolVar(&once, "once", false, "one discovery + summary pass, then exit")
	cmd.Flags().BoolVar(&foreground, "foreground", false, "run the watcher loop in this terminal instead of the background")
	cmd.Flags().BoolVar(&stop, "stop", false, "stop the background daemon")
	return cmd
}

func runOnce(cmd *cobra.Command, ctx context.Context) error {
	cfg := ompsummary.LoadConfig()
	sum := ompsummary.NewSummarizer(cfg)
	eng := ompsummary.NewEngine(cfg, sum)
	t := eng.Tick(ctx) // shows the cached card immediately, enqueues staleness
	go sum.Start(ctx)
	for !sum.Idle() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
	if t == nil {
		cmd.PrintErrln("omp summary: nothing to show (no live omp session near the focused surface)")
	} else {
		cmd.Printf("omp summary: card pushed for %q (workspace %s)\n", t.Title, t.Workspace)
	}
	return nil
}

func runForeground(cmd *cobra.Command, ctx context.Context) error {
	cfg := ompsummary.LoadConfig()
	sum := ompsummary.NewSummarizer(cfg)
	eng := ompsummary.NewEngine(cfg, sum)
	cmd.PrintErrln("omp summary: watching — card renders in the right sidebar (ctrl-c to stop)")
	go sum.Start(ctx)
	eng.Run(ctx) // blocks until the context ends
	return nil
}

// daemonPaths: one pidfile/logfile pair for the background watcher, wherever
// it was started from.
func daemonPaths() (pidFile, logFile string) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	base := filepath.Join(home, ".omp", "agent")
	return filepath.Join(base, "omp-summary-daemon.pid"),
		filepath.Join(base, "omp-summary-daemon.log")
}

func startDaemon(cmd *cobra.Command) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate binary: %w", err)
	}
	pidFile, logFile := daemonPaths()
	if pid, alive := daemonAlive(pidFile); alive {
		cmd.Printf("omp summary: already running in background (pid %d) — sidebar is open; stop with `cly omp summary --stop`\n", pid)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(logFile), 0o755); err != nil {
		return err
	}
	log, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer log.Close()
	c := exec.Command(exe, "omp", "summary", "--foreground")
	c.Stdin = nil
	c.Stdout = log
	c.Stderr = log
	c.SysProcAttr = &syscall.SysProcAttr{Setsid: true} // survive terminal close
	if err := c.Start(); err != nil {
		return fmt.Errorf("start daemon: %w", err)
	}
	pid := c.Process.Pid
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0o644); err != nil {
		return err
	}
	_ = c.Process.Release()
	// Give the child a beat to fail on startup (bad binary, port clash) so
	// the user sees the error here instead of hunting the log.
	time.Sleep(300 * time.Millisecond)
	if _, alive := daemonAlive(pidFile); !alive {
		logBytes, _ := os.ReadFile(logFile)
		tail := logBytes
		if len(tail) > 400 {
			tail = tail[len(tail)-400:]
		}
		_ = os.Remove(pidFile)
		return fmt.Errorf("daemon exited at startup; log tail:\n%s", strings.TrimSpace(string(tail)))
	}
	cmd.Printf("omp summary: sidebar open, watching in background (pid %d)\n  logs: %s\n  stop: cly omp summary --stop\n", pid, logFile)
	return nil
}

// daemonAlive reports whether the pid in pidFile names a live `omp summary`
// process. A pid alone is not proof (reuse) — the command line is checked.
func daemonAlive(pidFile string) (int, bool) {
	raw, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || pid <= 0 {
		return 0, false
	}
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "command=").Output()
	if err != nil || !strings.Contains(string(out), "omp summary") {
		return 0, false
	}
	return pid, true
}

func stopDaemon(cmd *cobra.Command) error {
	pidFile, _ := daemonPaths()
	pid, alive := daemonAlive(pidFile)
	if !alive {
		_ = os.Remove(pidFile)
		cmd.PrintErrln("omp summary: not running")
		return nil
	}
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		return fmt.Errorf("stop pid %d: %w", pid, err)
	}
	_ = os.Remove(pidFile)
	cmd.Printf("omp summary: stopped (pid %d)\n", pid)
	return nil
}
