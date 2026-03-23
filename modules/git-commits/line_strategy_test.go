package gitcommits

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildLineBatches(t *testing.T) {
	cs := &Changeset{
		Files: []FileChange{
			{
				Path:   "main.go",
				Status: StatusModified,
				Hunks: []Hunk{
					{ID: "h1", RangeLabel: "changed 1-5", Content: "@@ -1,3 +1,5 @@\n+line1\n+line2\n"},
					{ID: "h2", RangeLabel: "changed 20-25", Content: "@@ -20,3 +20,5 @@\n+line3\n"},
				},
				Diff: "diff --git a/main.go b/main.go\n--- a/main.go\n+++ b/main.go\n",
			},
		},
	}

	batches := BuildLineBatches(cs, 40000)
	require.Len(t, batches, 1)
	// Should include hunk IDs and content
	assert.Contains(t, batches[0].Text, "Hunk h1")
	assert.Contains(t, batches[0].Text, "Hunk h2")
	assert.Contains(t, batches[0].Text, "+line1")
}

func TestValidateLinePlan_HunkLevel(t *testing.T) {
	cs := &Changeset{
		Files: []FileChange{
			{
				Path:   "main.go",
				Status: StatusModified,
				Hunks: []Hunk{
					{ID: "h1", Content: "@@ -1,3 +1,5 @@\n+feat\n"},
					{ID: "h2", Content: "@@ -20,3 +20,5 @@\n+fix\n"},
				},
			},
			{
				Path:   "test.go",
				Status: StatusAdded,
				Hunks: []Hunk{
					{ID: "h1", Content: "@@ -0,0 +1,10 @@\n+test\n"},
				},
			},
		},
	}

	raw := &RawPlan{
		Groups: []RawGroup{
			{
				Title:   "feat: add feature",
				Type:    "feat",
				Summary: "feature work",
				Items: []RawItem{
					{File: "main.go", Hunks: []string{"h1"}},
					{File: "test.go"}, // All hunks
				},
			},
			{
				Title:   "fix: bug fix",
				Type:    "fix",
				Summary: "bug fix",
				Items: []RawItem{
					{File: "main.go", Hunks: []string{"h2"}},
				},
			},
		},
	}

	plan, err := ValidateLinePlan(raw, cs)
	require.NoError(t, err)
	require.Len(t, plan.Groups, 2)

	// First group: main.go h1 (partial) + test.go (whole)
	assert.Len(t, plan.Groups[0].Files, 2)
	assert.Equal(t, "main.go", plan.Groups[0].Files[0].Path)
	assert.Equal(t, []string{"h1"}, plan.Groups[0].Files[0].HunkIDs)
	assert.False(t, plan.Groups[0].Files[0].WholeFile)
	assert.Equal(t, "test.go", plan.Groups[0].Files[1].Path)
	assert.True(t, plan.Groups[0].Files[1].WholeFile)

	// Second group: main.go h2 (partial)
	assert.Len(t, plan.Groups[1].Files, 1)
	assert.Equal(t, "main.go", plan.Groups[1].Files[0].Path)
	assert.Equal(t, []string{"h2"}, plan.Groups[1].Files[0].HunkIDs)
	assert.False(t, plan.Groups[1].Files[0].WholeFile)
}

func TestValidateLinePlan_DedupHunks(t *testing.T) {
	cs := &Changeset{
		Files: []FileChange{
			{
				Path:   "a.go",
				Status: StatusModified,
				Hunks:  []Hunk{{ID: "h1"}, {ID: "h2"}},
			},
		},
	}

	raw := &RawPlan{
		Groups: []RawGroup{
			{Title: "g1", Items: []RawItem{{File: "a.go", Hunks: []string{"h1"}}}},
			{Title: "g2", Items: []RawItem{{File: "a.go", Hunks: []string{"h1", "h2"}}}}, // h1 duplicate
		},
	}

	plan, err := ValidateLinePlan(raw, cs)
	require.NoError(t, err)
	require.Len(t, plan.Groups, 2)

	// g1 gets h1
	assert.Equal(t, []string{"h1"}, plan.Groups[0].Files[0].HunkIDs)
	// g2 gets only h2 (h1 was deduped)
	assert.Equal(t, []string{"h2"}, plan.Groups[1].Files[0].HunkIDs)
}

func TestValidateLinePlan_UncoveredHunks(t *testing.T) {
	cs := &Changeset{
		Files: []FileChange{
			{
				Path:   "a.go",
				Status: StatusModified,
				Hunks:  []Hunk{{ID: "h1"}, {ID: "h2"}, {ID: "h3"}},
			},
		},
	}

	// Plan only covers h1
	raw := &RawPlan{
		Groups: []RawGroup{
			{Title: "g1", Items: []RawItem{{File: "a.go", Hunks: []string{"h1"}}}},
		},
	}

	plan, err := ValidateLinePlan(raw, cs)
	require.NoError(t, err)
	require.Len(t, plan.Groups, 1)

	// h2 and h3 should be auto-assigned to g1
	assert.Len(t, plan.Groups[0].Files[0].HunkIDs, 3)
}

func TestBuildHunkPatch(t *testing.T) {
	fc := &FileChange{
		Path:   "main.go",
		Status: StatusModified,
		Diff:   "diff --git a/main.go b/main.go\nindex abc..def 100644\n--- a/main.go\n+++ b/main.go\n@@ -1,3 +1,5 @@\n+line\n",
		Hunks: []Hunk{
			{ID: "h1", Content: "@@ -1,3 +1,5 @@\n+feat line\n"},
			{ID: "h2", Content: "@@ -20,3 +20,5 @@\n+fix line\n"},
		},
	}

	// Only h2
	patch := buildHunkPatch(fc, []string{"h2"})
	assert.Contains(t, patch, "diff --git a/main.go b/main.go")
	assert.Contains(t, patch, "+fix line")
	assert.NotContains(t, patch, "+feat line")

	// Only h1
	patch = buildHunkPatch(fc, []string{"h1"})
	assert.Contains(t, patch, "+feat line")
	assert.NotContains(t, patch, "+fix line")

	// Empty
	patch = buildHunkPatch(fc, []string{"h99"})
	assert.Empty(t, patch)
}

func TestRenderPlan_WithHunks(t *testing.T) {
	plan := &CommitPlan{
		Groups: []CommitGroup{
			{
				Title:   "feat: partial change",
				Summary: "Only some hunks",
				Files: []CommitFile{
					{Path: "main.go", Status: StatusModified, WholeFile: false, HunkIDs: []string{"h1", "h3"}},
					{Path: "test.go", Status: StatusAdded, WholeFile: true},
				},
			},
		},
	}

	output := RenderPlan(plan)
	assert.Contains(t, output, "main.go [h1,h3]")
	assert.Contains(t, output, "test.go")
	assert.NotContains(t, output, "test.go [")
}
