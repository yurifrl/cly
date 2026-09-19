package ompsummary

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/yurifrl/cly/pkg/cmux"
)

// Target is one displayable row: a surface whose focused omp session is live
// enough to summarize.
type Target struct {
	SessionID   string
	SurfaceID   string
	PaneRef     string
	SurfaceRef  string
	WorkspaceID string
	Title       string // surface title, or workspace title fallback
	Workspace   string
	CWD         string
	Focused     bool
	UpdatedAt   float64 // transcript mtime when known, else hook updated_at
}

// sessionsDir is overridable for tests.
var sessionsDir = func() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".omp", "agent", "sessions")
}

// transcriptPath resolves a session id to its transcript file:
// ~/.omp/agent/sessions/*/*_<session_id>.jsonl, newest mtime wins (the same
// id can appear under several munged cwd dirs; newest is the live one).
func transcriptPath(sessionID string) string {
	dir := sessionsDir()
	if dir == "" || sessionID == "" {
		return ""
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "*", "*_"+sessionID+".jsonl"))
	if len(matches) == 0 {
		// Some layouts write the bare id as the filename.
		matches, _ = filepath.Glob(filepath.Join(dir, "*", sessionID+".jsonl"))
	}
	if len(matches) == 0 {
		return ""
	}
	best := matches[0]
	bestMod, err := os.Stat(best)
	if err != nil {
		return ""
	}
	for _, m := range matches[1:] {
		mm, err := os.Stat(m)
		if err != nil {
			continue
		}
		if mm.ModTime().After(bestMod.ModTime()) {
			best, bestMod = m, mm
		}
	}
	return best
}

// Fingerprint identifies a transcript revision: size:mtimeNano.
func Fingerprint(path string) string {
	if path == "" {
		return ""
	}
	fi, err := os.Stat(path)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%d:%d", fi.Size(), fi.ModTime().UnixNano())
}

// Targets merges the cmux tree with omp hook records: one Target per surface
// that has a live omp session, focused surfaces first, then most recent.
func Targets(ctx context.Context, now float64, maxAge float64) []Target {
	tree, err := cmux.RunTree(ctx)
	if err != nil {
		return nil
	}
	rows, err := cmux.SessionsList(ctx, "--agent", "omp", "--all")
	if err != nil {
		return nil
	}
	return merge(tree, rows, now, maxAge)
}

// merge is Targets' pure core, unit-testable without the CLI.
func merge(tree *cmux.Tree, rows []cmux.SessionRow, now, maxAge float64) []Target {
	// Best session per surface: active_for_surface wins, else most recent.
	best := map[string]*cmux.SessionRow{}
	pick := func(cur *cmux.SessionRow, r cmux.SessionRow) *cmux.SessionRow {
		if cur == nil {
			return &r
		}
		if r.ActiveForSurf && !cur.ActiveForSurf {
			return &r
		}
		if r.ActiveForSurf == cur.ActiveForSurf && r.UpdatedAtUnix > cur.UpdatedAtUnix {
			return &r
		}
		return cur
	}
	for _, r := range rows {
		if r.Agent != "omp" || r.SurfaceID == nil || *r.SurfaceID == "" {
			continue
		}
		if r.RuntimeStatus == "gone" || r.AgentLifecycle == "done" {
			continue
		}
		if maxAge > 0 && now-r.UpdatedAtUnix > maxAge {
			continue
		}
		best[*r.SurfaceID] = pick(best[*r.SurfaceID], r)
	}

	var out []Target
	for _, w := range tree.Windows {
		for _, ws := range w.Workspaces {
			for _, p := range ws.Panes {
				for _, s := range p.Surfaces {
					r, ok := best[s.ID]
					if !ok {
						continue
					}
					t := Target{
						SessionID:   r.SessionID,
						SurfaceID:   s.ID,
						PaneRef:     s.PaneRef,
						SurfaceRef:  s.Ref,
						WorkspaceID: ws.ID,
						Title:       s.Title,
						Workspace:   ws.Title,
						CWD:         r.CWD,
						Focused:     s.Focused || activeIs(tree, s.ID),
						UpdatedAt:   r.UpdatedAtUnix,
					}
					if fp := transcriptPath(t.SessionID); fp != "" {
						if fi, err := os.Stat(fp); err == nil {
							t.UpdatedAt = float64(fi.ModTime().Unix())
						}
					}
					if t.CWD != "" {
						out = append(out, t)
					}
				}
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Focused != out[j].Focused {
			return out[i].Focused
		}
		return out[i].UpdatedAt > out[j].UpdatedAt
	})
	return out
}

// activeIs reports whether the tree's active surface has this id.
func activeIs(tree *cmux.Tree, id string) bool {
	return tree.Active != nil && tree.Active.SurfaceID == id
}

// CLI seams, overridable in tests.
var (
	runTree      = cmux.RunTree
	sessionsList = cmux.SessionsList
)

// ActiveTarget resolves the daemon's display target: the session row of the
// surface the user currently has frontmost. The cmux tree's `active` object
// is authoritative (what is actually frontmost right now); its surface_id
// matches exactly one row in `cmux sessions list`.
//
// When the frontmost surface has no live omp session — most often the omp
// summary sidebar itself, an empty shell tab, or a browser pane — the card
// the user is reading belongs to that surface's workspace, so the daemon
// shows the freshest live session in the same workspace instead of idling.
// Returns nil when there is nothing to show.
func ActiveTarget(ctx context.Context, now, maxAge float64) (*Target, error) {
	tree, err := runTree(ctx)
	if err != nil {
		return nil, fmt.Errorf("tree: %w", err)
	}
	rows, err := sessionsList(ctx, "--agent", "omp", "--all")
	if err != nil {
		return nil, fmt.Errorf("sessions list: %w", err)
	}

	activeID := ""
	if tree.Active != nil {
		activeID = tree.Active.SurfaceID
	}

	// bestRow returns the freshest live row for a surface id, or nil.
	bestRow := func(surfaceID string) *cmux.SessionRow {
		var row *cmux.SessionRow
		for _, r := range rows {
			if r.SurfaceID == nil || *r.SurfaceID != surfaceID {
				continue
			}
			if r.RuntimeStatus == "gone" || r.AgentLifecycle == "done" {
				continue
			}
			if maxAge > 0 && now-r.UpdatedAtUnix > maxAge {
				continue
			}
			if row == nil || r.UpdatedAtUnix > row.UpdatedAtUnix {
				r := r
				row = &r
			}
		}
		return row
	}

	// resolve builds a Target for a surface id present in the tree; ok=false
	// when the surface is gone or its session is gone, stale, or CWD-less.
	resolve := func(surfaceID string, focused bool) (*Target, bool) {
		row := bestRow(surfaceID)
		if row == nil || row.CWD == "" {
			return nil, false
		}
		t := &Target{
			SessionID: row.SessionID,
			SurfaceID: surfaceID,
			CWD:       row.CWD,
			Focused:   focused,
			UpdatedAt: row.UpdatedAtUnix,
		}
		// The session row carries no title; the tree's surface list does. The
		// row's workspace_id can be empty (hook-store dependent), so the
		// tree's workspace ID is the reliable source for the push.
		var found bool
		for _, w := range tree.Windows {
			for _, ws := range w.Workspaces {
				for _, p := range ws.Panes {
					for _, s := range p.Surfaces {
						if s.ID == surfaceID {
							t.PaneRef, t.SurfaceRef = s.PaneRef, s.Ref
							t.Title, t.Workspace, t.WorkspaceID = s.Title, ws.Title, ws.ID
							found = true
						}
					}
				}
			}
		}
		if !found {
			return nil, false
		}
		if fp := transcriptPath(t.SessionID); fp != "" {
			if fi, err := os.Stat(fp); err == nil {
				t.UpdatedAt = float64(fi.ModTime().Unix())
			}
		}
		return t, true
	}

	if activeID != "" {
		if t, ok := resolve(activeID, true); ok {
			return t, nil
		}
		// Workspace fallback: index surface ids by workspace, then pick the
		// freshest live session among the active surface's siblings.
		wsOf := map[string]string{}
		for _, w := range tree.Windows {
			for _, ws := range w.Workspaces {
				for _, p := range ws.Panes {
					for _, s := range p.Surfaces {
						wsOf[s.ID] = ws.ID
					}
				}
			}
		}
		wsID, known := wsOf[activeID]
		if known {
			bestSurface, bestAt := "", float64(0)
			for sid, wid := range wsOf {
				if wid != wsID {
					continue
				}
				if r := bestRow(sid); r != nil && r.UpdatedAtUnix > bestAt {
					bestSurface, bestAt = sid, r.UpdatedAtUnix
				}
			}
			if bestSurface != "" {
				if t, ok := resolve(bestSurface, false); ok {
					return t, nil
				}
			}
		}
	}
	return nil, nil
}

// extractText flattens message content (string or typed array) to plain text.
func extractText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err == nil {
		var b strings.Builder
		for _, blk := range blocks {
			if blk.Text != "" {
				if b.Len() > 0 {
					b.WriteByte(' ')
				}
				b.WriteString(blk.Text)
			}
		}
		return strings.TrimSpace(b.String())
	}
	return ""
}

// nowUnix is stubbed for tests.
var nowUnix = func() float64 { return float64(time.Now().Unix()) }
