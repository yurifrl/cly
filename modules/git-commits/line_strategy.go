package gitcommits

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// lineSystemPrompt instructs the LLM to group changes at the hunk level.
// A single file's hunks can be split across different commit groups.
const lineSystemPrompt = `You are a git commit planning assistant. Your job is to split changes into the smallest sensible set of logical, atomic commits at the LINE level.

Unlike file-level splitting, a single file CAN have its hunks split across different commits. Group hunks by logical concern: a feature, a bug fix, a refactor, a config change, etc.

GROUPING RULES (priority order):
1. Hunks that implement the same feature/fix = one group (even across different files)
2. Hunks that modify the same function or struct = one group
3. Import changes go with the hunks that need them
4. Test hunks go with the implementation hunks they test
5. Different concerns (feat vs fix vs chore) in the same file = separate groups

HARD CONSTRAINTS:
- Every hunk should appear in exactly one group (unless additional instructions say otherwise)
- Reference hunks by file path and hunk ID (e.g., h1, h2)
- Each group gets a conventional commit message (feat:, fix:, chore:, refactor:, docs:, test:, style:, build:, ci:, perf:)
- Files with only one hunk: just list the file (all hunks implied)
- Prefer fewer groups (2-5) over many tiny ones

OUTPUT FORMAT: Respond with ONLY a JSON object (no markdown fences, no explanation):
{
  "groups": [
    {
      "title": "feat: add session management",
      "type": "feat",
      "summary": "Brief explanation of what this group covers",
      "items": [
        { "file": "src/session.go", "hunks": ["h1", "h3"] },
        { "file": "src/config.go", "hunks": ["h2"] },
        { "file": "src/session_test.go" }
      ]
    }
  ]
}

When "hunks" is omitted or empty, ALL hunks of that file are included.`

// BuildLineBatches creates batches with hunk-level detail for the line strategy.
func BuildLineBatches(cs *Changeset, batchSize int) []Batch {
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}

	type fileText struct {
		file FileChange
		text string
		size int
	}

	var items []fileText
	for _, f := range cs.Files {
		text := buildLineAnalysis(f)
		items = append(items, fileText{
			file: f,
			text: text,
			size: len(text),
		})
	}

	// Same greedy packing as file strategy
	var batches []Batch
	var currentFiles []FileChange
	var currentTexts []string
	currentSize := 0

	for _, item := range items {
		if currentSize+item.size > batchSize && len(currentFiles) > 0 {
			batches = append(batches, makeBatch(len(batches), currentFiles, currentTexts))
			currentFiles = nil
			currentTexts = nil
			currentSize = 0
		}
		currentFiles = append(currentFiles, item.file)
		currentTexts = append(currentTexts, item.text)
		currentSize += item.size
	}
	if len(currentFiles) > 0 {
		batches = append(batches, makeBatch(len(batches), currentFiles, currentTexts))
	}
	total := len(batches)
	for i := range batches {
		batches[i].TotalCount = total
	}
	return batches
}

// buildLineAnalysis generates analysis text with per-hunk content visible.
func buildLineAnalysis(f FileChange) string {
	var b strings.Builder

	status := statusLabel(f.Status)
	b.WriteString(fmt.Sprintf("File: %s [%s]", f.Path, status))
	if f.OldPath != "" {
		b.WriteString(fmt.Sprintf(" (renamed from %s)", f.OldPath))
	}
	b.WriteString("\n")

	isBinary := strings.Contains(f.Diff, "GIT binary patch") ||
		strings.Contains(f.Diff, "Binary files")

	if isBinary {
		b.WriteString("  (binary file)\n")
		return b.String()
	}

	// Show each hunk with its content so the LLM can reason about lines
	for _, h := range f.Hunks {
		b.WriteString(fmt.Sprintf("\n  Hunk %s: %s\n", h.ID, h.RangeLabel))
		// Include the hunk content (trimmed to 4K per hunk to stay within budget)
		content := h.Content
		if len(content) > 4000 {
			content = content[:4000] + "\n  ... (truncated)\n"
		}
		b.WriteString(content)
	}

	return b.String()
}

func statusLabel(s FileStatus) string {
	switch s {
	case StatusAdded:
		return "ADDED"
	case StatusModified:
		return "MODIFIED"
	case StatusDeleted:
		return "DELETED"
	case StatusRenamed:
		return "RENAMED"
	default:
		return string(s)
	}
}

// ValidateLinePlan validates a raw plan that may reference specific hunks.
func ValidateLinePlan(raw *RawPlan, cs *Changeset) (*CommitPlan, error) {
	if raw == nil || len(raw.Groups) == 0 {
		return nil, fmt.Errorf("empty plan")
	}

	// Build lookup: path → FileChange, path → hunkID → Hunk
	byPath := make(map[string]*FileChange)
	byOldPath := make(map[string]*FileChange)
	for i := range cs.Files {
		f := &cs.Files[i]
		byPath[f.Path] = f
		if f.OldPath != "" {
			byOldPath[f.OldPath] = f
		}
	}

	// Track which hunks have been assigned: "path:hunkID" → true
	assignedHunks := make(map[string]bool)
	var groups []CommitGroup

	for _, rg := range raw.Groups {
		var files []CommitFile

		for _, item := range rg.Items {
			fc := resolveFile(item.File, byPath, byOldPath)
			if fc == nil {
				continue // LLM hallucination
			}

			// Determine which hunks this item covers
			var hunkIDs []string
			if len(item.Hunks) == 0 {
				// All hunks
				for _, h := range fc.Hunks {
					hunkIDs = append(hunkIDs, h.ID)
				}
			} else {
				hunkIDs = item.Hunks
			}

			// Deduplicate: skip hunks already assigned
			var validHunks []string
			for _, hid := range hunkIDs {
				key := fc.Path + ":" + hid
				if assignedHunks[key] {
					continue
				}
				// Verify hunk exists
				if findHunk(fc, hid) != nil {
					assignedHunks[key] = true
					validHunks = append(validHunks, hid)
				}
			}

			if len(validHunks) == 0 {
				continue
			}

			wholeFile := len(validHunks) == len(fc.Hunks)
			files = append(files, CommitFile{
				Path:      fc.Path,
				OldPath:   fc.OldPath,
				Status:    fc.Status,
				WholeFile: wholeFile,
				HunkIDs:   validHunks,
			})
		}

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

	// Auto-heal: assign uncovered hunks
	for i := range cs.Files {
		f := &cs.Files[i]
		var uncovered []string
		for _, h := range f.Hunks {
			key := f.Path + ":" + h.ID
			if !assignedHunks[key] {
				uncovered = append(uncovered, h.ID)
				assignedHunks[key] = true
			}
		}
		if len(uncovered) == 0 {
			continue
		}

		bestGroup := findBestGroup(f.Path, groups)
		if bestGroup < 0 && len(groups) > 0 {
			bestGroup = 0
		}
		if bestGroup >= 0 {
			// Check if file already exists in this group — append hunks
			found := false
			for fi := range groups[bestGroup].Files {
				if groups[bestGroup].Files[fi].Path == f.Path {
					groups[bestGroup].Files[fi].HunkIDs = append(groups[bestGroup].Files[fi].HunkIDs, uncovered...)
					groups[bestGroup].Files[fi].WholeFile = len(groups[bestGroup].Files[fi].HunkIDs) == len(f.Hunks)
					found = true
					break
				}
			}
			if !found {
				wholeFile := len(uncovered) == len(f.Hunks)
				groups[bestGroup].Files = append(groups[bestGroup].Files, CommitFile{
					Path:      f.Path,
					OldPath:   f.OldPath,
					Status:    f.Status,
					WholeFile: wholeFile,
					HunkIDs:   uncovered,
				})
			}
		}
	}

	// Handle files with no hunks (binary, new empty, deleted)
	for i := range cs.Files {
		f := &cs.Files[i]
		if len(f.Hunks) > 0 {
			continue // Already handled above
		}
		// Check if any group already has this file
		covered := false
		for _, g := range groups {
			for _, gf := range g.Files {
				if gf.Path == f.Path {
					covered = true
					break
				}
			}
			if covered {
				break
			}
		}
		if covered {
			continue
		}
		// Assign to best group
		bestGroup := findBestGroup(f.Path, groups)
		if bestGroup < 0 && len(groups) > 0 {
			bestGroup = 0
		}
		if bestGroup >= 0 {
			groups[bestGroup].Files = append(groups[bestGroup].Files, CommitFile{
				Path:      f.Path,
				OldPath:   f.OldPath,
				Status:    f.Status,
				WholeFile: true,
			})
		}
	}

	if len(groups) == 0 {
		return nil, fmt.Errorf("empty plan after healing")
	}

	return &CommitPlan{Groups: groups}, nil
}

func findHunk(fc *FileChange, id string) *Hunk {
	for i := range fc.Hunks {
		if fc.Hunks[i].ID == id {
			return &fc.Hunks[i]
		}
	}
	return nil
}

// ExecuteLine executes the plan using patch-based staging for hunk-level precision.
func ExecuteLine(plan *CommitPlan, cs *Changeset, noVerify bool) ([]CommitResult, error) {
	if _, err := gitExec("rev-parse", "HEAD"); err != nil {
		return nil, fmt.Errorf("no initial commit found — create one first")
	}

	originalHead, err := gitOutput("rev-parse", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	savedDiff, err := gitRawOutput("diff", "--cached", "--binary")
	if err != nil {
		return nil, fmt.Errorf("failed to save diff: %w", err)
	}
	if savedDiff != "" && !strings.HasSuffix(savedDiff, "\n") {
		savedDiff += "\n"
	}

	// Build file lookup for constructing patches
	byPath := make(map[string]*FileChange)
	for i := range cs.Files {
		byPath[cs.Files[i].Path] = &cs.Files[i]
	}

	if _, err := gitExec("reset"); err != nil {
		return nil, fmt.Errorf("git reset failed: %w", err)
	}

	var results []CommitResult

	for i, group := range plan.Groups {
		err := executeLineGroup(group, byPath, noVerify)
		if err != nil {
			rollbackErr := rollback(originalHead, savedDiff)
			if rollbackErr != nil {
				return results, fmt.Errorf("commit %d failed: %w\nROLLBACK ALSO FAILED: %v\nManual recovery: git reset --soft %s",
					i+1, err, rollbackErr, originalHead)
			}
			return results, fmt.Errorf("commit %d failed (rolled back): %w", i+1, err)
		}

		sha, _ := gitOutput("rev-parse", "--short", "HEAD")
		results = append(results, CommitResult{
			Title: group.Title,
			SHA:   sha,
			Files: len(group.Files),
		})
	}

	return results, nil
}

func executeLineGroup(group CommitGroup, byPath map[string]*FileChange, noVerify bool) error {
	for _, f := range group.Files {
		switch {
		case f.Status == StatusDeleted:
			if out, err := gitExec("rm", "--cached", "--", f.Path); err != nil {
				return fmt.Errorf("git rm %q: %s: %w", f.Path, strings.TrimSpace(out), err)
			}

		case f.Status == StatusRenamed && f.WholeFile:
			if f.OldPath != "" {
				if out, err := gitExec("rm", "--cached", "--", f.OldPath); err != nil {
					return fmt.Errorf("git rm %q: %s: %w", f.OldPath, strings.TrimSpace(out), err)
				}
			}
			if out, err := gitExec("add", "--", f.Path); err != nil {
				return fmt.Errorf("git add %q: %s: %w", f.Path, strings.TrimSpace(out), err)
			}

		case f.WholeFile:
			// Whole file — simple git add
			if out, err := gitExec("add", "--", f.Path); err != nil {
				return fmt.Errorf("git add %q: %s: %w", f.Path, strings.TrimSpace(out), err)
			}

		default:
			// Partial file — build a patch from specific hunks and apply
			fc := byPath[f.Path]
			if fc == nil {
				return fmt.Errorf("file %q not found in changeset", f.Path)
			}

			patch := buildHunkPatch(fc, f.HunkIDs)
			if patch == "" {
				continue
			}

			if err := applyPatch(patch); err != nil {
				return fmt.Errorf("apply patch for %q hunks %v: %w", f.Path, f.HunkIDs, err)
			}
		}
	}

	// Commit
	args := []string{"commit", "-m", group.Title}
	if group.Body != "" {
		args = append(args, "-m", group.Body)
	}
	if noVerify {
		args = append(args, "--no-verify")
	}

	if out, err := gitExec(args...); err != nil {
		return fmt.Errorf("git commit failed: %s: %w", strings.TrimSpace(out), err)
	}

	return nil
}

// buildHunkPatch constructs a valid unified diff patch from specific hunks of a file.
func buildHunkPatch(fc *FileChange, hunkIDs []string) string {
	wantHunks := make(map[string]bool)
	for _, id := range hunkIDs {
		wantHunks[id] = true
	}

	// Collect matching hunks
	var hunks []Hunk
	for _, h := range fc.Hunks {
		if wantHunks[h.ID] {
			hunks = append(hunks, h)
		}
	}

	if len(hunks) == 0 {
		return ""
	}

	var b strings.Builder

	// Write diff header from the original diff
	// Extract lines before the first @@ from fc.Diff
	lines := strings.Split(fc.Diff, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "@@ ") {
			break
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Write selected hunks
	for _, h := range hunks {
		b.WriteString(h.Content)
	}

	result := b.String()
	if !strings.HasSuffix(result, "\n") {
		result += "\n"
	}

	return result
}

// applyPatch writes a patch to a temp file and applies it to the index.
func applyPatch(patch string) error {
	tmpFile, err := os.CreateTemp("", "git-commits-hunk-*.patch")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(patch); err != nil {
		tmpFile.Close()
		return fmt.Errorf("write patch: %w", err)
	}
	tmpFile.Close()

	cmd := exec.Command("git", "apply", "--cached", tmpFile.Name())
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(out)), err)
	}

	return nil
}
