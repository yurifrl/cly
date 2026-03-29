package pitree

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// workspaceSlugs maps workspace name → pi session dir slug.
// Best-effort; unknown workspaces fall back to slug derivation.
var workspaceSlugs = map[string]string{
	"pi":                                "pi-my-extensions",
	"loadtests":                         "monorepo-loadtest",
	"π - bff-graphql":                   "monorepo-loadtest",
	"π - bff-graphql - monorepo-loadtest": "monorepo-loadtest",
	"π - DotFiles":                      "DotFiles",
	"incident-review":                   "tidbits-oncall-incident-review",
	"deeb":                              "deeb",
	"Obsidian":                          "Obsidian",
	"argus":                             "argus",
	"forge":                             "forge",
	"portal":                            "resilience-portal",
	"yarb":                              "yarb",
	"bmm":                               "tidbits-oncall-projects-bmm-certification",
	"π - consoly":                       "consoly",
	"allrepo":                           "all-repos",
	"slides":                            "aihub-slides_incident-review",
	"cmux":                              "cmux",
}

var sessionNameRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})T(\d{2})-(\d{2})`)

func parseSessionName(stem string) string {
	m := sessionNameRe.FindStringSubmatch(stem)
	if m != nil {
		return fmt.Sprintf("%s %s:%s", m[1], m[2], m[3])
	}
	return stem[:16]
}

// cmuxRun runs a cmux command and returns stdout.
func cmuxRun(args ...string) ([]byte, error) {
	// Use os/exec directly to avoid import cycles
	out, err := runCommand("cmux", args...)
	return out, err
}

// piSessionsDir returns the ~/.pi/agent/sessions base directory.
func piSessionsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".pi", "agent", "sessions")
}

// cwdSlugForWorkspace returns the session dir slug for a workspace name.
func cwdSlugForWorkspace(name string) string {
	// Direct lookup first
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
	// Prefer exact suffix match
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		lower := strings.ToLower(e.Name())
		if strings.HasSuffix(lower, "--"+slug+"--") || strings.HasSuffix(lower, "-"+slug+"--") {
			return filepath.Join(base, e.Name())
		}
	}
	// Fallback: contains
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
		})
	}
	return result
}

// workspaceInfo holds parsed cmux workspace data.
type workspaceInfo struct {
	name      string
	piCount   int
}

// parseSurfaces parses `cmux list-pane-surfaces --workspace <ws>` output.
// Returns number of surfaces whose title starts with "π".
func countPiSurfaces(output []byte) int {
	count := 0
	for _, line := range bytes.Split(output, []byte("\n")) {
		s := strings.TrimSpace(string(line))
		// lines look like: "* surface:65  π · cmux  [selected]"
		// strip leading "* " or "  "
		s = strings.TrimLeft(s, "* ")
		if strings.HasPrefix(s, "surface:") {
			// find title after the ref
			parts := strings.SplitN(s, "  ", 2)
			if len(parts) == 2 {
				title := strings.TrimSpace(parts[1])
				if strings.HasPrefix(title, "π") {
					count++
				}
			}
		}
	}
	return count
}

// parseWorkspaceNames parses `cmux list-workspaces` output.
// Returns slice of workspace names (no refs).
func parseWorkspaceNames(output []byte) []workspaceInfo {
	var result []workspaceInfo
	for _, line := range bytes.Split(output, []byte("\n")) {
		s := strings.TrimSpace(string(line))
		if s == "" {
			continue
		}
		// lines: "  workspace:5  pi" or "* workspace:18  cmux  [selected]"
		s = strings.TrimLeft(s, "* ")
		// split on double-space
		parts := strings.SplitN(s, "  ", 2)
		if len(parts) != 2 {
			continue
		}
		ref := strings.TrimSpace(parts[0])   // "workspace:5"
		rest := strings.TrimSpace(parts[1])  // "pi" or "cmux  [selected]"
		rest = strings.TrimSuffix(rest, "  [selected]")
		rest = strings.TrimSpace(rest)
		if !strings.HasPrefix(ref, "workspace:") {
			continue
		}
		result = append(result, workspaceInfo{name: rest})
	}
	return result
}

// ScanTree queries cmux and reads pi session files to build the current tree.
func ScanTree() ([]WorkspaceNode, error) {
	wsOut, err := cmuxRun("list-workspaces")
	if err != nil {
		return nil, fmt.Errorf("cmux list-workspaces: %w", err)
	}

	workspaces := parseWorkspaceNames(wsOut)
	if len(workspaces) == 0 {
		return nil, fmt.Errorf("no workspaces found")
	}

	// Get workspace refs so we can query surfaces
	// Re-parse to get ref→name mapping
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

	// Deduplicate by name (some workspaces share same cwd slug)
	seen := map[string]bool{}
	var nodes []WorkspaceNode
	for _, r := range refs {
		surfOut, err := cmuxRun("list-pane-surfaces", "--workspace", r.ref)
		if err != nil {
			continue
		}
		n := countPiSurfaces(surfOut)
		if n == 0 {
			continue
		}

		slug := cwdSlugForWorkspace(r.name)
		dir := findSessionDir(slug)
		sessions := topSessions(dir, n)
		if len(sessions) == 0 {
			continue
		}

		// Deduplicate nodes with same name (e.g. π - bff-graphql shares monorepo slug)
		key := r.name + ":" + slug
		if seen[key] {
			continue
		}
		seen[key] = true

		nodes = append(nodes, WorkspaceNode{
			Name:     r.name,
			Sessions: sessions,
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


