package every

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// PruneCategory groups state files by lifecycle for prune output.
type PruneCategory struct {
	Active  []string
	Stopped []string
	Orphan  []string
}

// Classify scans dir and bucketises every <name>.state.json file into one of
// three lifecycle categories. now is injected for testability. maxAge
// overrides the orphan threshold when > 0.
func Classify(dir string, now time.Time, maxAge time.Duration) (PruneCategory, error) {
	var cat PruneCategory
	states, mods, _, err := LoadAll(dir)
	if err != nil {
		return cat, err
	}
	threshold := maxAge
	if threshold <= 0 {
		threshold = OrphanThreshold
	}
	for _, st := range states {
		mod := mods[st.Name]
		life := lifecycleWithThreshold(st, mod, now, threshold)
		switch life {
		case LifecycleActive:
			cat.Active = append(cat.Active, st.Name)
		case LifecycleStopped:
			cat.Stopped = append(cat.Stopped, st.Name)
		case LifecycleOrphan:
			cat.Orphan = append(cat.Orphan, st.Name)
		}
	}
	sort.Strings(cat.Active)
	sort.Strings(cat.Stopped)
	sort.Strings(cat.Orphan)
	return cat, nil
}

func lifecycleWithThreshold(s *State, modTime, now time.Time, threshold time.Duration) string {
	pidLive := PIDAlive(s.PID)
	last := s.LastRunAt
	if last.IsZero() {
		last = modTime
	}
	if pidLive {
		return LifecycleActive
	}
	if now.Sub(last) > threshold {
		return LifecycleOrphan
	}
	return LifecycleStopped
}

// RemoveTask deletes both the state file and the log file for a given name.
// Missing files are ignored.
func RemoveTask(dir, name string) error {
	var firstErr error
	for _, p := range []string{StatePath(dir, name), LogPath(dir, name)} {
		if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// SweepOrphans removes orphan state files. Returns the names removed.
func SweepOrphans(dir string, now time.Time) ([]string, error) {
	if err := EnsureDir(dir); err != nil {
		return nil, err
	}
	cat, err := Classify(dir, now, 0)
	if err != nil {
		return nil, err
	}
	var removed []string
	for _, n := range cat.Orphan {
		if err := RemoveTask(dir, n); err != nil {
			return removed, err
		}
		removed = append(removed, n)
	}
	return removed, nil
}

// PruneOptions controls the prune CLI command behaviour.
type PruneOptions struct {
	Apply           bool
	MaxAge          time.Duration
	IncludeStopped  bool
}

// PruneResult is what `cly every prune` emits to the caller.
type PruneResult struct {
	Categories PruneCategory
	Removed    []string
	DryRun     bool
}

// Prune is the engine behind `cly every prune`. It never deletes anything in
// dry-run mode; with Apply=true it removes orphans (always) plus stopped
// tasks when IncludeStopped is set.
func Prune(dir string, now time.Time, opts PruneOptions) (PruneResult, error) {
	cat, err := Classify(dir, now, opts.MaxAge)
	if err != nil {
		return PruneResult{}, err
	}
	res := PruneResult{Categories: cat, DryRun: !opts.Apply}
	candidates := append([]string{}, cat.Orphan...)
	if opts.IncludeStopped {
		candidates = append(candidates, cat.Stopped...)
	}
	if !opts.Apply {
		res.Removed = nil
		return res, nil
	}
	for _, n := range candidates {
		if err := RemoveTask(dir, n); err != nil {
			return res, fmt.Errorf("remove %s: %w", n, err)
		}
		res.Removed = append(res.Removed, n)
	}
	return res, nil
}

// FormatPrune renders a human PruneResult summary.
func FormatPrune(dir string, res PruneResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "🔍 %s\n", displayDir(dir))
	fmt.Fprintf(&b, "  active:  %d%s\n", len(res.Categories.Active), maybeList(res.Categories.Active))
	fmt.Fprintf(&b, "  stopped: %d%s\n", len(res.Categories.Stopped), maybeList(res.Categories.Stopped))
	fmt.Fprintf(&b, "  orphan:  %d%s\n", len(res.Categories.Orphan), maybeList(res.Categories.Orphan))
	if res.DryRun {
		n := len(res.Categories.Orphan)
		if n == 0 {
			fmt.Fprintf(&b, "Nothing to prune.\n")
		} else {
			fmt.Fprintf(&b, "[dry-run] Would remove %d orphan task(s). Run with --apply.\n", n)
		}
	} else {
		fmt.Fprintf(&b, "Removed %d task(s)%s.\n", len(res.Removed), maybeList(res.Removed))
	}
	return b.String()
}

func maybeList(names []string) string {
	if len(names) == 0 {
		return ""
	}
	return " (" + strings.Join(names, ", ") + ")"
}

func displayDir(dir string) string {
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(dir, home) {
		return "~" + strings.TrimPrefix(dir, home)
	}
	return filepath.Clean(dir)
}
