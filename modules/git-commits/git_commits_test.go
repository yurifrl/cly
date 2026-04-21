package gitcommits

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Changeset parsing tests ---

func TestParseDiff_Empty(t *testing.T) {
	cs := ParseDiff("")
	assert.Empty(t, cs.Files)
}

func TestParseDiff_AddedFile(t *testing.T) {
	diff := `diff --git a/new.go b/new.go
new file mode 100644
index 0000000..abc1234
--- /dev/null
+++ b/new.go
@@ -0,0 +1,10 @@
+package main
+
+func hello() {
+    fmt.Println("hello")
+}
`
	cs := ParseDiff(diff)
	require.Len(t, cs.Files, 1)
	assert.Equal(t, "new.go", cs.Files[0].Path)
	assert.Equal(t, StatusAdded, cs.Files[0].Status)
	assert.Len(t, cs.Files[0].Hunks, 1)
}

func TestParseDiff_ModifiedFile(t *testing.T) {
	diff := `diff --git a/main.go b/main.go
index abc1234..def5678 100644
--- a/main.go
+++ b/main.go
@@ -10,3 +10,5 @@ func main() {
 	fmt.Println("existing")
+	fmt.Println("new line 1")
+	fmt.Println("new line 2")
 }
`
	cs := ParseDiff(diff)
	require.Len(t, cs.Files, 1)
	assert.Equal(t, "main.go", cs.Files[0].Path)
	assert.Equal(t, StatusModified, cs.Files[0].Status)
}

func TestParseDiff_DeletedFile(t *testing.T) {
	diff := `diff --git a/old.go b/old.go
deleted file mode 100644
index abc1234..0000000
--- a/old.go
+++ /dev/null
@@ -1,5 +0,0 @@
-package main
-
-func old() {
-    fmt.Println("old")
-}
`
	cs := ParseDiff(diff)
	require.Len(t, cs.Files, 1)
	assert.Equal(t, "old.go", cs.Files[0].Path)
	assert.Equal(t, StatusDeleted, cs.Files[0].Status)
}

func TestParseDiff_RenamedFile(t *testing.T) {
	diff := `diff --git a/old_name.go b/new_name.go
similarity index 95%
rename from old_name.go
rename to new_name.go
index abc1234..def5678 100644
--- a/old_name.go
+++ b/new_name.go
@@ -1,3 +1,3 @@
-package old
+package new
`
	cs := ParseDiff(diff)
	require.Len(t, cs.Files, 1)
	assert.Equal(t, "new_name.go", cs.Files[0].Path)
	assert.Equal(t, "old_name.go", cs.Files[0].OldPath)
	assert.Equal(t, StatusRenamed, cs.Files[0].Status)
}

func TestParseDiff_MultipleFiles(t *testing.T) {
	diff := `diff --git a/a.go b/a.go
new file mode 100644
index 0000000..abc1234
--- /dev/null
+++ b/a.go
@@ -0,0 +1,3 @@
+package main
diff --git a/b.go b/b.go
index abc1234..def5678 100644
--- a/b.go
+++ b/b.go
@@ -1,3 +1,5 @@
 package main
+func b() {}
`
	cs := ParseDiff(diff)
	require.Len(t, cs.Files, 2)
	assert.Equal(t, "a.go", cs.Files[0].Path)
	assert.Equal(t, "b.go", cs.Files[1].Path)
}

func TestParseDiffGitPath(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{"simple", "diff --git a/foo.go b/foo.go", "foo.go"},
		{"nested", "diff --git a/pkg/config/config.go b/pkg/config/config.go", "pkg/config/config.go"},
		{"path with b/", "diff --git a/a/b/c.go b/a/b/c.go", "a/b/c.go"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, parseDiffGitPath(tt.line))
		})
	}
}

func TestParseDiff_BinaryFile(t *testing.T) {
	diff := `diff --git a/image.gif b/image.gif
new file mode 100644
index 0000000..abc1234
GIT binary patch
literal 12345
zcmV;diff --git fake data inside binary
literal end
`
	cs := ParseDiff(diff)
	// Should parse as a single file, not split on the fake "diff --git" in binary data
	require.Len(t, cs.Files, 1)
	assert.Equal(t, "image.gif", cs.Files[0].Path)
	assert.Equal(t, StatusAdded, cs.Files[0].Status)
}

func TestParseDiff_MultipleHunks(t *testing.T) {
	diff := `diff --git a/main.go b/main.go
index abc1234..def5678 100644
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 package main
+import "fmt"
@@ -10,3 +11,5 @@ func main() {
 	existing()
+	newFunc()
`
	cs := ParseDiff(diff)
	require.Len(t, cs.Files, 1)
	assert.Len(t, cs.Files[0].Hunks, 2)
	assert.Equal(t, "h1", cs.Files[0].Hunks[0].ID)
	assert.Equal(t, "h2", cs.Files[0].Hunks[1].ID)
}

// --- Batching tests ---

func TestBuildBatches_BinaryFileSkipsDiff(t *testing.T) {
	cs := &Changeset{
		Files: []FileChange{
			{Path: "image.gif", Status: StatusAdded, Diff: "diff --git a/image.gif b/image.gif\nnew file mode 100644\nGIT binary patch\nliteral 12345\nblahblah\n"},
		},
	}

	batches := BuildBatches(cs, 40000)
	require.Len(t, batches, 1)
	// The batch text should NOT contain the binary patch data
	assert.NotContains(t, batches[0].Text, "GIT binary patch")
	// But should contain the file path
	assert.Contains(t, batches[0].Text, "image.gif")
}

func TestBuildBatches_SingleBatch(t *testing.T) {
	cs := &Changeset{
		Files: []FileChange{
			{Path: "a.go", Status: StatusModified, Diff: "small diff"},
			{Path: "b.go", Status: StatusAdded, Diff: "another small diff"},
		},
	}

	batches := BuildBatches(cs, 40000)
	assert.Len(t, batches, 1)
	assert.Len(t, batches[0].Files, 2)
	assert.Equal(t, 1, batches[0].TotalCount)
}

func TestBuildBatches_MultipleBatches(t *testing.T) {
	// Create files with large diffs to force multiple batches
	var files []FileChange
	for i := 0; i < 10; i++ {
		largeDiff := make([]byte, 5000)
		for j := range largeDiff {
			largeDiff[j] = 'x'
		}
		files = append(files, FileChange{
			Path:   fmt.Sprintf("file%d.go", i),
			Status: StatusModified,
			Diff:   string(largeDiff),
		})
	}

	cs := &Changeset{Files: files}
	batches := BuildBatches(cs, 20000) // Small budget to force splits

	assert.Greater(t, len(batches), 1, "should create multiple batches")

	// Verify all files are covered
	totalFiles := 0
	for _, b := range batches {
		totalFiles += len(b.Files)
		assert.Equal(t, len(batches), b.TotalCount)
	}
	assert.Equal(t, 10, totalFiles)
}

func TestBuildBatches_CustomSize(t *testing.T) {
	cs := &Changeset{
		Files: []FileChange{
			{Path: "a.go", Status: StatusModified, Diff: makeString(3000)},
			{Path: "b.go", Status: StatusModified, Diff: makeString(3000)},
			{Path: "c.go", Status: StatusModified, Diff: makeString(3000)},
		},
	}

	// With 8K budget, should split (each file analysis > 3K with headers)
	batches := BuildBatches(cs, 8000)
	assert.Greater(t, len(batches), 1)
}

func TestBuildBatches_DefaultSize(t *testing.T) {
	cs := &Changeset{
		Files: []FileChange{
			{Path: "a.go", Status: StatusModified, Diff: "tiny"},
		},
	}

	batches := BuildBatches(cs, 0) // Should use default
	assert.Len(t, batches, 1)
}

// --- Planner tests ---

func TestExtractPlan_ValidJSON(t *testing.T) {
	resp := `{"groups": [{"title": "feat: add feature", "type": "feat", "summary": "test", "items": [{"file": "a.go"}]}]}`
	plan, err := extractPlan(resp)
	require.NoError(t, err)
	require.Len(t, plan.Groups, 1)
	assert.Equal(t, "feat: add feature", plan.Groups[0].Title)
}

func TestExtractPlan_WithMarkdownFences(t *testing.T) {
	resp := "```json\n{\"groups\": [{\"title\": \"fix: bug\", \"type\": \"fix\", \"summary\": \"test\", \"items\": [{\"file\": \"b.go\"}]}]}\n```"
	plan, err := extractPlan(resp)
	require.NoError(t, err)
	require.Len(t, plan.Groups, 1)
	assert.Equal(t, "fix: bug", plan.Groups[0].Title)
}

func TestExtractPlan_WithSurroundingText(t *testing.T) {
	resp := `Here's the plan:
{"groups": [{"title": "chore: cleanup", "type": "chore", "summary": "test", "items": [{"file": "c.go"}]}]}
That looks good!`
	plan, err := extractPlan(resp)
	require.NoError(t, err)
	require.Len(t, plan.Groups, 1)
}

func TestExtractPlan_InvalidJSON(t *testing.T) {
	_, err := extractPlan("not json at all")
	assert.Error(t, err)
}

func TestExtractPlan_EmptyGroups(t *testing.T) {
	_, err := extractPlan(`{"groups": []}`)
	assert.Error(t, err)
}

func TestInferType(t *testing.T) {
	tests := []struct {
		title    string
		expected string
	}{
		{"feat: add feature", "feat"},
		{"fix: bug", "fix"},
		{"chore(deps): update", "chore"},
		{"refactor: clean up", "refactor"},
		{"no type here", "chore"},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			assert.Equal(t, tt.expected, inferType(tt.title))
		})
	}
}

func TestScaleMaxTokens(t *testing.T) {
	assert.Equal(t, 4000, ScaleMaxTokens(5000))
	assert.Equal(t, 8000, ScaleMaxTokens(15000))
	assert.Equal(t, 16000, ScaleMaxTokens(50000))
}

// --- Validator tests ---

func TestValidatePlan_Basic(t *testing.T) {
	cs := &Changeset{
		Files: []FileChange{
			{Path: "a.go", Status: StatusModified},
			{Path: "b.go", Status: StatusAdded},
		},
	}
	raw := &RawPlan{
		Groups: []RawGroup{
			{Title: "feat: add b", Type: "feat", Summary: "adds b", Items: []RawItem{{File: "b.go"}}},
			{Title: "fix: modify a", Type: "fix", Summary: "fixes a", Items: []RawItem{{File: "a.go"}}},
		},
	}

	plan, err := ValidatePlan(raw, cs, false)
	require.NoError(t, err)
	require.Len(t, plan.Groups, 2)
	assert.Equal(t, "feat: add b", plan.Groups[0].Title)
	assert.Equal(t, "fix: modify a", plan.Groups[1].Title)
}

func TestValidatePlan_Dedup(t *testing.T) {
	cs := &Changeset{
		Files: []FileChange{
			{Path: "a.go", Status: StatusModified},
		},
	}
	raw := &RawPlan{
		Groups: []RawGroup{
			{Title: "g1", Items: []RawItem{{File: "a.go"}}},
			{Title: "g2", Items: []RawItem{{File: "a.go"}}}, // Duplicate
		},
	}

	plan, err := ValidatePlan(raw, cs, false)
	require.NoError(t, err)
	// g2 becomes empty after dedup and gets dropped
	assert.Len(t, plan.Groups, 1)
	assert.Equal(t, "g1", plan.Groups[0].Title)
}

func TestValidatePlan_UncoveredFiles(t *testing.T) {
	cs := &Changeset{
		Files: []FileChange{
			{Path: "pkg/foo/a.go", Status: StatusModified},
			{Path: "pkg/foo/b.go", Status: StatusAdded},   // Not in plan
			{Path: "pkg/bar/c.go", Status: StatusModified}, // Not in plan
		},
	}
	raw := &RawPlan{
		Groups: []RawGroup{
			{Title: "g1", Items: []RawItem{{File: "pkg/foo/a.go"}}},
		},
	}

	plan, err := ValidatePlan(raw, cs, false)
	require.NoError(t, err)
	// b.go should be auto-assigned to g1 (same pkg/foo prefix)
	assert.Len(t, plan.Groups, 1)
	assert.Len(t, plan.Groups[0].Files, 3)
}

func TestValidatePlan_Empty(t *testing.T) {
	cs := &Changeset{Files: []FileChange{{Path: "a.go", Status: StatusModified}}}
	raw := &RawPlan{Groups: []RawGroup{}}

	_, err := ValidatePlan(raw, cs, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty plan")
}

func TestValidatePlan_RenameResolution(t *testing.T) {
	cs := &Changeset{
		Files: []FileChange{
			{Path: "new.go", OldPath: "old.go", Status: StatusRenamed},
		},
	}
	raw := &RawPlan{
		Groups: []RawGroup{
			{Title: "rename", Items: []RawItem{{File: "old.go"}}}, // References old path
		},
	}

	plan, err := ValidatePlan(raw, cs, false)
	require.NoError(t, err)
	require.Len(t, plan.Groups, 1)
	assert.Equal(t, "new.go", plan.Groups[0].Files[0].Path)
	assert.Equal(t, "old.go", plan.Groups[0].Files[0].OldPath)
}

func TestValidatePlan_NilPlan(t *testing.T) {
	cs := &Changeset{Files: []FileChange{{Path: "a.go", Status: StatusModified}}}
	_, err := ValidatePlan(nil, cs, false)
	assert.Error(t, err)
}

// --- Revision prompt tests ---

func TestBuildRevisionPrompt(t *testing.T) {
	plan := &CommitPlan{
		Groups: []CommitGroup{
			{Title: "feat: add auth", Files: []CommitFile{{Path: "auth.go"}, {Path: "auth_test.go"}}},
			{Title: "fix: config bug", Files: []CommitFile{{Path: "config.go"}}},
		},
	}

	result := buildRevisionPrompt("", plan, "combine commits 1 and 2")
	assert.Contains(t, result, "PREVIOUS PLAN")
	assert.Contains(t, result, "feat: add auth")
	assert.Contains(t, result, "fix: config bug")
	assert.Contains(t, result, "auth.go")
	assert.Contains(t, result, "config.go")
	assert.Contains(t, result, "USER FEEDBACK: combine commits 1 and 2")
}

func TestBuildRevisionPrompt_WithExisting(t *testing.T) {
	plan := &CommitPlan{
		Groups: []CommitGroup{
			{Title: "chore: cleanup", Files: []CommitFile{{Path: "a.go"}}},
		},
	}

	result := buildRevisionPrompt("prefer small commits", plan, "split config separately")
	assert.Contains(t, result, "prefer small commits")
	assert.Contains(t, result, "PREVIOUS PLAN")
	assert.Contains(t, result, "USER FEEDBACK: split config separately")
}

// --- Preview tests ---

func TestRenderPlan(t *testing.T) {
	plan := &CommitPlan{
		Groups: []CommitGroup{
			{
				Title:   "feat: add feature",
				Summary: "Adds the new feature",
				Files: []CommitFile{
					{Path: "feature.go", Status: StatusAdded},
				},
			},
			{
				Title:   "fix: broken test",
				Summary: "Fixes test",
				Files: []CommitFile{
					{Path: "test.go", Status: StatusModified},
				},
			},
		},
	}

	output := RenderPlan(plan)
	assert.Contains(t, output, "feat: add feature")
	assert.Contains(t, output, "fix: broken test")
	assert.Contains(t, output, "feature.go")
	assert.Contains(t, output, "test.go")
	assert.Contains(t, output, "2 commits")
}

// --- Helpers ---

func makeString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'x'
	}
	return string(b)
}
