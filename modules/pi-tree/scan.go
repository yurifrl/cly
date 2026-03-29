package pitree

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// workspaceSlugs maps workspace name → pi session dir slug.
// Best-effort; unknown workspaces fall back to fuzzy name matching.
var workspaceSlugs = map[string]string{
	"pi":                                  "pi-my-extensions",
	"loadtests":                           "monorepo-loadtest",
	"π - bff-graphql":                     "monorepo-loadtest",
	"π - bff-graphql - monorepo-loadtest": "monorepo-loadtest",
	"π - DotFiles":                        "DotFiles",
	"incident-review":                     "tidbits-oncall-incident-review",
	"deeb":                                "deeb",
	"Obsidian":                            "Obsidian",
	"argus":                               "argus",
	"forge":                               "forge",
	"portal":                              "resilience-portal",
	"yarb":                                "yarb",
	"bmm":                                 "tidbits-oncall-projects-bmm-certification",
	"π - consoly":                         "consoly",
	"allrepo":                             "all-repos",
	"slides":                              "aihub-slides_incident-review",
	"cmux":                                "cmux",
}

func parseSessionName(stem string) string {
	parts := strings.SplitN(stem, "T", 2)
	if len(parts) == 2 && len(parts[0]) == 10 {
		timePart := strings.ReplaceAll(parts[1], "-", ":")
		if len(timePart) >= 5 {
			return parts[0] + " " + timePart[:5]
		}
	}
	if len(stem) >= 16 {
		return stem[:16]
	}
	return stem
}

// cmuxRun runs a cmux command and returns stdout.
func cmuxRun(args ...string) ([]byte, error) {
	return runCommand("cmux", args...)
}

// piSessionsDir returns the ~/.pi/agent/sessions base directory.
func piSessionsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".pi", "agent", "sessions")
}

// cwdSlugForWorkspace returns the session dir slug for a workspace name.
func cwdSlugForWorkspace(name string) string {
	for wsName, slug := range workspaceSlugs {
		if strings.EqualFold(wsName, name) {
			return slug
		}
	}
	return ""
}

// findSessionDir locates the actual directory under ~/.pi/agent/sessions
// whose name contains the slug.
func findSessionDir(slug string) string {
	if slug == "" {
		return ""
	}
	base := piSessionsDir()
	entries, err := os.ReadDir(base)
	if err != nil {
		return ""
	}
	slug = strings.ToLower(slug)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		lower := strings.ToLower(e.Name())
		if strings.HasSuffix(lower, "--"+slug+"--") || strings.HasSuffix(lower, "-"+slug+"--") {
			return filepath.Join(base, e.Name())
		}
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if strings.Contains(strings.ToLower(e.Name()), slug) {
			return filepath.Join(base, e.Name())
		}
	}
	return ""
}

// listAllSessionDirs returns all directory names under ~/.pi/agent/sessions.
func listAllSessionDirs() []string {
	base := piSessionsDir()
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	return dirs
}

// findSessionDirByName tries to match a workspace name to a session directory.
// Session dirs look like "--Users-yuri-Workdir-Yuri-cly--"
// We match if the dir name ends with the workspace name (case-insensitive).
func findSessionDirByName(wsName string, allDirs []string) string {
	base := piSessionsDir()
	wsLower := strings.ToLower(wsName)

	// Exact suffix: dir ends with -<name>--
	for _, d := range allDirs {
		lower := strings.ToLower(d)
		lower = strings.TrimSuffix(lower, "--")
		if strings.HasSuffix(lower, "-"+wsLower) || lower == wsLower {
			return filepath.Join(base, d)
		}
	}
	// Contains match
	for _, d := range allDirs {
		if strings.Contains(strings.ToLower(d), wsLower) {
			return filepath.Join(base, d)
		}
	}
	return ""
}

// topSessions returns the N most recently modified session files.
func topSessions(dir string, n int) []PiSession {
	if dir == "" || n == 0 {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	type fileInfo struct {
		name  string
		mtime time.Time
		size  int64
	}
	var files []fileInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, fileInfo{name: e.Name(), mtime: info.ModTime(), size: info.Size()})
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].mtime.After(files[j].mtime)
	})

	var result []PiSession
	for i := 0; i < n && i < len(files); i++ {
		f := files[i]
		stem := strings.TrimSuffix(f.name, ".jsonl")
		parts := strings.SplitN(stem, "_", 2)
		sid := stem
		started := stem
		if len(parts) == 2 {
			sid = parts[1]
			started = parseSessionName(parts[0])
		}
		result = append(result, PiSession{
			SessionID: sid,
			StartedAt: started,
			SizeBytes: f.size,
			FilePath:  filepath.Join(dir, f.name),
		})
	}
	return result
}

// ScanTree queries cmux and reads pi session files to build the current tree.
func ScanTree() ([]WorkspaceNode, error) {
	wsOut, err := cmuxRun("list-workspaces")
	if err != nil {
		return nil, fmt.Errorf("cmux list-workspaces: %w", err)
	}

	type wsRef struct {
		ref  string
		name string
	}
	var refs []wsRef
	for _, line := range bytes.Split(wsOut, []byte("\n")) {
		s := strings.TrimSpace(string(line))
		if s == "" {
			continue
		}
		s = strings.TrimLeft(s, "* ")
		parts := strings.SplitN(s, "  ", 2)
		if len(parts) != 2 {
			continue
		}
		ref := strings.TrimSpace(parts[0])
		name := strings.TrimSpace(parts[1])
		name = strings.TrimSuffix(name, "  [selected]")
		name = strings.TrimSpace(name)
		if strings.HasPrefix(ref, "workspace:") {
			refs = append(refs, wsRef{ref: ref, name: name})
		}
	}

	if len(refs) == 0 {
		return nil, fmt.Errorf("no workspaces found")
	}

	// Build list of all session dirs for fuzzy matching
	allSessionDirs := listAllSessionDirs()

	seen := map[string]bool{}
	var nodes []WorkspaceNode
	for _, r := range refs {
		// Try slug map first, then fuzzy match by workspace name
		dir := ""
		slug := cwdSlugForWorkspace(r.name)
		if slug != "" {
			dir = findSessionDir(slug)
		}
		if dir == "" {
			dir = findSessionDirByName(r.name, allSessionDirs)
		}

		sessions := topSessions(dir, 5)
		if len(sessions) == 0 {
			continue
		}

		if seen[r.name] {
			continue
		}
		seen[r.name] = true

		nodes = append(nodes, WorkspaceNode{
			Name:         r.name,
			WorkspaceRef: r.ref,
			Sessions:     sessions,
		})
	}

	// Sort by most recent session first
	sort.Slice(nodes, func(i, j int) bool {
		if len(nodes[i].Sessions) == 0 {
			return false
		}
		if len(nodes[j].Sessions) == 0 {
			return true
		}
		return nodes[i].Sessions[0].StartedAt > nodes[j].Sessions[0].StartedAt
	})

	return nodes, nil
}
