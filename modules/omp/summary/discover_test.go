package ompsummary

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func treeFixture(focusedID string) *treeShape {
	return &treeShape{
		Windows: []treeWin{{
			Workspaces: []treeWs{{
				ID:    "WS1",
				Title: "cly",
				Panes: []treePane{{
					Ref:      "pane:1",
					Surfaces: []treeSurf{{ID: "S1", Ref: "surface:1", PaneRef: "pane:1", Focused: focusedID == "S1"}},
				}},
			}},
		}},
	}
}

func TestMergePicksActiveSessionForSurface(t *testing.T) {
	tree := treeFixture("S1")
	older := 1000.0
	newer := 1010.0
	rows := []cmuxRow{
		{Agent: "omp", SessionID: "idle-older", CWD: "/tmp/p", SurfaceID: strPtr("S1"), UpdatedAtUnix: older, ActiveForSurf: true},
		{Agent: "omp", SessionID: "newer-not-active", CWD: "/tmp/p", SurfaceID: strPtr("S1"), UpdatedAtUnix: newer},
	}
	targets := mergeFixture(tree, rows, 1100, 3600)
	if len(targets) != 1 {
		t.Fatalf("targets = %d, want 1", len(targets))
	}
	if targets[0].SessionID != "idle-older" {
		t.Fatalf("picked %q, want the active_for_surface session", targets[0].SessionID)
	}
	if !targets[0].Focused {
		t.Fatal("S1 is the tree-active surface; target should be focused")
	}
	if targets[0].Workspace != "cly" || targets[0].PaneRef != "pane:1" {
		t.Fatalf("target = %+v", targets[0])
	}
}

func TestMergeDropsStaleSessions(t *testing.T) {
	tree := treeFixture("S1")
	rows := []cmuxRow{
		{Agent: "omp", SessionID: "old", CWD: "/tmp/p", SurfaceID: strPtr("S1"), UpdatedAtUnix: 100},
	}
	if got := mergeFixture(tree, rows, 4000, 3600); len(got) != 0 {
		t.Fatalf("expected stale session dropped, got %+v", got)
	}
}

func TestMergeFocusedFirst(t *testing.T) {
	tree := &treeShape{
		Windows: []treeWin{{
			Workspaces: []treeWs{{
				ID:    "WS1",
				Title: "w",
				Panes: []treePane{{
					Ref:      "pane:1",
					Surfaces: []treeSurf{
						{ID: "S1", Ref: "surface:1", PaneRef: "pane:1"},
						{ID: "S2", Ref: "surface:2", PaneRef: "pane:1", Focused: true},
					},
				}},
			}},
		}},
	}
	rows := []cmuxRow{
		{Agent: "omp", SessionID: "bg", CWD: "/tmp/p", SurfaceID: strPtr("S1"), UpdatedAtUnix: 2000},
		{Agent: "omp", SessionID: "fg", CWD: "/tmp/q", SurfaceID: strPtr("S2"), UpdatedAtUnix: 1000},
	}
	got := mergeFixture(tree, rows, 2100, 3600)
	if len(got) != 2 {
		t.Fatalf("targets = %d, want 2", len(got))
	}
	if got[0].SessionID != "fg" || !got[0].Focused {
		t.Fatalf("first = %+v, want focused fg", got[0])
	}
	if got[1].SessionID != "bg" || got[1].Focused {
		t.Fatalf("second = %+v, want unfocused bg", got[1])
	}
}

func TestMergeFallsBackToMostRecentSession(t *testing.T) {
	// No active_for_surface row: the most recent session for the surface wins.
	tree := treeFixture("S1")
	rows := []cmuxRow{
		{Agent: "omp", SessionID: "older", CWD: "/tmp/p", SurfaceID: strPtr("S1"), UpdatedAtUnix: 1000},
		{Agent: "omp", SessionID: "newer", CWD: "/tmp/p", SurfaceID: strPtr("S1"), UpdatedAtUnix: 1005},
	}
	got := mergeFixture(tree, rows, 1100, 3600)
	if len(got) != 1 || got[0].SessionID != "newer" {
		t.Fatalf("got %+v, want the most recent session", got)
	}
}

// ---- Extract ----

func writeTranscript(t *testing.T, lines ...string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "s.jsonl")
	if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

var msgLine = func(role, text string) string {
	if role == "assistant-toolcall" {
		return `{"type":"message","id":"m","timestamp":"2026-09-19T00:00:00Z","message":{"role":"assistant","content":[{"type":"toolCall","id":"c1","name":"edit","arguments":{},"intent":"x"}]}}`
	}
	return `{"type":"message","id":"m","timestamp":"2026-09-19T00:00:00Z","message":{"role":"` + role + `","content":[{"type":"text","text":"` + text + `"}]}}`
}

func TestExtractTurnsAndDigest(t *testing.T) {
	p := writeTranscript(t,
		msgLine("user", "fix the login bug"),
		`{"type":"custom","customType":"tool_execution_start","data":{"toolName":"edit","intent":"fix login","toolCallId":"c1"}}`,
		msgLine("assistant", "fixed it"),
		msgLine("assistant-toolcall", ""),
	)
	digest, turns, err := Extract(p, 64<<10, 8<<10)
	if err != nil {
		t.Fatal(err)
	}
	if turns != 2 {
		t.Fatalf("turns = %d, want 2 (user+assistant text only)", turns)
	}
	for _, want := range []string{"user: fix the login bug", "→ edit: fix login", "assistant: fixed it"} {
		if !strings.Contains(digest, want) {
			t.Fatalf("digest missing %q:\n%s", want, digest)
		}
	}
	if strings.Contains(digest, "assistant-toolcall") {
		t.Fatal("toolCall content leaked into digest")
	}
}

func TestExtractDropsPartialFirstLine(t *testing.T) {
	// First line is a truncated JSON fragment (mid-line tail read); second complete.
	p := writeTranscript(t,
		`,"message":{"role":"user","content":[{"type":"text","text":"cut"}]},"id":"x"}`,
		msgLine("user", "full line"),
	)
	digest, turns, err := Extract(p, 1<<10, 8<<10)
	if err != nil {
		t.Fatal(err)
	}
	if turns != 1 || !strings.Contains(digest, "full line") {
		t.Fatalf("digest=%q turns=%d, want only the full line", digest, turns)
	}
	if strings.Contains(digest, "cut") {
		t.Fatalf("partial line leaked:\n%s", digest)
	}
}

func TestExtractLineBoundaryCap(t *testing.T) {
	long := strings.Repeat("x", 400)
	p := writeTranscript(t,
		msgLine("user", long),
		msgLine("assistant", short+long),
	)
	digest, _, err := Extract(p, 64<<10, 500)
	if err != nil {
		t.Fatal(err)
	}
	if len(digest) > 500 {
		t.Fatalf("digest = %d chars, want ≤ 500", len(digest))
	}
	if strings.HasPrefix(digest, strings.Repeat("x", 50)) {
		t.Fatal("cap should start on a line boundary")
	}
}

const short = "answer: "
