package cmux

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Tree is the `cmux --id-format both tree --json` output, decoded tolerantly:
// nullable fields are pointers, unknown fields are ignored.
// TreeActive mirrors the live `active` object: ids of the surface/pane the
// user currently has frontmost (a distinct, flatter shape than TreeSurface).
type TreeActive struct {
	IsBrowserSurface bool   `json:"is_browser_surface"`
	PaneID           string `json:"pane_id"`
	PaneRef          string `json:"pane_ref"`
	SurfaceID        string `json:"surface_id"`
	SurfaceRef       string `json:"surface_ref"`
}

type Tree struct {
	Active  *TreeActive  `json:"active"`
	Windows []TreeWindow `json:"windows"`
}

type TreeWindow struct {
	ID         string          `json:"id"`
	Ref        string          `json:"ref"`
	Index      int             `json:"index"`
	Visible    bool            `json:"visible"`
	Active     bool            `json:"active"`
	Workspaces []TreeWorkspace `json:"workspaces"`
}

type TreeWorkspace struct {
	ID      string       `json:"id"`
	Ref     string       `json:"ref"`
	Title   string       `json:"title"`
	Index   int          `json:"index"`
	Active  bool         `json:"active"`
	Selected bool        `json:"selected"`
	Panes   []TreePane   `json:"panes"`
}

type TreePane struct {
	ID             string        `json:"id"`
	Ref            string        `json:"ref"`
	Index          int           `json:"index"`
	Focused        bool          `json:"focused"`
	SelectedSurfID string        `json:"selected_surface_id"`
	Surfaces       []TreeSurface `json:"surfaces"`
}

type TreeSurface struct {
	ID           string  `json:"id"`
	Ref          string  `json:"ref"`
	Title        string  `json:"title"`
	Index        int     `json:"index"`
	IndexInPane  int     `json:"index_in_pane"`
	PaneID       string  `json:"pane_id"`
	PaneRef      string  `json:"pane_ref"`
	Focused      bool    `json:"focused"`
	Active       bool    `json:"active"`
	Selected     bool    `json:"selected"`
	SelectedInPn bool    `json:"selected_in_pane"`
	Type         string  `json:"type"`
	TTY          string  `json:"tty"`
	URL          *string `json:"url"`
}

// Tree runs `cmux --id-format both tree --json` and decodes the result.
// Refs are only present because of --id-format both; without it refs are
// empty and downstream focus calls silently no-match.
func RunTree(ctx context.Context) (*Tree, error) {
	out, err := exec.CommandContext(ctx, "cmux", "--id-format", "both", "tree", "--json").Output()
	if err != nil {
		return nil, fmt.Errorf("cmux tree: %w", err)
	}
	var t Tree
	if err := json.Unmarshal(out, &t); err != nil {
		return nil, fmt.Errorf("cmux tree decode: %w", err)
	}
	return &t, nil
}

// SessionRow is one record from `cmux sessions list --json`.
type SessionRow struct {
	SessionID       string   `json:"session_id"`
	Agent           string   `json:"agent"`
	AgentLifecycle  string   `json:"agent_lifecycle"`
	RuntimeStatus   string   `json:"runtime_status"`
	CWD             string   `json:"cwd"`
	SurfaceID       *string  `json:"surface_id"`
	WorkspaceID     string   `json:"workspace_id"`
	PID             int      `json:"pid"`
	UpdatedAtUnix   float64  `json:"updated_at_unix"`
	TranscriptBackd bool     `json:"transcript_backed"`
	ActiveForSurf   bool     `json:"active_for_surface"`
	ActiveForWrkspc bool     `json:"active_for_workspace"`
}

// SessionsList runs `cmux sessions list ... --json` and returns the decoded
// rows. Extra args (e.g. "--agent", "omp") are appended after the subcommand.
func SessionsList(ctx context.Context, extra ...string) ([]SessionRow, error) {
	args := append([]string{"sessions", "list"}, extra...)
	args = append(args, "--json")
	out, err := exec.CommandContext(ctx, "cmux", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("cmux sessions list: %w", err)
	}
	var doc struct {
		Sessions []SessionRow `json:"sessions"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		return nil, fmt.Errorf("cmux sessions decode: %w", err)
	}
	return doc.Sessions, nil
}

// FocusPane raises the pane containing a surface via `cmux focus-pane`.
func FocusPane(ctx context.Context, paneRef string) error {
	if paneRef == "" {
		return fmt.Errorf("focus-pane: empty pane ref")
	}
	out, err := exec.CommandContext(ctx, "cmux", "focus-pane", "--pane", paneRef).CombinedOutput()
	if err != nil {
		return fmt.Errorf("cmux focus-pane %s: %w: %s", paneRef, err, strings.TrimSpace(string(out)))
	}
	return nil
}
