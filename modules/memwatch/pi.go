package memwatch

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	pitree "github.com/yurifrl/cly/modules/pi-tree"
)

// procSample is one process matched by the shared ps scan.
type procSample struct {
	PID   int
	RSSKB int64
}

// scanAgentRSS returns processes whose ps comm basename is one of kinds,
// sorted by RSS descending. Shared by PiProcesses and OMPProcesses.
// macOS ps prints comm as a full path when stdout is not a TTY and as a
// short name when it is — match on the basename either way.
func scanAgentRSS(ctx context.Context, kinds ...string) ([]procSample, error) {
	kindSet := make(map[string]bool, len(kinds))
	for _, k := range kinds {
		kindSet[k] = true
	}
	out, err := exec.CommandContext(ctx, "bash", "-c",
		`ps -eo pid,rss,comm | awk '{ n = $3; sub(/.*\//, "", n); print n, $1, $2 }'`).Output()
	if err != nil {
		return nil, fmt.Errorf("ps failed: %w", err)
	}

	var samples []procSample
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 3 {
			continue
		}
		if !kindSet[fields[0]] {
			continue
		}
		pid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		rss, err := strconv.ParseInt(fields[2], 10, 64)
		if err != nil {
			continue
		}
		samples = append(samples, procSample{PID: pid, RSSKB: rss})
	}

	sort.Slice(samples, func(i, j int) bool { return samples[i].RSSKB > samples[j].RSSKB })
	return samples, nil
}

// PiProc is a single running pi instance.
type PiProc struct {
	PID          int      `json:"pid"`
	CWD          string   `json:"cwd"`
	Label        string   `json:"label"`         // short label derived from CWD
	RSSKB        int64    `json:"rss_kb"`
	Workspace    string   `json:"workspace,omitempty"`
	WorkspaceRef string   `json:"workspace_ref,omitempty"` // e.g. workspace:3
	SessionNames []string `json:"session_names,omitempty"`
}

// PiProcesses returns every running `pi` instance with its RSS, CWD, and
// (best-effort) matching pi-tree session names.
func PiProcesses(ctx context.Context) ([]PiProc, error) {
	samples, err := scanAgentRSS(ctx, "pi")
	if err != nil {
		return nil, err
	}

	// Build cwd -> (workspace, session names) map from pi-tree scan.
	type wsInfo struct {
		workspace string
		ref       string
		names     []string
	}
	cwdInfo := map[string]*wsInfo{}
	if tree, err := pitree.ScanTree(); err == nil {
		for _, ws := range tree {
			for _, s := range ws.Sessions {
				if !s.IsOpen || s.FilePath == "" {
					continue
				}
				cwd := sessionFilePathToCWD(s.FilePath)
				if cwd == "" {
					continue
				}
				if _, ok := cwdInfo[cwd]; !ok {
					cwdInfo[cwd] = &wsInfo{workspace: ws.Name, ref: ws.WorkspaceRef}
				}
				name := s.SessionName
				if name == "" {
					name = s.SessionID
				}
				cwdInfo[cwd].names = append(cwdInfo[cwd].names, name)
			}
		}
	}

	var procs []PiProc
	for _, s := range samples {
		cwd := processCWD(ctx, s.PID)
		p := PiProc{
			PID:   s.PID,
			CWD:   cwd,
			Label: labelForCWD(cwd),
			RSSKB: s.RSSKB,
		}
		if info, ok := cwdInfo[cwd]; ok {
			p.Workspace = info.workspace
			p.WorkspaceRef = info.ref
			p.SessionNames = info.names
		}
		procs = append(procs, p)
	}
	return procs, nil
}

func processCWD(ctx context.Context, pid int) string {
	out, err := exec.CommandContext(ctx, "lsof", "-a", "-p", strconv.Itoa(pid), "-d", "cwd", "-Fn").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "n/") {
			return line[1:]
		}
	}
	return ""
}

func labelForCWD(cwd string) string {
	if cwd == "" {
		return "?"
	}
	return filepath.Base(cwd)
}

// sessionFilePathToCWD reverses pi's session-dir encoding:
//   /Users/yuri/.pi/agent/sessions/--Users-yuri-Workdir-Yuri-cly--/<file>.jsonl
//   -> /Users/yuri/Workdir/Yuri/cly
func sessionFilePathToCWD(p string) string {
	dir := filepath.Base(filepath.Dir(p))
	if !strings.HasPrefix(dir, "--") || !strings.HasSuffix(dir, "--") {
		return ""
	}
	inner := strings.TrimPrefix(strings.TrimSuffix(dir, "--"), "--")
	return "/" + strings.ReplaceAll(inner, "-", "/")
}
