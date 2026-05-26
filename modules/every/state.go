package every

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"syscall"
	"time"
)

// Status values for the persisted state file.
const (
	StatusHealthy = "healthy"
	StatusFailing = "failing"
	StatusGaveUp  = "gave_up"
)

// Lifecycle values computed from a state file at read time.
const (
	LifecycleActive  = "active"
	LifecycleStopped = "stopped"
	LifecycleOrphan  = "orphan"
)

// OrphanThreshold is the age at which a state file with a dead pid is
// considered orphaned and eligible for sweep/prune.
const OrphanThreshold = 7 * 24 * time.Hour

// Totals tracks success/fail counters.
type Totals struct {
	Runs    int `json:"runs"`
	Success int `json:"success"`
	Fail    int `json:"fail"`
}

// State is the on-disk record for a scheduled task.
type State struct {
	Name             string    `json:"name"`
	Command          string    `json:"command"`
	IntervalSec      int       `json:"interval_sec"`
	BackoffSec       int       `json:"backoff_sec"`
	MaxFails         int       `json:"max_fails"`
	JitterSec        int       `json:"jitter_sec"`
	PID              int       `json:"pid"`
	StartedAt        time.Time `json:"started_at"`
	LastRunAt        time.Time `json:"last_run_at"`
	NextRunAt        time.Time `json:"next_run_at"`
	LastExit         int       `json:"last_exit"`
	LastDurationMs   int64     `json:"last_duration_ms"`
	ConsecutiveFails int       `json:"consecutive_fails"`
	Totals24h        Totals    `json:"totals_24h"`
	TotalsLifetime   Totals    `json:"totals_lifetime"`
	Status           string    `json:"status"`
	SnoozeUntil      time.Time `json:"snooze_until,omitempty"`
}

var nameRE = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// ValidateName returns nil if name is non-empty, <=64 chars and matches the
// allowed character set.
func ValidateName(name string) error {
	if name == "" {
		return errors.New("name is required")
	}
	if len(name) > 64 {
		return fmt.Errorf("name too long (max 64): %s", name)
	}
	if !nameRE.MatchString(name) {
		return fmt.Errorf("invalid name %q (allowed: A-Z a-z 0-9 . _ -)", name)
	}
	return nil
}

// DefaultDir returns the default state directory (~/.local/state/cly/every).
func DefaultDir() (string, error) {
	if d := os.Getenv("CLY_EVERY_DIR"); d != "" {
		return d, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", "cly", "every"), nil
}

// EnsureDir creates the state directory if it does not exist.
func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0o755)
}

// StatePath returns the path to <name>.state.json under dir.
func StatePath(dir, name string) string { return filepath.Join(dir, name+".state.json") }

// LogPath returns the path to <name>.log under dir.
func LogPath(dir, name string) string { return filepath.Join(dir, name+".log") }

// ReadState reads and parses a state file. Missing files surface as
// fs.ErrNotExist; corrupted files surface as a json error.
func ReadState(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &s, nil
}

// WriteState atomically writes the state to <path> using a tmp+rename dance.
func WriteState(path string, s *State) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

// PIDAlive returns true if the OS reports the pid as a live process owned by
// the current user (kill -0 semantics). On macOS this works for any pid you
// own; for foreign pids it returns false.
func PIDAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if err := p.Signal(syscall.Signal(0)); err != nil {
		return false
	}
	return true
}

// Lifecycle classifies a state at read time. now is injected for testability.
// fileModTime is used as a fallback when LastRunAt is zero.
func Lifecycle(s *State, fileModTime time.Time, now time.Time) string {
	pidLive := PIDAlive(s.PID)
	last := s.LastRunAt
	if last.IsZero() {
		last = fileModTime
	}
	age := now.Sub(last)
	if pidLive {
		win := time.Duration(s.IntervalSec)*time.Second*2 + time.Minute
		if win <= 0 {
			win = time.Minute
		}
		if age <= win {
			return LifecycleActive
		}
		// pid alive but stale heartbeat: still consider active so we don't
		// claim orphan against a live process.
		return LifecycleActive
	}
	if age > OrphanThreshold {
		return LifecycleOrphan
	}
	return LifecycleStopped
}

// LoadAll scans dir and returns parsed states keyed by name. Corrupt files
// are skipped (with their error noted in the returned errs map).
func LoadAll(dir string) (states []*State, mods map[string]time.Time, errs map[string]error, err error) {
	entries, e := os.ReadDir(dir)
	if e != nil {
		if errors.Is(e, fs.ErrNotExist) {
			return nil, map[string]time.Time{}, map[string]error{}, nil
		}
		return nil, nil, nil, e
	}
	mods = map[string]time.Time{}
	errs = map[string]error{}
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		name := ent.Name()
		if filepath.Ext(name) != ".json" || !hasSuffix(name, ".state.json") {
			continue
		}
		path := filepath.Join(dir, name)
		st, perr := ReadState(path)
		if perr != nil {
			errs[name] = perr
			continue
		}
		info, ierr := ent.Info()
		if ierr == nil {
			mods[st.Name] = info.ModTime()
		}
		states = append(states, st)
	}
	sort.Slice(states, func(i, j int) bool { return states[i].Name < states[j].Name })
	return states, mods, errs, nil
}

func hasSuffix(s, suf string) bool {
	if len(s) < len(suf) {
		return false
	}
	return s[len(s)-len(suf):] == suf
}
