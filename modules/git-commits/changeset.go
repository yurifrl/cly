package gitcommits

import (
	"fmt"
	"os/exec"
	"strings"
)

// FileStatus represents the type of change to a file.
type FileStatus string

const (
	StatusAdded    FileStatus = "A"
	StatusModified FileStatus = "M"
	StatusDeleted  FileStatus = "D"
	StatusRenamed  FileStatus = "R"
)

// Hunk represents a change block within a file diff.
type Hunk struct {
	ID         string
	OldStart   int
	NewStart   int
	OldLines   int
	NewLines   int
	RangeLabel string
	Content    string
}

// FileChange represents a single file's changes in the changeset.
type FileChange struct {
	Path    string
	OldPath string // Only set for renames
	Status  FileStatus
	Hunks   []Hunk
	Diff    string // Raw diff text for this file
}

// Changeset represents the full set of changes to be committed.
type Changeset struct {
	Files []FileChange
}

// GetChangeset collects the current git changes.
// If all is true, stages all files (including untracked) first.
func GetChangeset(all bool) (*Changeset, error) {
	// Verify git is available
	if _, err := exec.LookPath("git"); err != nil {
		return nil, fmt.Errorf("git not found in PATH: %w", err)
	}

	if all {
		// Stage everything including untracked
		if out, err := gitExec("add", "."); err != nil {
			return nil, fmt.Errorf("git add failed: %s: %w", out, err)
		}
	}

	// Get staged diff with rename detection and binary support
	diffOutput, err := gitRawOutput("-c", "diff.submodule=short", "diff", "--cached", "-M", "--no-color", "--binary")
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w", err)
	}

	if strings.TrimSpace(diffOutput) == "" {
		return nil, fmt.Errorf("no staged changes found")
	}

	cs := ParseDiff(diffOutput)
	if len(cs.Files) == 0 {
		return nil, fmt.Errorf("no staged changes found")
	}

	return cs, nil
}

// ParseDiff parses unified diff output into a Changeset.
func ParseDiff(diffOutput string) *Changeset {
	cs := &Changeset{}
	if strings.TrimSpace(diffOutput) == "" {
		return cs
	}

	// Split on "diff --git" boundaries
	sections := splitDiffSections(diffOutput)

	for _, section := range sections {
		fc := parseFileSection(section)
		if fc != nil {
			cs.Files = append(cs.Files, *fc)
		}
	}

	return cs
}

// splitDiffSections splits raw diff output into per-file sections.
// Only splits on "diff --git " at the start of a line to avoid false splits
// inside binary patch data.
func splitDiffSections(diffOutput string) []string {
	var sections []string
	var current strings.Builder
	prefix := "diff --git "

	for _, line := range strings.Split(diffOutput, "\n") {
		if strings.HasPrefix(line, prefix) {
			// Flush previous section
			if current.Len() > 0 {
				sections = append(sections, current.String())
				current.Reset()
			}
		}
		if current.Len() > 0 || strings.HasPrefix(line, prefix) {
			current.WriteString(line)
			current.WriteString("\n")
		}
	}

	// Flush last section
	if current.Len() > 0 {
		sections = append(sections, current.String())
	}

	return sections
}

// parseFileSection parses a single file's diff section.
func parseFileSection(section string) *FileChange {
	lines := strings.Split(section, "\n")
	if len(lines) == 0 {
		return nil
	}

	fc := &FileChange{
		Diff: section,
	}

	// Parse the "diff --git a/X b/Y" header
	// Git may quote paths with special characters: "diff --git \"a/f\" \"b/f\""
	if strings.HasPrefix(lines[0], "diff --git ") {
		fc.Path = parseDiffGitPath(lines[0])
	}

	// Scan headers for status indicators
	var hunkID int
	for i, line := range lines {
		switch {
		case strings.HasPrefix(line, "new file mode"):
			fc.Status = StatusAdded
		case strings.HasPrefix(line, "deleted file mode"):
			fc.Status = StatusDeleted
		case strings.HasPrefix(line, "rename from "):
			fc.OldPath = strings.TrimPrefix(line, "rename from ")
			fc.Status = StatusRenamed
		case strings.HasPrefix(line, "rename to "):
			fc.Path = strings.TrimPrefix(line, "rename to ")
		case strings.HasPrefix(line, "--- a/"):
			if fc.Status == "" {
				fc.Status = StatusModified
			}
		case strings.HasPrefix(line, "+++ b/"):
			if fc.Path == "" {
				fc.Path = strings.TrimPrefix(line, "+++ b/")
			}
		case strings.HasPrefix(line, "+++ \"b/"):
			if fc.Path == "" {
				p := strings.TrimPrefix(line, "+++ \"b/")
				p = strings.TrimSuffix(p, "\"")
				fc.Path = unquoteGitPath(p)
			}
		case strings.HasPrefix(line, "@@ "):
			hunkID++
			hunk := parseHunkHeader(line, hunkID)
			// Collect hunk content
			var content strings.Builder
			content.WriteString(line)
			content.WriteString("\n")
			for j := i + 1; j < len(lines); j++ {
				if strings.HasPrefix(lines[j], "@@ ") || strings.HasPrefix(lines[j], "diff --git ") {
					break
				}
				content.WriteString(lines[j])
				content.WriteString("\n")
			}
			hunk.Content = content.String()
			fc.Hunks = append(fc.Hunks, hunk)
		}
	}

	// Default to modified if we have a path but no status
	if fc.Status == "" && fc.Path != "" {
		fc.Status = StatusModified
	}

	if fc.Path == "" {
		return nil
	}

	return fc
}

// parseDiffGitPath extracts the destination path from a "diff --git a/X b/Y" line.
// Handles quoted paths: diff --git "a/has spaces" "b/has spaces"
func parseDiffGitPath(line string) string {
	// Remove "diff --git " prefix
	rest := strings.TrimPrefix(line, "diff --git ")

	// Check for quoted paths: "a/..." "b/..."
	if strings.HasPrefix(rest, "\"") {
		// Find the second quoted path
		// Format: "a/path" "b/path"
		parts := strings.SplitN(rest, "\" \"", 2)
		if len(parts) == 2 {
			p := strings.TrimSuffix(parts[1], "\"")
			p = strings.TrimPrefix(p, "b/")
			return unquoteGitPath(p)
		}
	}

	// Unquoted: a/X b/Y — split from the right since paths can contain " b/"
	// Strategy: the a-path and b-path are the same length for non-renames,
	// so find the midpoint. For renames, fall through to +++ b/ parsing.
	// Simplest reliable approach: take the last " b/" occurrence
	idx := strings.LastIndex(rest, " b/")
	if idx >= 0 {
		return rest[idx+3:]
	}

	return ""
}

// unquoteGitPath handles git's C-style quoting (e.g., \t, \n, \\, octal escapes).
func unquoteGitPath(s string) string {
	// Simple common cases
	s = strings.ReplaceAll(s, "\\\\", "\\")
	s = strings.ReplaceAll(s, "\\t", "\t")
	s = strings.ReplaceAll(s, "\\n", "\n")
	s = strings.ReplaceAll(s, "\\\"", "\"")
	return s
}

// parseHunkHeader parses "@@ -old,lines +new,lines @@" format.
func parseHunkHeader(line string, id int) Hunk {
	h := Hunk{
		ID: fmt.Sprintf("h%d", id),
	}

	// Parse @@ -old,lines +new,lines @@ ...
	parts := strings.SplitN(line, "@@", 3)
	if len(parts) < 2 {
		return h
	}

	rangePart := strings.TrimSpace(parts[1])
	ranges := strings.Fields(rangePart)

	for _, r := range ranges {
		if strings.HasPrefix(r, "-") {
			r = strings.TrimPrefix(r, "-")
			parseRange(r, &h.OldStart, &h.OldLines)
		} else if strings.HasPrefix(r, "+") {
			r = strings.TrimPrefix(r, "+")
			parseRange(r, &h.NewStart, &h.NewLines)
		}
	}

	h.RangeLabel = fmt.Sprintf("changed %d-%d", h.NewStart, h.NewStart+h.NewLines)

	return h
}

func parseRange(s string, start, lines *int) {
	parts := strings.SplitN(s, ",", 2)
	fmt.Sscanf(parts[0], "%d", start)
	if len(parts) == 2 {
		fmt.Sscanf(parts[1], "%d", lines)
	} else {
		*lines = 1
	}
}

// repoRoot returns the absolute path of the git repository root.
// Falls back to the current directory if git is unavailable.
func repoRoot() string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "."
	}
	return strings.TrimSpace(string(out))
}

// gitExec runs a git command from the repo root and returns stdout+stderr.
func gitExec(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot()
	out, err := cmd.CombinedOutput()
	return string(out), err
}
