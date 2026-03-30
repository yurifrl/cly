package pitree

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testSnapshots() []Snapshot {
	return []Snapshot{
		{
			Version:   1,
			CreatedAt: time.Date(2026, 3, 28, 20, 55, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 3, 28, 20, 55, 0, 0, time.UTC),
			Tree: []WorkspaceNode{
				{Name: "cly", Sessions: []PiSession{
					{SessionID: "aaa", StartedAt: "2026-03-28 20:00", SizeBytes: 1024},
					{SessionID: "bbb", StartedAt: "2026-03-28 19:00", SizeBytes: 2048},
				}},
				{Name: "oncall", Sessions: []PiSession{
					{SessionID: "ccc", StartedAt: "2026-03-28 18:00", SizeBytes: 512},
				}},
			},
		},
		{
			Version:   2,
			CreatedAt: time.Date(2026, 3, 29, 13, 31, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 3, 29, 13, 37, 0, 0, time.UTC),
			Tree: []WorkspaceNode{
				{Name: "cly", Sessions: []PiSession{
					{SessionID: "ddd", StartedAt: "2026-03-29 13:00", SizeBytes: 4096},
				}},
			},
		},
		{
			Version:   3,
			CreatedAt: time.Date(2026, 3, 29, 14, 10, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 3, 29, 14, 17, 0, 0, time.UTC),
			Tree: []WorkspaceNode{
				{Name: "cly", Sessions: []PiSession{
					{SessionID: "eee", StartedAt: "2026-03-29 14:00", SizeBytes: 8192},
					{SessionID: "fff", StartedAt: "2026-03-29 14:05", SizeBytes: 1024},
				}},
				{Name: "Obsidian", Sessions: []PiSession{
					{SessionID: "ggg", StartedAt: "2026-03-29 14:10", SizeBytes: 3072},
				}},
			},
		},
	}
}

func testLiveNodes() []WorkspaceNode {
	return []WorkspaceNode{
		{Name: "cly", Sessions: []PiSession{
			{SessionID: "live-1", StartedAt: "2026-03-29 17:00", SizeBytes: 16384},
		}},
	}
}

func isolateState(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	origSnap := snapshotsPath
	origHist := lastHistIdxPath
	snapshotsPath = func() string { return filepath.Join(tmpDir, "snapshots.json") }
	lastHistIdxPath = func() string { return filepath.Join(tmpDir, "last-hist-idx") }
	t.Cleanup(func() {
		snapshotsPath = origSnap
		lastHistIdxPath = origHist
	})
}

func key(m tuiModel, k string) tuiModel {
	msg := tea.KeyPressMsg{Code: rune(k[0]), Text: k}
	updated, _ := m.Update(msg)
	return updated.(tuiModel)
}

func TestTUI_InitialState_ShowsLiveTree(t *testing.T) {
	isolateState(t)
	m := newTUIModel(testLiveNodes(), testSnapshots(), 0)

	assert.False(t, m.showHist)
	assert.Equal(t, len(testSnapshots())-1, m.histIdx)
	assert.Len(t, m.nodes, 1)
	assert.Equal(t, "cly", m.nodes[0].Name)
	assert.Equal(t, "live-1", m.nodes[0].Sessions[0].SessionID)
}

func TestTUI_InitialState_EmptyLive_FallsBackToSnapshot(t *testing.T) {
	isolateState(t)
	m := newTUIModel(nil, testSnapshots(), 0)

	assert.True(t, m.showHist)
	assert.Contains(t, m.message, "no live sessions")
	assert.True(t, len(m.nodes) > 0)
}

func TestTUI_HistoryToggle_KeepsSelectedSnapshot(t *testing.T) {
	isolateState(t)
	m := newTUIModel(testLiveNodes(), testSnapshots(), 0)

	m = key(m, "h")
	assert.True(t, m.showHist)

	// Navigate to v1 (index 0) — up twice from v3
	m = key(m, "k")
	m = key(m, "k")
	assert.Equal(t, 0, m.histIdx)

	// Close history — should keep v1's tree
	m = key(m, "h")
	assert.False(t, m.showHist)
	require.Len(t, m.nodes, 2)
	assert.Equal(t, "cly", m.nodes[0].Name)
	assert.Equal(t, "oncall", m.nodes[1].Name)
	assert.Len(t, m.nodes[0].Sessions, 2)
}

func TestTUI_HistoryEnter_LoadsSnapshotWithCorrectCount(t *testing.T) {
	isolateState(t)
	m := newTUIModel(testLiveNodes(), testSnapshots(), 0)

	m = key(m, "h")
	m = key(m, "k") // v3→v2
	m = key(m, "k") // v2→v1
	m = key(m, "enter")

	assert.False(t, m.showHist)
	assert.Contains(t, m.message, "v1")
	assert.Contains(t, m.message, "3 sessions")
	require.Len(t, m.nodes, 2)
}

func TestTUI_HistoryEnter_V2_ShowsCorrectSessionCount(t *testing.T) {
	isolateState(t)
	m := newTUIModel(testLiveNodes(), testSnapshots(), 0)

	m = key(m, "h")
	m = key(m, "k") // v3→v2
	m = key(m, "enter")

	assert.Contains(t, m.message, "1 sessions")
	require.Len(t, m.nodes, 1)
	assert.Equal(t, "ddd", m.nodes[0].Sessions[0].SessionID)
}

func TestTUI_RestoresLastHistIdx(t *testing.T) {
	isolateState(t)
	SaveLastHistIdx(1)

	m := newTUIModel(testLiveNodes(), testSnapshots(), 0)
	assert.Equal(t, 0, m.histIdx, "should restore to v1 (index 0)")
}

func TestTUI_ViewShowsHistoryPanel(t *testing.T) {
	isolateState(t)
	m := newTUIModel(testLiveNodes(), testSnapshots(), 0)
	m = key(m, "h")
	m.width = 120
	m.height = 40

	view := m.View().Content
	assert.Contains(t, view, "History")
	assert.Contains(t, view, "v1")
	assert.Contains(t, view, "v2")
	assert.Contains(t, view, "v3")
	assert.Contains(t, view, "3 sessions")
	assert.Contains(t, view, "1 sessions")
}

func TestTUI_Quit(t *testing.T) {
	isolateState(t)
	m := newTUIModel(testLiveNodes(), testSnapshots(), 0)
	m = key(m, "q")
	assert.True(t, m.quit)
}

func TestTUI_CursorMovement(t *testing.T) {
	isolateState(t)
	m := newTUIModel(testSnapshots()[0].Tree, testSnapshots(), 0)

	assert.Equal(t, 0, m.cur.ws)
	assert.Equal(t, 0, m.cur.sess)

	m = key(m, "j")
	assert.Equal(t, 0, m.cur.ws)
	assert.Equal(t, 1, m.cur.sess)

	m = key(m, "j")
	assert.Equal(t, 1, m.cur.ws)
	assert.Equal(t, 0, m.cur.sess)
}

func TestTUI_SearchFilter(t *testing.T) {
	isolateState(t)
	m := newTUIModel(testSnapshots()[2].Tree, nil, 0)

	m = key(m, "/")
	assert.Equal(t, filterSearch, m.filterMode)

	for _, ch := range "obs" {
		updated, _ := m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		m = updated.(tuiModel)
	}
	assert.Len(t, m.nodes, 1)
	assert.Equal(t, "Obsidian", m.nodes[0].Name)
}

func TestSnapshot_SoftDelete(t *testing.T) {
	isolateState(t)
	_, _, _ = Upsert(testSnapshots()[0].Tree, true)
	_, _, _ = Upsert(testSnapshots()[1].Tree, true)
	_, _, _ = Upsert(testSnapshots()[2].Tree, true)

	all, _ := LoadSnapshots()
	assert.Len(t, all, 3)
	assert.Len(t, ActiveSnapshots(all), 3)

	// Soft delete v2
	require.NoError(t, DeleteSnapshot(2))

	all, _ = LoadSnapshots()
	assert.Len(t, all, 3, "all snapshots still in file")
	active := ActiveSnapshots(all)
	assert.Len(t, active, 2, "only 2 active")
	assert.Equal(t, 1, active[0].Version)
	assert.Equal(t, 3, active[1].Version)
}

func TestCountSessions(t *testing.T) {
	assert.Equal(t, 3, countSessions(testSnapshots()[0].Tree))
	assert.Equal(t, 0, countSessions(nil))
	assert.Equal(t, 0, countSessions([]WorkspaceNode{}))
}

func TestSessionDirToWorkingDir(t *testing.T) {
	result := sessionDirToWorkingDir("/home/.pi/sessions/--Users-yuri-Workdir-Yuri-cly--/file.jsonl")
	assert.Equal(t, "/Users/yuri/Workdir/Yuri/cly", result)
	assert.Equal(t, "", sessionDirToWorkingDir(""))
}

func TestSnapshot_UpsertAndLoad(t *testing.T) {
	isolateState(t)
	tree := testSnapshots()[0].Tree

	snap, isNew, err := Upsert(tree, false)
	require.NoError(t, err)
	assert.True(t, isNew)
	assert.Equal(t, 1, snap.Version)

	snap, isNew, err = Upsert(tree, false)
	require.NoError(t, err)
	assert.False(t, isNew)
	assert.Equal(t, 1, snap.Version)

	tree2 := testSnapshots()[1].Tree
	snap, isNew, err = Upsert(tree2, false)
	require.NoError(t, err)
	assert.True(t, isNew)
	assert.Equal(t, 2, snap.Version)

	snap, isNew, err = Upsert(tree2, true)
	require.NoError(t, err)
	assert.True(t, isNew)
	assert.Equal(t, 3, snap.Version)

	loaded, err := LoadSnapshots()
	require.NoError(t, err)
	assert.Len(t, loaded, 3)
}

func TestLastHistIdx_SaveAndLoad(t *testing.T) {
	isolateState(t)

	assert.Equal(t, -1, LoadLastHistIdx())
	SaveLastHistIdx(3)
	assert.Equal(t, 3, LoadLastHistIdx())
	SaveLastHistIdx(1)
	assert.Equal(t, 1, LoadLastHistIdx())
}

func TestLastHistIdx_PersistsToFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "last-hist-idx")
	orig := lastHistIdxPath
	lastHistIdxPath = func() string { return path }
	t.Cleanup(func() { lastHistIdxPath = orig })

	SaveLastHistIdx(42)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "42", strings.TrimSpace(string(data)))
}

func TestViewOutput_ContainsTreeContent(t *testing.T) {
	isolateState(t)
	m := newTUIModel(testSnapshots()[0].Tree, testSnapshots(), 0)
	m.width = 120
	m.height = 40

	view := m.View().Content
	assert.Contains(t, view, "cly")
	assert.Contains(t, view, "aaa")
	assert.Contains(t, view, "oncall")
	assert.Contains(t, view, "ccc")
	assert.NotContains(t, view, "History")
}

func TestTUI_Save_CreatesNewVersion(t *testing.T) {
	isolateState(t)

	// Seed with one snapshot
	snaps := testSnapshots()
	tree := snaps[0].Tree
	_, _, err := Upsert(tree, true)
	require.NoError(t, err)

	loaded, _ := LoadSnapshots()
	m := newTUIModel(tree, loaded, 0)

	// Press 's' to save current view
	m = key(m, "s")
	assert.Contains(t, m.message, "📸 saved v2")
	assert.Contains(t, m.message, "3 sessions")

	// Verify snapshot was persisted
	loaded, err = LoadSnapshots()
	require.NoError(t, err)
	assert.Len(t, loaded, 2)
	assert.Equal(t, 2, loaded[1].Version)
}

func TestTUI_Save_ViewedSnapshot_CreatesNewVersion(t *testing.T) {
	isolateState(t)

	// Create 2 snapshots with different trees
	_, _, _ = Upsert(testSnapshots()[0].Tree, true) // v1: 3 sessions
	_, _, _ = Upsert(testSnapshots()[1].Tree, true) // v2: 1 session

	loaded, _ := LoadSnapshots()
	m := newTUIModel(testLiveNodes(), loaded, 0)

	// Open history, select v1
	m = key(m, "h")
	m = key(m, "k") // v2→v1
	m = key(m, "enter")
	assert.Contains(t, m.message, "v1")

	// Save the v1 view — should create v3 with v1's tree
	m = key(m, "s")
	assert.Contains(t, m.message, "📸 saved v3")
	assert.Contains(t, m.message, "3 sessions")

	loaded, _ = LoadSnapshots()
	require.Len(t, loaded, 3)
	assert.Equal(t, 3, countSessions(loaded[2].Tree))
}

func TestTUI_Save_EmptyTree_ShowsNothing(t *testing.T) {
	isolateState(t)
	m := newTUIModel(nil, nil, 0)
	m = key(m, "s")
	assert.Equal(t, "nothing to save", m.message)
}

func TestTUI_HelpBar_ShowsSave(t *testing.T) {
	isolateState(t)
	m := newTUIModel(testSnapshots()[0].Tree, testSnapshots(), 0)
	m.width = 120
	m.height = 40
	view := m.View().Content
	assert.Contains(t, view, "s: save")
}

func TestTUI_DeleteSnapshot_InHistory(t *testing.T) {
	isolateState(t)
	_, _, _ = Upsert(testSnapshots()[0].Tree, true) // v1
	_, _, _ = Upsert(testSnapshots()[1].Tree, true) // v2
	_, _, _ = Upsert(testSnapshots()[2].Tree, true) // v3

	loaded, _ := LoadSnapshots()
	require.Len(t, loaded, 3)

	m := newTUIModel(testLiveNodes(), loaded, 0)

	// Open history, navigate to v2 (middle)
	m = key(m, "h")
	m = key(m, "k") // v3→v2

	// Delete v2
	m = key(m, "d")
	assert.Contains(t, m.message, "deleted v2")

	// Verify v2 is soft-deleted
	all, _ := LoadSnapshots()
	assert.Len(t, all, 3, "all still in file")
	active := ActiveSnapshots(all)
	assert.Len(t, active, 2)
	assert.Equal(t, 1, active[0].Version)
	assert.Equal(t, 3, active[1].Version)
}

func TestTUI_DeleteSnapshot_LastOneBlocked(t *testing.T) {
	isolateState(t)
	_, _, _ = Upsert(testSnapshots()[0].Tree, true)

	loaded, _ := LoadSnapshots()
	m := newTUIModel(testLiveNodes(), loaded, 0)

	m = key(m, "h")
	m = key(m, "d")
	assert.Equal(t, "can't delete last snapshot", m.message)

	remaining, _ := LoadSnapshots()
	assert.Len(t, remaining, 1)
}

func TestDeleteSnapshot(t *testing.T) {
	isolateState(t)
	_, _, _ = Upsert(testSnapshots()[0].Tree, true)
	_, _, _ = Upsert(testSnapshots()[1].Tree, true)

	err := DeleteSnapshot(1)
	require.NoError(t, err)

	// All snapshots still in file (soft delete)
	loaded, _ := LoadSnapshots()
	require.Len(t, loaded, 2)
	assert.True(t, loaded[0].Deleted)

	// Only active ones filtered
	active := ActiveSnapshots(loaded)
	require.Len(t, active, 1)
	assert.Equal(t, 2, active[0].Version)
}

func TestDeleteSnapshot_NotFound(t *testing.T) {
	isolateState(t)
	_, _, _ = Upsert(testSnapshots()[0].Tree, true)

	err := DeleteSnapshot(99)
	assert.Error(t, err)
}

func TestTUI_HelpBar_ShowsDeleteInHistory(t *testing.T) {
	isolateState(t)
	m := newTUIModel(testLiveNodes(), testSnapshots(), 0)
	m = key(m, "h")
	m.width = 120
	m.height = 40
	view := m.View().Content
	assert.Contains(t, view, "d: delete")
}

func TestSessionName_InView(t *testing.T) {
	isolateState(t)
	tree := []WorkspaceNode{
		{Name: "cly", Sessions: []PiSession{
			{SessionID: "abc123", SessionName: "Fix TUI bugs", StartedAt: "2026-03-29 17:00", SizeBytes: 1024},
		}},
	}
	m := newTUIModel(tree, nil, 0)
	m.width = 120
	m.height = 40
	view := m.View().Content
	assert.Contains(t, view, "Fix TUI bugs")
}

func TestOpenSets_BuiltFromLiveNodes(t *testing.T) {
	isolateState(t)
	live := []WorkspaceNode{
		{Name: "cly", Sessions: []PiSession{
			{SessionID: "active-1", StartedAt: "2026-03-29 17:00"},
			{SessionID: "old-1", StartedAt: "2026-03-28 10:00"},
		}},
		{Name: "oncall", Sessions: []PiSession{
			{SessionID: "active-2", StartedAt: "2026-03-29 16:00"},
		}},
	}
	m := newTUIModel(live, nil, 0)

	assert.True(t, m.openWS["cly"])
	assert.True(t, m.openWS["oncall"])
	assert.False(t, m.openWS["missing"])

	assert.True(t, m.openSess["active-1"])
	assert.False(t, m.openSess["old-1"])
	assert.True(t, m.openSess["active-2"])
}
