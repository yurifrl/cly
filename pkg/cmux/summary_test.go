package cmux

import (
	"encoding/json"
	"testing"
)

// Fixture mirrors the live `cmux --id-format both tree --json` shape
// (windows → workspaces → panes → surfaces) including a null url.
const treeFixture = `{
  "active": {"pane_id":"P2","pane_ref":"pane:2","surface_id":"S2","surface_ref":"surface:2"},
  "caller": "cly",
  "windows": [
    {"id":"W1","ref":"window:1","index":0,"key":"k1","visible":true,"active":false,
     "current":false,"selected_workspace_id":"WS1","selected_workspace_ref":"workspace:1",
     "workspace_count":1,
     "workspaces": [
       {"id":"WS1","ref":"workspace:1","title":"cly","index":0,"active":true,
        "selected":true,"pinned":false,"description":"d","layout":"tall",
        "panes": [
          {"id":"P1","ref":"pane:1","index":0,"active":true,"focused":false,
           "surface_count":2,"surface_ids":["S1","S3"],"surface_refs":["surface:1","surface:3"],
           "selected_surface_id":"S1","selected_surface_ref":"surface:1",
           "surfaces": [
             {"id":"S1","ref":"surface:1","title":"π > cly","index":0,"index_in_pane":0,
              "pane_id":"P1","pane_ref":"pane:1","focused":true,"active":true,
              "selected":true,"selected_in_pane":true,"here":false,
              "render_health":"ok","tty":"ttys001","type":"terminal","url":null},
             {"id":"S3","ref":"surface:3","title":"build","index":1,"index_in_pane":1,
              "pane_id":"P1","pane_ref":"pane:1","focused":false,"active":false,
              "selected":false,"selected_in_pane":false,"here":false,
              "render_health":"not_started","tty":null,"type":"terminal","url":null}
           ]}
        ]}
    ]}
  ]
}`

func TestTreeDecodeTolerantOfNulls(t *testing.T) {
	var tree Tree
	if err := json.Unmarshal([]byte(treeFixture), &tree); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(tree.Windows) != 1 {
		t.Fatalf("windows = %d, want 1", len(tree.Windows))
	}
	ws := tree.Windows[0].Workspaces[0]
	if ws.ID != "WS1" || len(ws.Panes) != 1 {
		t.Fatalf("workspace = %+v", ws)
	}
	s1 := ws.Panes[0].Surfaces[0]
	if s1.ID != "S1" || s1.Ref != "surface:1" || s1.PaneRef != "pane:1" ||
		s1.Title != "π > cly" || !s1.Focused || !s1.Selected {
		t.Fatalf("surface S1 = %+v", s1)
	}
	if tree.Active == nil || tree.Active.SurfaceID != "S2" || tree.Active.PaneRef != "pane:2" {
		t.Fatalf("active = %+v", tree.Active)
	}
}

func TestSessionRowDecodeNullSurfaceID(t *testing.T) {
	raw := `{"session_id":"01a0b78a","agent":"omp","agent_lifecycle":"running",
		"runtime_status":"running","cwd":"/tmp/x","surface_id":null,
		"workspace_id":"WS1","pid":42,"updated_at_unix":1789787031.58,
		"transcript_backed":true,"active_for_surface":false,"active_for_workspace":true}`
	var row SessionRow
	if err := json.Unmarshal([]byte(raw), &row); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if row.SessionID != "01a0b78a" || row.CWD != "/tmp/x" {
		t.Fatalf("row = %+v", row)
	}
	if row.SurfaceID != nil {
		t.Fatalf("surface_id should decode null → nil, got %q", *row.SurfaceID)
	}
}

func TestSessionRowDecodeSurfaceID(t *testing.T) {
	raw := `{"session_id":"s1","agent":"omp","surface_id":"S1"}`
	var row SessionRow
	if err := json.Unmarshal([]byte(raw), &row); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if row.SurfaceID == nil || *row.SurfaceID != "S1" {
		t.Fatalf("surface_id = %v, want S1", row.SurfaceID)
	}
}
