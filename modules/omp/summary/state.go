package ompsummary

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// State is a project's summary cache: one JSON file per project under the
// global ~/.omp/agent/summary/ dir. version 3 entries carry everything the
// sidebar card needs.
type State struct {
	Version   int               `json:"version"` // stateVersion
	UpdatedAt float64           `json:"updated_at_unix"`
	Sessions  map[string]*Entry `json:"sessions"`
}

// stateVersion is the on-disk format version. v3 adds response/action/timeline
// to each entry; loading a v2 file is a no-op migration (missing fields decode
// as empty).
const stateVersion = 3

// Turn is one timeline message on a session card.
type Turn struct {
	Who  string `json:"who"` // "user" | "ai"
	Text string `json:"text"`
}

// Entry is one omp session's card: what the user asked, what the assistant is
// doing, the last command, and which AI produced it.
type Entry struct {
	Status string `json:"status"` // pending | running | ok | error
	Error  string `json:"error,omitempty"`

	SurfaceID string `json:"surface_id,omitempty"`
	Title     string `json:"title,omitempty"`     // surface title
	Workspace string `json:"workspace,omitempty"` // workspace title

	Goal        string  `json:"goal,omitempty"`     // one-line: what this session is trying to accomplish
	Response    string  `json:"response,omitempty"` // last assistant reply
	Action      string  `json:"action,omitempty"`   // what the user must answer; empty = nothing
	UserAsk     string  `json:"user_ask,omitempty"` // most recent substantive user message
	UserAskAt   float64 `json:"user_ask_at,omitempty"`
	LastCmd     string  `json:"last_command,omitempty"` // last tool: intent one-liner
	Exit        int     `json:"exit"`                   // exit code of the last command; -1 unknown
	Timeline    []Turn  `json:"timeline,omitempty"`     // oldest→newest, max 5
	Compactions int     `json:"compactions,omitempty"`  // OMP context folds in the transcript
	CompactedAt float64 `json:"compacted_at,omitempty"` // latest fold timestamp
	FoldNote    string  `json:"fold_note,omitempty"`    // one-line distill of the latest fold

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

// summaryDir is the global home for per-project summary state; overridable
// for tests. Empty on home-dir failure — reads then just miss.
var summaryDir = func() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".omp", "agent", "summary")
}

// StatePath is the global per-project cache location: summaryDir/<slug(cwd)>.json
func StatePath(cwd string) string {
	return filepath.Join(summaryDir(), slug(cwd)+".json")
}

// slug turns a cwd into a stable, filesystem-safe file stem: percent-encoded
// full path, truncated to stay well under macOS's 255-byte name cap, plus a
// 16-hex hash so distinct paths stay distinct across truncation.
func slug(cwd string) string {
	s := url.PathEscape(filepath.ToSlash(cwd))
	if len(s) > 180 {
		s = s[:180]
	}
	h := fnv.New64a()
	h.Write([]byte(cwd))
	return s + fmt.Sprintf("-%016x", h.Sum64())
}

// loadAt reads a state file at an explicit path. v1 files are migrated
// in-memory (Goal=old text, fingerprint/model preserved) so the card shows
// something on first load without an AI round-trip. Corrupt file → empty
// state (self-heals on next Save).
func loadAt(p string) *State {
	st := &State{Version: stateVersion, Sessions: map[string]*Entry{}}
	b, err := os.ReadFile(p)
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
	case probe.Version == stateVersion && probe.Sessions != nil:
		st.Sessions = probe.Sessions
	case probe.Version == stateVersion-1 && probe.Sessions != nil: // v2: no-op migration, missing fields decode empty
		st.Sessions = probe.Sessions
	case probe.Summaries != nil: // v1
		for id, v := range probe.Summaries {
			st.Sessions[id] = migrateV1(v)
		}
	}
	return st
}

// LoadState reads this project's state from the global summary dir.
func LoadState(cwd string) *State {
	return loadAt(StatePath(cwd))
}

// saveAt atomically writes a state file at an explicit path: write temp,
// fsync, rename over the target, 0644.
func saveAt(p string, st *State) error {
	if st.Sessions == nil {
		st.Sessions = map[string]*Entry{}
	}
	st.Version = stateVersion
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
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

// SaveState atomically writes this project's state into the global summary dir.
func SaveState(cwd string, st *State) error {
	return saveAt(StatePath(cwd), st)
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

// SweepAll prunes stale sessions from every state file in the global summary
// dir and removes files left empty, including unparseable ones (they load as
// empty). maxAge is the entry-TTL in seconds. Best-effort: a missing dir just
// means nothing was summarized yet.
func SweepAll(maxAge float64) {
	dir := summaryDir()
	if dir == "" {
		return
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	now := nowUnix()
	for _, de := range ents {
		if de.IsDir() || !strings.HasSuffix(de.Name(), ".json") || strings.HasSuffix(de.Name(), ".tmp") {
			continue
		}
		p := filepath.Join(dir, de.Name())
		st := loadAt(p)
		n := len(st.Sessions)
		st.Prune(now, maxAge)
		switch {
		case n == 0: // empty or corrupt: dead file, remove
			_ = os.Remove(p)
		case len(st.Sessions) == n: // nothing pruned: keep file and its mtime
		default:
			_ = saveAt(p, st)
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
