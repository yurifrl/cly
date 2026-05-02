package gitcommits

import (
	"fmt"
	"path/filepath"
	"strings"
)

// matchIgnore reports whether path matches any of the given gitignore-like patterns.
// Supported forms:
//   - "dir/"          -> matches any path under dir/
//   - "name"          -> matches basename via filepath.Match
//   - "a/b/*.go"      -> matches full path via filepath.Match
func matchIgnore(path string, patterns []string) (string, bool) {
	path = filepath.ToSlash(path)
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" || strings.HasPrefix(p, "#") {
			continue
		}
		p = filepath.ToSlash(p)

		// Leading '/' anchors the pattern to the repo root.
		anchored := false
		if strings.HasPrefix(p, "/") {
			anchored = true
			p = strings.TrimPrefix(p, "/")
		}

		// Directory match (trailing '/').
		if strings.HasSuffix(p, "/") {
			if strings.HasPrefix(path, p) {
				return "/" + p, true
			}
			if !anchored && strings.Contains(path, "/"+p) {
				return p, true
			}
			continue
		}

		// Full-path glob (contains slash) — always anchored to root.
		if strings.Contains(p, "/") {
			if ok, _ := filepath.Match(p, path); ok {
				return p, true
			}
			if strings.HasPrefix(path, p+"/") || path == p {
				return p, true
			}
			continue
		}

		// Single segment: anchored -> root only; otherwise basename anywhere.
		if anchored {
			if path == p || strings.HasPrefix(path, p+"/") {
				return "/" + p, true
			}
			if ok, _ := filepath.Match(p, path); ok && !strings.Contains(path, "/") {
				return "/" + p, true
			}
			continue
		}
		base := filepath.Base(path)
		if ok, _ := filepath.Match(p, base); ok {
			return p, true
		}
		if base == p {
			return p, true
		}
	}
	return "", false
}

// filterIgnored removes files matching patterns from the changeset and
// unstages them from git. Files present in preStaged (staged by the user
// before this run) are kept and committed normally even if they match an
// ignore pattern — the filter only prevents auto-adding unwanted files.
// Returns the list of removed paths.
func filterIgnored(cs *Changeset, patterns []string, preStaged map[string]bool) ([]string, error) {
	if len(patterns) == 0 || cs == nil {
		return nil, nil
	}
	kept := cs.Files[:0]
	var removed []string
	for _, f := range cs.Files {
		if _, hit := matchIgnore(f.Path, patterns); hit && !preStaged[f.Path] {
			removed = append(removed, f.Path)
			continue
		}
		kept = append(kept, f)
	}
	cs.Files = kept

	for _, p := range removed {
		if out, err := gitExec("restore", "--staged", "--", p); err != nil {
			return removed, fmt.Errorf("git restore --staged %s failed: %s: %w", p, out, err)
		}
	}
	return removed, nil
}

// stagedFiles returns the set of paths currently staged (index vs HEAD).
func stagedFiles() (map[string]bool, error) {
	out, err := gitRawOutput("diff", "--cached", "--name-only", "-z")
	if err != nil {
		return nil, err
	}
	set := make(map[string]bool)
	for _, p := range strings.Split(out, "\x00") {
		if p = strings.TrimSpace(p); p != "" {
			set[p] = true
		}
	}
	return set, nil
}
