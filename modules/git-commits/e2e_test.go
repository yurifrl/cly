package gitcommits

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_ExecuteCommits creates a real git repo and tests the executor.
func TestE2E_ExecuteCommits(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	// Create temp dir with git repo
	dir := t.TempDir()
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v failed: %s", args, out)
	}

	// Init repo with initial commit
	runGit("init")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test\n"), 0644))
	runGit("add", ".")
	runGit("commit", "-m", "initial commit")

	// Create mixed changes
	require.NoError(t, os.WriteFile(filepath.Join(dir, "feat.go"), []byte("package main\nfunc feat() {}\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "fix.go"), []byte("package main\nfunc fix() {}\n"), 0644))
	runGit("add", ".")

	// Change to the temp dir for git operations
	oldDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer os.Chdir(oldDir)

	// Build a plan manually (no LLM needed)
	plan := &CommitPlan{
		Groups: []CommitGroup{
			{
				Title:   "feat: add feature",
				Type:    "feat",
				Summary: "adds feat.go",
				Files: []CommitFile{
					{Path: "feat.go", Status: StatusAdded},
				},
			},
			{
				Title:   "fix: add fix",
				Type:    "fix",
				Summary: "adds fix.go",
				Files: []CommitFile{
					{Path: "fix.go", Status: StatusAdded},
				},
			},
		},
	}

	results, err := Execute(plan, true) // --no-verify
	require.NoError(t, err)
	require.Len(t, results, 2)

	assert.Equal(t, "feat: add feature", results[0].Title)
	assert.Equal(t, "fix: add fix", results[1].Title)
	assert.NotEmpty(t, results[0].SHA)
	assert.NotEmpty(t, results[1].SHA)

	// Verify git log has 3 commits (initial + 2 new)
	cmd := exec.Command("git", "log", "--oneline")
	cmd.Dir = dir
	out, err := cmd.Output()
	require.NoError(t, err)

	lines := 0
	for _, line := range splitLines(string(out)) {
		if line != "" {
			lines++
		}
	}
	assert.Equal(t, 3, lines, "should have 3 commits")
}

// TestE2E_ExecuteRollback tests that rollback works on failure.
func TestE2E_ExecuteRollback(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	dir := t.TempDir()
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v failed: %s", args, out)
	}

	// Init repo
	runGit("init")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test\n"), 0644))
	runGit("add", ".")
	runGit("commit", "-m", "initial")

	// Stage a real file
	require.NoError(t, os.WriteFile(filepath.Join(dir, "real.go"), []byte("package main\n"), 0644))
	runGit("add", ".")

	oldDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer os.Chdir(oldDir)

	// Record HEAD before
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	headBefore, err := cmd.Output()
	require.NoError(t, err)

	// Plan with a non-existent file that will cause failure
	plan := &CommitPlan{
		Groups: []CommitGroup{
			{
				Title: "good: first commit",
				Files: []CommitFile{
					{Path: "real.go", Status: StatusAdded},
				},
			},
			{
				Title: "bad: will fail",
				Files: []CommitFile{
					{Path: "nonexistent.go", Status: StatusAdded}, // This file doesn't exist
				},
			},
		},
	}

	_, err = Execute(plan, true)
	assert.Error(t, err, "should fail on nonexistent file")

	// HEAD should be restored
	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	headAfter, err := cmd.Output()
	require.NoError(t, err)
	assert.Equal(t, string(headBefore), string(headAfter), "HEAD should be restored after rollback")
}

func splitLines(s string) []string {
	var lines []string
	for _, line := range []string{} {
		_ = line
	}
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
