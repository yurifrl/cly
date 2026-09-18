package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/pkg/style"
)

// historyEntry is one completed gsync run, appended to
// ~/.local/state/cly/gsync/history.jsonl after every run (success or fail).
type historyEntry struct {
	Time     time.Time `json:"time"`
	Target   string    `json:"target"`
	Bucket   string    `json:"bucket"`
	Sources  int       `json:"sources"`
	Folders  int       `json:"folders"`
	Uploaded int       `json:"uploaded"`
	Skipped  int       `json:"skipped"`
	Errors   int       `json:"errors"`
	Seconds  float64   `json:"seconds"`
	Mode     string    `json:"mode"` // "interactive" | "headless"
}

// gsyncStateDir returns ~/.local/state/cly/gsync.
func gsyncStateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", "cly", "gsync"), nil
}

func gsyncHistoryPath() (string, error) {
	dir, err := gsyncStateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "history.jsonl"), nil
}

// appendHistory appends one run to the JSONL history, creating the directory
// as needed. History write failures never fail the sync itself.
func appendHistory(e historyEntry) {
	path, err := gsyncHistoryPath()
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	line, err := json.Marshal(e)
	if err != nil {
		return
	}
	_, _ = f.Write(append(line, '\n'))
}

// readHistory returns all recorded runs, oldest first.
func readHistory() ([]historyEntry, error) {
	path, err := gsyncHistoryPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []historyEntry
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var e historyEntry
		if err := json.Unmarshal([]byte(line), &e); err == nil {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	return out, nil
}

// RegisterGsyncStatus adds `cly gsync status`.
func RegisterGsyncStatus(gsyncCmd *cobra.Command) {
	c := &cobra.Command{
		Use:   "status",
		Short: "List data from previous gsync runs",
		Long: "Lists recent gsync runs (time, target, uploaded/skipped/errors, duration) from " +
			"~/.local/state/cly/gsync/history.jsonl, plus a per-target summary of the last run.",
		Args: cobra.NoArgs,
		RunE: runGsyncStatus,
	}
	c.Flags().IntP("limit", "n", 15, "How many recent runs to show")
	c.Flags().String("target", "", "Only show runs for this target")
	c.Flags().Bool("json", false, "Emit JSON instead of a table")
	gsyncCmd.AddCommand(c)
}

func runGsyncStatus(cmd *cobra.Command, args []string) error {
	limit, _ := cmd.Flags().GetInt("limit")
	target, _ := cmd.Flags().GetString("target")
	asJSON, _ := cmd.Flags().GetBool("json")

	entries, err := readHistory()
	if err != nil {
		return err
	}
	if target != "" {
		var filtered []historyEntry
		for _, e := range entries {
			if e.Target == target {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}
	if asJSON {
		out, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(out))
		return nil
	}
	if len(entries) == 0 {
		fmt.Println(style.YellowStyle.Render("no gsync runs recorded yet"))
		return nil
	}

	fmt.Println(style.TitleStyle.Render(fmt.Sprintf("gsync runs (%d recorded)", len(entries))))
	fmt.Printf("  %-17s %-10s %6s %6s %5s %7s  %-11s %s\n", "TIME", "TARGET", "UP", "SKIP", "ERR", "SECONDS", "MODE", "BUCKET")
	tail := entries
	if len(tail) > limit {
		tail = tail[len(tail)-limit:]
	}
	for _, e := range tail {
		marker := style.GreenStyle.Render("✓")
		if e.Errors > 0 {
			marker = style.RedStyle.Render("✗")
		}
		fmt.Printf(" %s %-17s %-10s %6d %6d %5d %7.0f  %-11s %s\n",
			marker,
			e.Time.Format("2006-01-02 15:04"),
			truncate(e.Target, 10),
			e.Uploaded, e.Skipped, e.Errors, e.Seconds, e.Mode, e.Bucket)
	}

	// Per-target last-run summary.
	fmt.Println(style.TitleStyle.Render("\nlast run per target"))
	latest := map[string]historyEntry{}
	for _, e := range entries {
		if prev, ok := latest[e.Target]; !ok || e.Time.After(prev.Time) {
			latest[e.Target] = e
		}
	}
	targets := make([]string, 0, len(latest))
	for t := range latest {
		targets = append(targets, t)
	}
	sort.Strings(targets)
	for _, t := range targets {
		e := latest[t]
		ago := time.Since(e.Time).Round(time.Second)
		icon := style.GreenStyle.Render("✓")
		if e.Errors > 0 {
			icon = style.RedStyle.Render("✗")
		}
		fmt.Printf("  %s %-12s %d↑ %d⏭ %d✗  %s ago  (bucket %s)\n",
			icon, e.Target, e.Uploaded, e.Skipped, e.Errors, ago, e.Bucket)
	}
	return nil
}
