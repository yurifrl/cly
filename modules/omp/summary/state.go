package ompsummary

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Summary is the AI summary for one omp session, persisted in the session's
// project dir at .omp/summary.json.
type Summary struct {
	Text        string  `json:"text"`
	Status      string  `json:"status"` // pending | running | ok | error
	Error       string  `json:"error,omitempty"`
	Model       string  `json:"model,omitempty"`  // summarizer model, not session model
	Provider    string  `json:"provider,omitempty"`
	Fingerprint string  `json:"fingerprint"` // transcript (size:mtime) at summary time
	UpdatedAt   float64 `json:"updated_at_unix"`
	Turns       int     `json:"turns"`
}

// State is the on-disk document.
type State struct {
	UpdatedAt float64             `json:"updated_at_unix"`
	Summaries map[string]*Summary `json:"summaries"` // keyed by session_id
}

// StatePath returns <cwd>/.omp/summary.json.
func StatePath(cwd string) string {
	return filepath.Join(cwd, ".omp", "summary.json")
}

// LoadState reads the state file for cwd. Missing file → empty state.
// Corrupt file → empty state (self-heals on next Save).
func LoadState(cwd string) *State {
	st := &State{Summaries: map[string]*Summary{}}
	b, err := os.ReadFile(StatePath(cwd))
	if err != nil {
		return st
	}
	var d State
	if err := json.Unmarshal(b, &d); err != nil || d.Summaries == nil {
		return st
	}
	return &d
}

// SaveState atomically writes the state file for cwd: write temp, fsync,
// rename over the target, 0644.
func SaveState(cwd string, st *State) error {
	dir := filepath.Join(cwd, ".omp")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir .omp: %w", err)
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".summary-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op after successful rename
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpName, StatePath(cwd))
}

// Update folds a single session summary into st and stamps UpdatedAt.
func (st *State) Update(sessionID string, s *Summary) {
	if st.Summaries == nil {
		st.Summaries = map[string]*Summary{}
	}
	s.UpdatedAt = nowUnix()
	st.Summaries[sessionID] = s
	st.UpdatedAt = s.UpdatedAt
}

// Prune drops sessions older than maxAge seconds to bound file growth
// (hook stores accumulate hundreds of stale rows on reused tabs).
func (st *State) Prune(now, maxAge float64) {
	for id, s := range st.Summaries {
		if now-s.UpdatedAt > maxAge {
			delete(st.Summaries, id)
		}
	}
}
