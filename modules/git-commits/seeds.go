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
// each file onto one of these seeds (or votes "own"); a seed left unmerged
// (or voted "own") stands as its own commit. Seeds carry conventional
// type/scope so merged groups reuse the same consolidation keys as the chat
// planner. When every jev provider fails the planner errors out — no
// heuristic-only plan exists.

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
	var seeds []Seed

	// Uniform moves get a single seed up front: when a batch of files leaves
	// one top-level directory for another (as rename entries or as
	// delete+add pairs, e.g. all of .claude/** replaced by .agents/**), they
	// are one logical move no matter how many directories they span.
	// Per-directory seeding here produced one commit per moved file. jev
	// still decides whether this seed merges with anything else.
	if src, dst, migration := moveMigration(files); migration != nil {
		var parts []string
		var renamed, deleted, added int
		for _, m := range migration {
			switch m.Status {
			case StatusRenamed:
				renamed++
			case StatusDeleted:
				deleted++
			case StatusAdded:
				added++
			}
		}
		if renamed > 0 {
			parts = append(parts, fmt.Sprintf("%d renamed", renamed))
		}
		if deleted > 0 {
			parts = append(parts, fmt.Sprintf("%d deleted", deleted))
		}
		if added > 0 {
			parts = append(parts, fmt.Sprintf("%d added", added))
		}
		s := Seed{
			Files: migration,
			Type:  "chore",
			Scope: seedScope(dst),
			Area:  src,
		}
		s.Label = fmt.Sprintf("one migration: move %s/** to %s/** (%s, e.g. %s → %s)",
			src, dst, strings.Join(parts, ", "), migration[0].OldPath, migration[0].Path)
		if migration[0].OldPath == "" {
			s.Label = fmt.Sprintf("one migration: replace %s/** with %s/** (%s)",
				src, dst, strings.Join(parts, ", "))
		}
		seeds = append(seeds, s)
		files = withoutFiles(files, migration)
	}

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

	for _, area := range order {
		fs := areas[area]
		seeds = append(seeds, Seed{
			Files: fs,
			Type:  inferSeedType(fs),
			Scope: seedScope(area),
			Area:  area,
		})
	}

	for i := range seeds {
		seeds[i].ID = fmt.Sprintf("b%d", i+1)
		if seeds[i].Label == "" {
			seeds[i].Label = seedLabel(seeds[i])
		}
	}
	return seeds
}

// moveMigration reports when the changeset contains one dominant uniform
// move: a batch of files leaving one top-level directory for another,
// expressed either as git rename entries or as separate deletion+addition
// pairs whose relative paths mirror each other. Git only detects renames
// when contents stay similar; a move that rewrites the files (e.g. a skill
// migration) drops below the similarity threshold and surfaces as D+A, so
// both shapes must count. Returns the source and destination prefixes and
// the member files (renames as-is; pairs contribute both sides; unpaired
// deletions under src join as part of deprecating it), or nil when no move
// dominates.
func moveMigration(files []FileChange) (src, dst string, members []FileChange) {
	type split struct{ src, dst string }
	renameBySplit := make(map[split][]FileChange)
	pairsBySplit := make(map[split][][2]FileChange)
	addsByRel := make(map[string][]FileChange)
	var dels []FileChange
	for _, f := range files {
		switch f.Status {
		case StatusRenamed:
			if f.OldPath == "" {
				continue
			}
			s := split{pathTop(f.OldPath), pathTop(f.Path)}
			renameBySplit[s] = append(renameBySplit[s], f)
		case StatusDeleted:
			dels = append(dels, f)
		case StatusAdded:
			addsByRel[relAfterTop(f.Path)] = append(addsByRel[relAfterTop(f.Path)], f)
		}
	}

	tally := make(map[split]int)
	for s, fs := range renameBySplit {
		tally[s] += len(fs)
	}
	for _, d := range dels {
		dTop, rel := pathTop(d.Path), relAfterTop(d.Path)
		match, candidates := FileChange{}, 0
		for _, a := range addsByRel[rel] {
			if pathTop(a.Path) != dTop {
				candidates++
				match = a
			}
		}
		if candidates != 1 {
			continue // ambiguous or same-dir: not a recognizable move pair
		}
		s := split{dTop, pathTop(match.Path)}
		pairsBySplit[s] = append(pairsBySplit[s], [2]FileChange{d, match})
		tally[s]++
	}

	var best split
	bestCount := 0
	for s, n := range tally {
		if n > bestCount {
			best, bestCount = s, n
		}
	}
	if bestCount < 2 {
		return "", "", nil
	}
	src, dst = best.src, best.dst

	members = append(members, renameBySplit[best]...)
	if pairs, ok := pairsBySplit[best]; ok {
		paired := make(map[string]bool, len(pairs))
		for _, p := range pairs {
			members = append(members, p[0], p[1])
			paired[p[0].Path] = true
		}
		// Deletions under src with no counterpart belong to the move too:
		// deprecating a directory includes files dropped outright.
		for _, d := range dels {
			if pathTop(d.Path) == src && !paired[d.Path] {
				members = append(members, d)
			}
		}
	}
	if len(members) < 2 {
		return "", "", nil
	}
	return src, dst, members
}

// pathTop is the first path segment of a repo path, or "" at the root.
func pathTop(p string) string {
	if i := strings.IndexByte(p, '/'); i >= 0 {
		return p[:i]
	}
	return ""
}

// relAfterTop strips the first path segment (e.g. .claude/skills/x →
// skills/x) so delete+add pairs of a move compare equal; root files compare
// by full name.
func relAfterTop(p string) string {
	if i := strings.IndexByte(p, '/'); i >= 0 {
		return p[i+1:]
	}
	return p
}

// withoutFiles returns files minus the dropped entries, keeping order.
func withoutFiles(files, drop []FileChange) []FileChange {
	dropped := make(map[string]bool, len(drop))
	for _, f := range drop {
		dropped[f.Path] = true
	}
	rest := make([]FileChange, 0, len(files))
	for _, f := range files {
		if !dropped[f.Path] {
			rest = append(rest, f)
		}
	}
	return rest
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
