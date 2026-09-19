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
	SessionID  string
	SurfaceID  string
	PaneRef    string
	SurfaceRef string
	Title      string // surface title, or workspace title fallback
	Workspace  string
	CWD        string
	Focused    bool
	UpdatedAt  float64 // transcript mtime when known, else hook updated_at
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
						SessionID:  r.SessionID,
						SurfaceID:  s.ID,
						PaneRef:    s.PaneRef,
						SurfaceRef: s.Ref,
						Title:      s.Title,
						Workspace:  ws.Title,
						CWD:        r.CWD,
						Focused:    s.Focused || activeIs(tree, s.ID),
						UpdatedAt:  r.UpdatedAtUnix,
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

// Extract reads a transcript tail and flattens it into a compact, readable
// digest for the summarizer prompt.
//
// Line shape (verified): {"type":"message","message":{"role","content":
// [{"type":"text","text"}|{"type":"toolCall","name","intent",...}]}}, plus
// {"type":"custom","customType":"tool_execution_*","data":{...}} and
// {"type":"custom_message"}. Multi-MB files: only the last maxTail bytes are
// read; the first partial line is dropped; output capped at maxChars.
func Extract(path string, maxTail, maxChars int) (string, int, error) {
	if path == "" {
		return "", 0, fmt.Errorf("no transcript path")
	}
	fi, err := os.Stat(path)
	if err != nil {
		return "", 0, err
	}
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	read := int64(maxTail)
	if fi.Size() < read {
		read = fi.Size()
	}
	buf := make([]byte, read)
	if _, err := f.ReadAt(buf, fi.Size()-read); err != nil {
		return "", 0, err
	}
	lines := strings.Split(string(buf), "\n")
	if read < fi.Size() && len(lines) > 0 {
		lines = lines[1:] // drop partial first line
	}

	var parts []string
	turns := 0
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		var e struct {
			Type       string `json:"type"`
			CustomType string `json:"customType"`
			Message    *struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
			} `json:"message"`
			Data struct {
				ToolName string `json:"toolName"`
				Intent   string `json:"intent"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(ln), &e); err != nil {
			continue
		}
		switch {
		case e.Type == "message" && e.Message != nil:
			text := extractText(e.Message.Content)
			if text == "" {
				continue
			}
			role := "user"
			if e.Message.Role != "" {
				role = e.Message.Role
			}
			if role == "user" || role == "assistant" {
				turns++
			}
			parts = append(parts, role+": "+text)
		case e.Type == "custom" && e.Data.ToolName != "":
			parts = append(parts, "→ "+e.Data.ToolName+": "+e.Data.Intent)
		}
	}
	out := strings.Join(parts, "\n")
	if len(out) > maxChars {
		out = out[len(out)-maxChars:]
		if i := strings.IndexByte(out, '\n'); i > 0 {
			out = out[i+1:] // start on a line boundary
		}
	}
	return out, turns, nil
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
