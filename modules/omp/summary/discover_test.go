package ompsummary

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yurifrl/cly/pkg/cmux"
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

func TestDigestTailTurnsAndLines(t *testing.T) {
	p := writeTranscript(t,
		msgLine("user", "fix the login bug"),
		`{"type":"custom","customType":"tool_execution_start","data":{"toolName":"edit","intent":"fix login","toolCallId":"c1"}}`,
		msgLine("assistant", "fixed it"),
		msgLine("assistant-toolcall", ""),
	)
	d, out, err := DigestTail(p, 64<<10, 8<<10)
	if err != nil {
		t.Fatal(err)
	}
	if d.Turns != 2 {
		t.Fatalf("turns = %d, want 2 (user+assistant text only)", d.Turns)
	}
	if d.FirstUser != "fix the login bug" || d.LastUser != "fix the login bug" {
		t.Fatalf("user capture = %q / %q", d.FirstUser, d.LastUser)
	}
	if d.LastAssist != "fixed it" {
		t.Fatalf("LastAssist = %q", d.LastAssist)
	}
	if d.LastTool != "edit: fix login" {
		t.Fatalf("LastTool = %q", d.LastTool)
	}
	if d.Exit != ExitUnknown {
		t.Fatalf("Exit = %d, want ExitUnknown without a toolResult", d.Exit)
	}
	for _, want := range []string{"user: fix the login bug", "→ edit: fix login", "assistant: fixed it"} {
		if !strings.Contains(out, want) {
			t.Fatalf("tail missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "assistant-toolcall") {
		t.Fatal("toolCall content leaked into tail")
	}
}

func TestDigestTailExtractsLastExitCode(t *testing.T) {
	p := writeTranscript(t,
		msgLine("user", "run the build"),
		msgLine("toolResult", "edit exited with code 0; later build exited with code 3"),
	)
	d, out, err := DigestTail(p, 64<<10, 8<<10)
	if err != nil {
		t.Fatal(err)
	}
	if d.Exit != 3 {
		t.Fatalf("Exit = %d, want 3 (last match wins)", d.Exit)
	}
	if !strings.Contains(out, "→ edit exited with code") {
		t.Fatalf("toolResult line missing from tail:\n%s", out)
	}
}

func TestDigestTailSkipsBrokenLine(t *testing.T) {
	// A truncated JSON fragment mid-file fails to parse and is skipped, not fatal.
	p := writeTranscript(t,
		`{"type":"message","id":"m","timestamp":"2026-09-19T00:00:0`,
		msgLine("user", "full line"),
	)
	d, out, err := DigestTail(p, 64<<10, 8<<10)
	if err != nil {
		t.Fatal(err)
	}
	if d.Turns != 1 || !strings.Contains(out, "full line") {
		t.Fatalf("digest=%+v tail=%q, want only the full line", d, out)
	}
}

func TestDigestTailDropsPartialFirstLine(t *testing.T) {
	// File larger than maxTail: the read starts mid line 1 and its remainder
	// is a broken fragment; the drop must discard it and keep line 2 intact.
	cut := msgLine("user", "cut")
	full := msgLine("user", "full line")
	p := filepath.Join(t.TempDir(), "s.jsonl")
	if err := os.WriteFile(p, []byte(cut+"\n"+full+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	maxTail := len(cut)/2 + 1 + len(full) + 1
	if maxTail >= len(cut)+1+len(full)+1 {
		t.Fatal("test setup: maxTail must be smaller than the file")
	}
	d, out, err := DigestTail(p, maxTail, 8<<10)
	if err != nil {
		t.Fatal(err)
	}
	if d.Turns != 1 || !strings.Contains(out, "full line") {
		t.Fatalf("digest=%+v tail=%q, want only the full line", d, out)
	}
	if strings.Contains(out, "cut") {
		t.Fatalf("partial line leaked:\n%s", out)
	}
}

func TestDigestTailLineBoundaryCap(t *testing.T) {
	long := strings.Repeat("x", 400)
	p := writeTranscript(t,
		msgLine("user", long),
		msgLine("assistant", short+long),
	)
	_, out, err := DigestTail(p, 64<<10, 500)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) > 500 {
		t.Fatalf("tail = %d chars, want ≤ 500", len(out))
	}
	if strings.HasPrefix(out, strings.Repeat("x", 50)) {
		t.Fatal("cap should start on a line boundary")
	}
}

const short = "answer: "

// seamFixture installs canned tree+rows into the CLI seams and restores the
// originals on cleanup.
func seamFixture(t *testing.T, tree *cmux.Tree, rows []cmux.SessionRow) {
	t.Helper()
	oldTree, oldRows := runTree, sessionsList
	runTree = func(context.Context) (*cmux.Tree, error) { return tree, nil }
	sessionsList = func(context.Context, ...string) ([]cmux.SessionRow, error) { return rows, nil }
	t.Cleanup(func() { runTree, sessionsList = oldTree, oldRows })
}

// stickyTree: S1 is a live omp terminal, S2 is the custom sidebar (no
// session row ever attaches to it).
func stickyTree() *cmux.Tree {
	return &cmux.Tree{
		Windows: []cmux.TreeWindow{{
			Workspaces: []cmux.TreeWorkspace{{
				ID:    "WS1",
				Title: "cly",
				Panes: []cmux.TreePane{{
					Ref: "pane:1",
					Surfaces: []cmux.TreeSurface{
						{ID: "S1", Ref: "surface:1", PaneRef: "pane:1", Title: "omp working", Type: "terminal"},
						{ID: "S2", Ref: "surface:2", PaneRef: "pane:1", Title: "omp cards", Type: "customSidebar"},
					},
				}},
			}},
		}},
	}
}

func stickyRow(surface string) cmux.SessionRow {
	return cmux.SessionRow{
		SessionID:     "sess-1",
		Agent:         "omp",
		RuntimeStatus: "running",
		CWD:           "/tmp/proj",
		SurfaceID:     &surface,
		UpdatedAtUnix: 1900,
	}
}

func TestActiveTargetPrefersFocusedTerminal(t *testing.T) {
	tree := stickyTree()
	tree.Active = &cmux.TreeActive{SurfaceID: "S1", PaneRef: "pane:1", SurfaceRef: "surface:1"}
	seamFixture(t, tree, []cmux.SessionRow{stickyRow("S1")})

	tgt, err := ActiveTarget(context.Background(), 2000, 600)
	if err != nil || tgt == nil {
		t.Fatalf("target=%+v err=%v, want focused S1", tgt, err)
	}
	if tgt.SurfaceID != "S1" || !tgt.Focused || tgt.SessionID != "sess-1" {
		t.Fatalf("target=%+v, want focused sess-1 on S1", tgt)
	}
	if tgt.WorkspaceID != "WS1" || tgt.Title != "omp working" {
		t.Fatalf("target=%+v, want workspace WS1 and surface title", tgt)
	}
}

func TestActiveTargetFallsBackToWorkspaceSessionWhenSidebarFocused(t *testing.T) {
	tree := stickyTree()
	tree.Active = &cmux.TreeActive{SurfaceID: "S2", PaneRef: "pane:1", SurfaceRef: "surface:2"}
	seamFixture(t, tree, []cmux.SessionRow{stickyRow("S1")})

	tgt, err := ActiveTarget(context.Background(), 2000, 600)
	if err != nil || tgt == nil {
		t.Fatalf("target=%+v err=%v, want workspace fallback to S1", tgt, err)
	}
	if tgt.SurfaceID != "S1" || tgt.Focused {
		t.Fatalf("target=%+v, want unfocused fallback on S1", tgt)
	}
	if tgt.WorkspaceID != "WS1" {
		t.Fatalf("target=%+v, want workspace WS1", tgt)
	}
}

func TestActiveTargetNilWhenWorkspaceHasNoLiveSession(t *testing.T) {
	tree := stickyTree()
	tree.Active = &cmux.TreeActive{SurfaceID: "S2", PaneRef: "pane:1", SurfaceRef: "surface:2"}
	row := stickyRow("S1")
	row.RuntimeStatus = "gone"
	seamFixture(t, tree, []cmux.SessionRow{row})

	tgt, err := ActiveTarget(context.Background(), 2000, 600)
	if tgt != nil || err != nil {
		t.Fatalf("target=%+v err=%v, want nil (no live session in workspace)", tgt, err)
	}
}

func TestActiveTargetWorkspaceFallbackPicksFreshestSibling(t *testing.T) {
	tree := stickyTree()
	tree.Windows[0].Workspaces[0].Panes[0].Surfaces = append(
		tree.Windows[0].Workspaces[0].Panes[0].Surfaces,
		cmux.TreeSurface{ID: "S3", Ref: "surface:3", PaneRef: "pane:1", Title: "omp other", Type: "terminal"},
	)
	tree.Active = &cmux.TreeActive{SurfaceID: "S2", PaneRef: "pane:1", SurfaceRef: "surface:2"}
	old := stickyRow("S1")
	old.UpdatedAtUnix = 1800
	fresh := stickyRow("S3")
	fresh.SessionID = "sess-3"
	fresh.UpdatedAtUnix = 1950
	seamFixture(t, tree, []cmux.SessionRow{old, fresh})

	tgt, err := ActiveTarget(context.Background(), 2000, 600)
	if err != nil || tgt == nil {
		t.Fatalf("target=%+v err=%v, want freshest sibling S3", tgt, err)
	}
	if tgt.SurfaceID != "S3" || tgt.SessionID != "sess-3" {
		t.Fatalf("target=%+v, want sess-3 on S3", tgt)
	}
}

func TestActiveTargetIgnoresStaleRow(t *testing.T) {
	tree := stickyTree()
	tree.Active = &cmux.TreeActive{SurfaceID: "S1", PaneRef: "pane:1", SurfaceRef: "surface:1"}
	row := stickyRow("S1")
	row.UpdatedAtUnix = 100 // 1900s old, maxAge 600
	seamFixture(t, tree, []cmux.SessionRow{row})

	tgt, err := ActiveTarget(context.Background(), 2000, 600)
	if tgt != nil || err != nil {
		t.Fatalf("target=%+v err=%v, want nil (stale row)", tgt, err)
	}
}
