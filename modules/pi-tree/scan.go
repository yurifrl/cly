package pitree

import (
	"bytes"
	"encoding/json"
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

// activeSessions returns the N most recently modified session files in a directory.
// Each returned session has IsOpen = true.
func activeSessions(dir string, maxCount int) []PiSession {
	if dir == "" || maxCount <= 0 {
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

	// Only take the top N most recently modified files
	if len(files) > maxCount {
		files = files[:maxCount]
	}

	var result []PiSession
	for _, f := range files {
		stem := strings.TrimSuffix(f.name, ".jsonl")
		parts := strings.SplitN(stem, "_", 2)
		sid := stem
		started := stem
		if len(parts) == 2 {
			sid = parts[1]
			started = parseSessionName(parts[0])
		}
		fullPath := filepath.Join(dir, f.name)
		result = append(result, PiSession{
			SessionID:   sid,
			SessionName: readSessionName(fullPath),
			StartedAt:   started,
			SizeBytes:   f.size,
			FilePath:    fullPath,
			IsOpen:      true,
		})
	}
	return result
}

// readSessionName extracts the name from the last session_info entry in a JSONL file.
func readSessionName(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	// Scan from the end for the last session_info line
	name := ""
	for _, line := range bytes.Split(data, []byte("\n")) {
		if !bytes.Contains(line, []byte(`"type":"session_info"`)) {
			continue
		}
		var entry struct {
			Name string `json:"name"`
		}
		if json.Unmarshal(line, &entry) == nil && entry.Name != "" {
			name = entry.Name
		}
	}
	return name
}

// cwdToSessionDir converts a working directory path to the pi session directory name.
// e.g. "/Users/yuri/Workdir/Yuri/cly" → "--Users-yuri-Workdir-Yuri-cly--"
func cwdToSessionDir(cwd string) string {
	slug := strings.TrimPrefix(cwd, "/")
	slug = strings.ReplaceAll(slug, "/", "-")
	return "--" + slug + "--"
}

// openPiSessions returns all running pi (node) processes and their cwds.
// Returns map of cwd → session dir path.
// openPiProcesses returns a map of CWD → count of running pi processes.
func openPiProcesses() map[string]int {
	// Find PIDs of processes whose command name is exactly "pi"
	pidOut, err := runCommand("bash", "-c",
		`ps -eo pid,comm | awk '$2 == "pi" {print $1}'`)
	if err != nil {
		return nil
	}
	pids := strings.Fields(strings.TrimSpace(string(pidOut)))
	if len(pids) == 0 {
		return nil
	}

	counts := make(map[string]int)
	for _, pid := range pids {
		out, err := runCommand("lsof", "-a", "-p", pid, "-d", "cwd", "-Fn")
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(line, "n/") {
				cwd := line[1:]
				if cwd != "/" {
					counts[cwd]++
				}
				break
			}
		}
	}
	return counts
}

// ScanTree queries cmux and reads pi session files to build the current tree.
func ScanTree() ([]WorkspaceNode, error) {
	// Get cmux workspaces for grouping
	wsOut, err := cmuxRun("list-workspaces")
	if err != nil {
		return nil, fmt.Errorf("cmux list-workspaces: %w", err)
	}

	type wsInfo struct {
		ref  string
		name string
	}
	var workspaces []wsInfo
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
			workspaces = append(workspaces, wsInfo{ref: ref, name: name})
		}
	}

	// Build cwd→workspace map (each workspace dir → its name/ref)
	allDirs := listAllSessionDirs()
	sessionBase := piSessionsDir()

	// Map: cwd → wsInfo
	cwdToWS := map[string]wsInfo{}
	for _, ws := range workspaces {
		slug := cwdSlugForWorkspace(ws.name)
		dir := ""
		if slug != "" {
			dir = findSessionDir(slug)
		}
		if dir == "" {
			dir = findSessionDirByName(ws.name, allDirs)
		}
		if dir != "" {
			cwd := sessionDirNameToWorkingDir(filepath.Base(dir))
			cwdToWS[cwd] = ws
		}
	}

	// Find running pi processes and their CWDs
	piCounts := openPiProcesses()

	// Scan session directories for active sessions
	type sessionEntry struct {
		cwd     string
		session PiSession
	}
	var entries []sessionEntry

	for _, d := range allDirs {
		dir := filepath.Join(sessionBase, d)
		cwd := sessionDirNameToWorkingDir(d)
		if cwd == "" {
			continue
		}
		// Skip dirs whose resolved path doesn't exist or points inside session storage
		if strings.HasPrefix(cwd, sessionBase) {
			continue
		}
		if info, err := os.Stat(cwd); err != nil || !info.IsDir() {
			continue
		}

		// Only include directories that have running pi processes
		count := piCounts[cwd]
		if count == 0 {
			continue
		}

		sessions := activeSessions(dir, count)
		if len(sessions) == 0 {
			continue
		}

		for _, s := range sessions {
			entries = append(entries, sessionEntry{cwd: cwd, session: s})
		}
	}

	if len(entries) == 0 {
		return nil, nil
	}

	// Group by workspace name; entries not matching a workspace get the dir basename
	wsNodes := map[string]*WorkspaceNode{}
	wsOrder := []string{}

	for _, e := range entries {
		ws, ok := cwdToWS[e.cwd]
		wsName := ws.name
		wsRef := ws.ref
		if !ok || wsName == "" {
			// Derive name from cwd
			wsName = filepath.Base(e.cwd)
		}
		if _, exists := wsNodes[wsName]; !exists {
			wsNodes[wsName] = &WorkspaceNode{
				Name:         wsName,
				WorkspaceRef: wsRef,
			}
			wsOrder = append(wsOrder, wsName)
		}
		wsNodes[wsName].Sessions = append(wsNodes[wsName].Sessions, e.session)
	}

	var nodes []WorkspaceNode
	for _, name := range wsOrder {
		nodes = append(nodes, *wsNodes[name])
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

// sessionDirNameToWorkingDir converts a session dir basename to the working dir path.
func sessionDirNameToWorkingDir(dirName string) string {
	dirName = strings.TrimPrefix(dirName, "--")
	dirName = strings.TrimSuffix(dirName, "--")
	if dirName == "" {
		return ""
	}
	parts := strings.Split(dirName, "-")
	result := resolveEncodedPath(parts)
	if result != "" {
		return result
	}
	return "/" + strings.ReplaceAll(dirName, "-", "/")
}

// allSessionsInDir returns all .jsonl sessions in a directory sorted newest first.
func allSessionsInDir(dir string) []PiSession {
	return activeSessions(dir, 0)
}
