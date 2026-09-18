package memwatch

import (
	"context"
	"encoding/json"
	"os/exec"
	"time"
)

// OmpProc is a single running omp instance.
type OmpProc struct {
	PID          int    `json:"pid"`
	CWD          string `json:"cwd"`
	Label        string `json:"label"` // short label derived from CWD
	RSSKB        int64  `json:"rss_kb"`
	Workspace    string `json:"workspace,omitempty"`
	WorkspaceRef string `json:"workspace_ref,omitempty"` // e.g. workspace:3
}

// cmuxWorkspace is one entry of `cmux workspace list --json`.
type cmuxWorkspace struct {
	Ref              string `json:"ref"`
	CustomTitle      string `json:"custom_title"`
	CurrentDirectory string `json:"current_directory"`
}

// cmuxWorkspaces lists open cmux workspaces (ref, title, cwd). Best-effort:
// returns nil when the cmux binary is missing or the call fails.
func cmuxWorkspaces(ctx context.Context) []cmuxWorkspace {
	if _, err := exec.LookPath("cmux"); err != nil {
		return nil
	}
	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(cctx, "cmux", "workspace", "list", "--json").Output()
	if err != nil {
		return nil
	}
	var parsed struct {
		Workspaces []cmuxWorkspace `json:"workspaces"`
	}
	if json.Unmarshal(out, &parsed) != nil {
		return nil
	}
	return parsed.Workspaces
}

// OMPProcesses returns every running `omp` instance with its RSS and CWD,
// mapped (best-effort) to an open cmux workspace by working directory.
func OMPProcesses(ctx context.Context) ([]OmpProc, error) {
	samples, err := scanAgentRSS(ctx, "omp")
	if err != nil {
		return nil, err
	}

	// cwd -> cmux workspace (title, ref).
	wsByCWD := map[string]cmuxWorkspace{}
	for _, w := range cmuxWorkspaces(ctx) {
		if w.CurrentDirectory != "" && w.Ref != "" {
			wsByCWD[w.CurrentDirectory] = w
		}
	}

	var procs []OmpProc
	for _, s := range samples {
		cwd := processCWD(ctx, s.PID)
		p := OmpProc{
			PID:   s.PID,
			CWD:   cwd,
			Label: labelForCWD(cwd),
			RSSKB: s.RSSKB,
		}
		if w, ok := wsByCWD[cwd]; ok {
			p.Workspace = w.CustomTitle
			p.WorkspaceRef = w.Ref
		}
		procs = append(procs, p)
	}
	return procs, nil
}
