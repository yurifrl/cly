package gitcommits

import (
	"fmt"
	"path/filepath"
	"strings"
)

// CommitPlan is the validated, execution-ready plan.
type CommitPlan struct {
	Groups []CommitGroup
}

// CommitGroup is a single commit to be created.
type CommitGroup struct {
	Title   string
	Type    string
	Summary string
	Body    string
	Files   []CommitFile
}

// CommitFile is a file in a commit group.
type CommitFile struct {
	Path      string
	OldPath   string
	Status    FileStatus
	WholeFile bool
	HunkIDs   []string // Non-empty for line strategy partial-file commits
}

// ValidatePlan validates a raw LLM plan against the changeset and auto-heals issues.
func ValidatePlan(raw *RawPlan, cs *Changeset, skipAutoHeal bool) (*CommitPlan, error) {
	if raw == nil || len(raw.Groups) == 0 {
		return nil, fmt.Errorf("empty plan")
	}

	// Build lookup maps from changeset
	byPath := make(map[string]*FileChange)
	byOldPath := make(map[string]*FileChange)
	for i := range cs.Files {
		f := &cs.Files[i]
		byPath[f.Path] = f
		if f.OldPath != "" {
			byOldPath[f.OldPath] = f
		}
	}

	seen := make(map[string]bool)
	var groups []CommitGroup

	for _, rg := range raw.Groups {
		var files []CommitFile

		for _, item := range rg.Items {
			path := item.File

			// Already assigned to an earlier group — deduplicate
			if seen[path] {
				continue
			}

			// Resolve file reference
			fc := resolveFile(path, byPath, byOldPath)
			if fc == nil {
				// Unknown file — skip silently (LLM hallucination)
				continue
			}

			seen[fc.Path] = fc.Path != ""
			if fc.OldPath != "" {
				seen[fc.OldPath] = true
			}

			files = append(files, CommitFile{
				Path:      fc.Path,
				OldPath:   fc.OldPath,
				Status:    fc.Status,
				WholeFile: true,
			})
		}

		// Drop empty groups
		if len(files) == 0 {
			continue
		}

		groups = append(groups, CommitGroup{
			Title:   rg.Title,
			Type:    rg.Type,
			Summary: rg.Summary,
			Body:    rg.Body,
			Files:   files,
		})
	}

	// Auto-heal: assign uncovered files (skip when user provided custom prompt)
	if !skipAutoHeal {
	for i := range cs.Files {
		f := &cs.Files[i]
		if seen[f.Path] {
			continue
		}

		// Find group with longest shared directory prefix
		bestGroup := findBestGroup(f.Path, groups)
		if bestGroup < 0 && len(groups) > 0 {
			bestGroup = 0 // Default to first group
		}

		if bestGroup >= 0 {
			groups[bestGroup].Files = append(groups[bestGroup].Files, CommitFile{
				Path:      f.Path,
				OldPath:   f.OldPath,
				Status:    f.Status,
				WholeFile: true,
			})
			seen[f.Path] = true
		}
	}
	}

	if len(groups) == 0 {
		return nil, fmt.Errorf("empty plan after healing")
	}

	return &CommitPlan{Groups: groups}, nil
}

// resolveFile matches a file reference to a changeset entry.
func resolveFile(path string, byPath, byOldPath map[string]*FileChange) *FileChange {
	if fc, ok := byPath[path]; ok {
		return fc
	}
	if fc, ok := byOldPath[path]; ok {
		return fc
	}
	// Try without leading slash
	path = strings.TrimPrefix(path, "/")
	if fc, ok := byPath[path]; ok {
		return fc
	}
	return nil
}

// findBestGroup finds the group whose files share the longest directory prefix with path.
func findBestGroup(path string, groups []CommitGroup) int {
	bestIdx := -1
	bestLen := 0
	pathDir := filepath.Dir(path)

	for gi, g := range groups {
		for _, f := range g.Files {
			fDir := filepath.Dir(f.Path)
			shared := sharedPrefix(pathDir, fDir)
			if len(shared) > bestLen {
				bestLen = len(shared)
				bestIdx = gi
			}
		}
	}

	return bestIdx
}

// sharedPrefix returns the common directory prefix between two paths.
func sharedPrefix(a, b string) string {
	aParts := strings.Split(a, "/")
	bParts := strings.Split(b, "/")

	var shared []string
	for i := 0; i < len(aParts) && i < len(bParts); i++ {
		if aParts[i] == bParts[i] {
			shared = append(shared, aParts[i])
		} else {
			break
		}
	}

	return strings.Join(shared, "/")
}
