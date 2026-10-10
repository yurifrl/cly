package gitcommits

import (
	"fmt"
	"path"
	"strings"
)

// seeds.go — deterministic affinity bundles for the jev planner.
//
// buildSeeds clusters the changeset into candidate bundles using cheap path
// heuristics (rename pairing, shared directory area). Jev then re-assigns
// each file onto one of these seeds (or votes "own"); the heuristic assignment
// is the default whenever jev is unavailable or unsure, so the planner always
// produces a valid plan. Seeds carry conventional type/scope so merged groups
// reuse the same consolidation keys as the chat planner.

// Seed is one heuristic affinity bundle.
type Seed struct {
	// ID is the option id jev sees ("b1", "b2", …).
	ID string
	// Files are the heuristic members. Jev may add or remove files.
	Files []FileChange
	// Type is the inferred conventional commit type (feat/fix/docs/...).
	Type string
	// Scope is the conventional commit scope derived from the area.
	Scope string
	// Label is the human description jev sees for this option.
	Label string
	// Area is the shared directory prefix the seed was built from.
	Area string
}

// buildSeeds clusters files into heuristic bundles. Renames arrive as single
// entries (git -M, Status R with OldPath) so they are already atomic and
// cluster like any other file. Files keep first-seen order so batch
// composition is deterministic.
func buildSeeds(files []FileChange) []Seed {
	// Area clustering. Area = directory prefix; "root" is its own area so
	// top-level files don't scatter.
	var order []string
	areas := make(map[string][]FileChange)
	for _, f := range files {
		area := fileArea(f)
		if _, seen := areas[area]; !seen {
			order = append(order, area)
		}
		areas[area] = append(areas[area], f)
	}

	seeds := make([]Seed, 0, len(order))
	for _, area := range order {
		files := areas[area]
		seeds = append(seeds, Seed{
			Files: files,
			Type:  inferSeedType(files),
			Scope: seedScope(area),
			Area:  area,
		})
	}

	for i := range seeds {
		seeds[i].ID = fmt.Sprintf("b%d", i+1)
		seeds[i].Label = seedLabel(seeds[i])
	}
	return seeds
}

// fileArea names the affinity area of one file: its directory ("root" for
// top-level files).
func fileArea(f FileChange) string {
	dir := path.Dir(f.Path)
	if dir == "." || dir == "" {
		return "root"
	}
	return dir
}

// seedScope converts an area into a conventional commit scope: the last path
// segment, dash-normalized ("modules/git-commits" → "git-commits").
func seedScope(area string) string {
	if area == "" || area == "." {
		return "root"
	}
	base := path.Base(area)
	return strings.ToLower(strings.ReplaceAll(base, "_", "-"))
}

// inferSeedType picks a conventional type from paths and statuses.
func inferSeedType(files []FileChange) string {
	allTest := true
	allDocs := true
	allBuild := true
	for _, f := range files {
		p := f.Path
		base := path.Base(p)
		isTest := strings.HasSuffix(base, "_test.go") ||
			strings.Contains(p, "/testdata/") ||
			strings.HasSuffix(base, ".test.js") ||
			strings.HasSuffix(base, ".test.ts") ||
			strings.HasSuffix(base, ".test.tsx") ||
			strings.HasSuffix(base, "_test.py") ||
			strings.Contains(base, ".spec.")
		isDocs := strings.HasSuffix(base, ".md") ||
			strings.HasSuffix(base, ".mdx") ||
			strings.Contains(p, "/docs/") ||
			strings.Contains(p, "/documentation/")
		isBuild := strings.Contains(p, "/.github/") ||
			strings.HasPrefix(p, ".github/") ||
			base == "Makefile" ||
			base == "Dockerfile" ||
			base == "docker-compose.yml" ||
			base == "docker-compose.yaml" ||
			strings.HasSuffix(base, ".mk") ||
			base == ".gitignore" ||
			base == ".gitattributes" ||
			base == ".editorconfig"
		if !isTest {
			allTest = false
		}
		if !isDocs {
			allDocs = false
		}
		if !isBuild {
			allBuild = false
		}
	}
	switch {
	case allTest:
		return "test"
	case allDocs:
		return "docs"
	case allBuild:
		return "chore"
	}
	for _, f := range files {
		if f.Status == StatusAdded {
			return "feat"
		}
	}
	return "fix"
}

// seedLabel renders the option description jev sees for a seed.
func seedLabel(s Seed) string {
	const maxNames = 4
	names := make([]string, 0, len(s.Files))
	for i, f := range s.Files {
		if i == maxNames {
			names = append(names, fmt.Sprintf("+%d more", len(s.Files)-maxNames))
			break
		}
		status := "M"
		switch f.Status {
		case StatusAdded:
			status = "A"
		case StatusDeleted:
			status = "D"
		case StatusRenamed:
			status = "R"
		}
		names = append(names, fmt.Sprintf("%s(%s)", path.Base(f.Path), status))
	}
	return fmt.Sprintf("%s changes in %s: %s", s.Type, s.Area, strings.Join(names, ", "))
}
