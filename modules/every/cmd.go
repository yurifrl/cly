package every

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// Register attaches `cly every` and its subcommands to parent.
func Register(parent *cobra.Command) {
	root := &cobra.Command{
		Use:   "every <interval> [-n NAME] -- <command...>",
		Short: "Run a command at a fixed interval with retry/backoff",
		Long: `Daemonless scheduled task runner.

Persists state and logs in ~/.local/state/cly/every/<NAME>.{state.json,log}.
Replaces the "while true; do cmd && sleep N || sleep M; done" pattern.`,
		Args: cobra.MinimumNArgs(2),
		RunE: runEvery,
	}
	root.Flags().StringP("name", "n", "", "task name (defaults to sha256(command)[:8])")
	root.Flags().Duration("backoff", time.Minute, "sleep between retries when failing")
	root.Flags().Int("max-fails", 0, "give up after N consecutive failures (0 = infinite)")
	root.Flags().Duration("jitter", 0, "± jitter applied to interval/backoff")
	root.Flags().Duration("initial-delay", 0, "delay before first run")
	root.Flags().Bool("notify", false, "fire desktop notifications on transitions")

	root.AddCommand(statusCmd())
	root.AddCommand(logsCmd())
	root.AddCommand(pruneCmd())
	parent.AddCommand(root)
}

func runEvery(cmd *cobra.Command, args []string) error {
	intervalStr := args[0]
	command := args[1:]
	if len(command) == 0 {
		return errors.New("missing command (use -- <command...>)")
	}

	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		return fmt.Errorf("interval: %w", err)
	}
	if interval <= 0 {
		return errors.New("interval must be > 0")
	}

	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		name = autoName(command)
	}
	if err := ValidateName(name); err != nil {
		return err
	}

	backoff, _ := cmd.Flags().GetDuration("backoff")
	maxFails, _ := cmd.Flags().GetInt("max-fails")
	jitter, _ := cmd.Flags().GetDuration("jitter")
	initialDelay, _ := cmd.Flags().GetDuration("initial-delay")
	notifyOn, _ := cmd.Flags().GetBool("notify")

	dir, err := DefaultDir()
	if err != nil {
		return err
	}

	r := NewRunner(dir)
	cfg := RunConfig{
		Name:         name,
		Command:      command,
		Interval:     interval,
		Backoff:      backoff,
		Jitter:       jitter,
		InitialDelay: initialDelay,
		MaxFails:     maxFails,
		Notify:       notifyOn,
	}
	return r.Run(cmd.Context(), cfg)
}

func autoName(command []string) string {
	sum := sha256.Sum256([]byte(strings.Join(command, " ")))
	return hex.EncodeToString(sum[:])[:8]
}

// ---------------------------------------------------------------------------
// status

func statusCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "status [NAME]",
		Short: "Show task status",
		RunE:  runStatus,
	}
	c.Flags().Bool("watch", false, "redraw every 1s")
	c.Flags().Bool("json", false, "emit JSON")
	return c
}

func runStatus(cmd *cobra.Command, args []string) error {
	dir, err := DefaultDir()
	if err != nil {
		return err
	}
	asJSON, _ := cmd.Flags().GetBool("json")
	watch, _ := cmd.Flags().GetBool("watch")

	render := func() error {
		if len(args) == 1 {
			return renderOne(cmd, dir, args[0], asJSON)
		}
		return renderAll(cmd, dir, asJSON)
	}

	if !watch {
		return render()
	}
	for {
		fmt.Fprint(cmd.OutOrStdout(), "\033[H\033[2J")
		if err := render(); err != nil {
			return err
		}
		select {
		case <-cmd.Context().Done():
			return nil
		case <-time.After(time.Second):
		}
	}
}

func renderAll(cmd *cobra.Command, dir string, asJSON bool) error {
	states, mods, _, err := LoadAll(dir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	now := time.Now()
	if asJSON {
		out := []*State{}
		out = append(out, states...)
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}
	rows := make([]StatusRow, 0, len(states))
	orphans := 0
	for _, s := range states {
		life := Lifecycle(s, mods[s.Name], now)
		rows = append(rows, RowFromState(s, life))
		if life == LifecycleOrphan {
			orphans++
		}
	}
	fmt.Fprint(cmd.OutOrStdout(), FormatStatusTable(rows, now))
	if orphans > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "\n💡 %d orphan task(s). Run `cly every prune --apply` to remove.\n", orphans)
	}
	return nil
}

func renderOne(cmd *cobra.Command, dir, name string, asJSON bool) error {
	st, err := ReadState(StatePath(dir, name))
	if err != nil {
		return err
	}
	if asJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(st)
	}
	info, _ := os.Stat(StatePath(dir, name))
	mod := time.Time{}
	if info != nil {
		mod = info.ModTime()
	}
	now := time.Now()
	row := RowFromState(st, Lifecycle(st, mod, now))
	fmt.Fprint(cmd.OutOrStdout(), FormatStatusTable([]StatusRow{row}, now))
	fmt.Fprintln(cmd.OutOrStdout(), "")
	fmt.Fprintf(cmd.OutOrStdout(), "command:    %s\n", st.Command)
	fmt.Fprintf(cmd.OutOrStdout(), "interval:   %s\n", time.Duration(st.IntervalSec)*time.Second)
	fmt.Fprintf(cmd.OutOrStdout(), "backoff:    %s\n", time.Duration(st.BackoffSec)*time.Second)
	fmt.Fprintf(cmd.OutOrStdout(), "started:    %s\n", FormatTS(st.StartedAt))
	fmt.Fprintf(cmd.OutOrStdout(), "last run:   %s\n", FormatTS(st.LastRunAt))
	fmt.Fprintf(cmd.OutOrStdout(), "next run:   %s\n", FormatTS(st.NextRunAt))
	fmt.Fprintf(cmd.OutOrStdout(), "consecutive_fails: %d\n\n", st.ConsecutiveFails)

	events, _ := ReadLog(LogPath(dir, name))
	for _, e := range LastN(events, 20) {
		fmt.Fprintln(cmd.OutOrStdout(), FormatLogEvent(e))
	}
	return nil
}

// ---------------------------------------------------------------------------
// logs

func logsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "logs <NAME>",
		Short: "Show task logs",
		Args:  cobra.ExactArgs(1),
		RunE:  runLogs,
	}
	c.Flags().BoolP("follow", "f", false, "tail the log file")
	c.Flags().Duration("since", 0, "filter events newer than DUR")
	c.Flags().Int("lines", 100, "show last N events")
	return c
}

func runLogs(cmd *cobra.Command, args []string) error {
	name := args[0]
	if err := ValidateName(name); err != nil {
		return err
	}
	dir, err := DefaultDir()
	if err != nil {
		return err
	}
	follow, _ := cmd.Flags().GetBool("follow")
	since, _ := cmd.Flags().GetDuration("since")
	lines, _ := cmd.Flags().GetInt("lines")

	path := LogPath(dir, name)
	_ = MaybeTrimLog(path, time.Now(), false)

	events, err := ReadLog(path)
	if err != nil {
		return err
	}
	if since > 0 {
		events = FilterSince(events, time.Now().Add(-since))
	}
	for _, e := range LastN(events, lines) {
		fmt.Fprintln(cmd.OutOrStdout(), FormatLogEvent(e))
	}
	if !follow {
		return nil
	}

	seen := len(events)
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(200 * time.Millisecond):
		}
		events, err := ReadLog(path)
		if err != nil {
			return err
		}
		for ; seen < len(events); seen++ {
			fmt.Fprintln(cmd.OutOrStdout(), FormatLogEvent(events[seen]))
		}
	}
}

// ---------------------------------------------------------------------------
// prune

func pruneCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "prune",
		Short: "Remove orphan task state files",
		RunE:  runPrune,
	}
	c.Flags().Bool("apply", false, "actually delete (default: dry-run)")
	c.Flags().Duration("max-age", OrphanThreshold, "orphan threshold")
	c.Flags().Bool("include-stopped", false, "also remove stopped tasks")
	return c
}

func runPrune(cmd *cobra.Command, args []string) error {
	dir, err := DefaultDir()
	if err != nil {
		return err
	}
	apply, _ := cmd.Flags().GetBool("apply")
	maxAge, _ := cmd.Flags().GetDuration("max-age")
	includeStopped, _ := cmd.Flags().GetBool("include-stopped")
	res, err := Prune(dir, time.Now(), PruneOptions{
		Apply:          apply,
		MaxAge:         maxAge,
		IncludeStopped: includeStopped,
	})
	if err != nil {
		return err
	}
	fmt.Fprint(cmd.OutOrStdout(), FormatPrune(dir, res))
	return nil
}
