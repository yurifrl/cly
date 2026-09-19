package ompsummary

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// State is .omp/summary.json: the per-project cache of session cards.
// version 2 entries carry everything the sidebar card needs.
type State struct {
	Version   int               `json:"version"` // 2
	UpdatedAt float64           `json:"updated_at_unix"`
	Sessions  map[string]*Entry `json:"sessions"`
}

// Entry is one omp session's card: what the user asked, what the assistant is
// doing, the last command, and which AI produced it.
type Entry struct {
	Status    string `json:"status"` // pending | running | ok | error
	Error     string `json:"error,omitempty"`

	SurfaceID string `json:"surface_id,omitempty"`
	Title     string `json:"title,omitempty"`    // surface title
	Workspace string `json:"workspace,omitempty"` // workspace title

	Goal        string  `json:"goal,omitempty"`       // one-line: what this session is trying to accomplish
	AIStatus    string  `json:"ai_status,omitempty"`  // one-line: what the assistant is doing/asking now
	UserAsk     string  `json:"user_ask,omitempty"`   // most recent substantive user message
	UserAskAt   float64 `json:"user_ask_at,omitempty"`
	LastCmd     string  `json:"last_command,omitempty"` // last tool: intent one-liner
	Exit        int     `json:"exit"`                   // exit code of the last command; -1 unknown

	Model       string  `json:"model,omitempty"`
	Provider    string  `json:"provider,omitempty"`
	Fingerprint string  `json:"fingerprint,omitempty"`
	UpdatedAt   float64 `json:"updated_at_unix"`
	Turns       int     `json:"turns,omitempty"`
}

// v1 file shape (pre-card): per-session one-liner summaries.
type v1Summary struct {
	Text        string  `json:"text"`
	Status      string  `json:"status"`
	Error       string  `json:"error,omitempty"`
	Model       string  `json:"model,omitempty"`
	Provider    string  `json:"provider,omitempty"`
	Fingerprint string  `json:"fingerprint,omitempty"`
	UpdatedAt   float64 `json:"updated_at_unix"`
	Turns       int     `json:"turns,omitempty"`
}

// v1Summary carries exit=-1 so a migrated entry never claims a known exit.
func migrateV1(v *v1Summary) *Entry {
	return &Entry{
		Status:      v.Status,
		Error:       v.Error,
		Goal:        v.Text, // v1 text was exactly this one-liner
		Model:       v.Model,
		Provider:    v.Provider,
		Fingerprint: v.Fingerprint,
		UpdatedAt:   v.UpdatedAt,
		Turns:       v.Turns,
		Exit:        -1,
	}
}

// LoadState reads cwd/.omp/summary.json. v1 files are migrated in-memory
// (Goal=old text, fingerprint/model preserved) so the card shows something on
// first load without an AI round-trip. Corrupt file → empty state (self-heals
// on next Save).
func LoadState(cwd string) *State {
	st := &State{Version: 2, Sessions: map[string]*Entry{}}
	b, err := os.ReadFile(StatePath(cwd))
	if err != nil {
		return st
	}
	var probe struct {
		Version   int               `json:"version"`
		Sessions  map[string]*Entry `json:"sessions"`
		Summaries map[string]*v1Summary
	}
	if err := json.Unmarshal(b, &probe); err != nil {
		return st
	}
	switch {
	case probe.Version == 2 && probe.Sessions != nil:
		st.Sessions = probe.Sessions
	case probe.Summaries != nil: // v1
		for id, v := range probe.Summaries {
			st.Sessions[id] = migrateV1(v)
		}
	}
	return st
}

// StatePath is the per-project cache location.
func StatePath(cwd string) string {
	return filepath.Join(cwd, ".omp", "summary.json")
}

// SaveState atomically writes the state file for cwd: write temp, fsync,
// rename over the target, 0644.
func SaveState(cwd string, st *State) error {
	if st.Sessions == nil {
		st.Sessions = map[string]*Entry{}
	}
	st.Version = 2
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	p := StatePath(cwd)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	tmp := p + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.Write(b); err != nil {
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
	return os.Rename(tmp, p)
}

// Get returns the entry for a session id, or nil.
func (st *State) Get(sessionID string) *Entry {
	return st.Sessions[sessionID]
}

// Update folds a single session entry in and stamps UpdatedAt.
func (st *State) Update(sessionID string, e *Entry) {
	if st.Sessions == nil {
		st.Sessions = map[string]*Entry{}
	}
	e.UpdatedAt = nowUnix()
	st.Sessions[sessionID] = e
	st.UpdatedAt = e.UpdatedAt
}

// Prune drops sessions older than maxAge seconds to bound file growth
// (hook stores accumulate hundreds of stale rows on reused tabs).
func (st *State) Prune(now, maxAge float64) {
	for id, e := range st.Sessions {
		if now-e.UpdatedAt > maxAge {
			delete(st.Sessions, id)
		}
	}
}

// Status values for an entry's AI lifecycle.
const (
	StatusPending = "pending"
	StatusRunning = "running"
	StatusOK      = "ok"
	StatusError   = "error"
)

// NewerThan reports whether any entry is strictly newer than at.
func (st *State) NewerThan(at float64) bool {
	for _, e := range st.Sessions {
		if e.UpdatedAt > at {
			return true
		}
	}
	return false
}

// ExitUnknown marks entries whose last command has no exit code yet.
const ExitUnknown = -1

