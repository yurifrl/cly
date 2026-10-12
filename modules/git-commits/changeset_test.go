package gitcommits

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// Regression: gc -a crashed mid-plan because index entries staged before the
// run but since deleted from the worktree (or newly covered by ignore rules)
// could not be re-added after the executor's reset, rolling back the whole
// plan. GetChangeset must prune such stale staged entries up front.
func TestGetChangesetPrunesStaleStaged(t *testing.T) {
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

	runGit("init")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test\n"), 0644))
	runGit("add", ".")
	runGit("commit", "-m", "initial")

	// keep: plain staged new file.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("keep\n"), 0644))
	// deleted: staged, then removed from the worktree.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "deleted.txt"), []byte("bye\n"), 0644))
	// ignored: matched by the ignore rule staged below (pytest-cache style).
	require.NoError(t, os.WriteFile(filepath.Join(dir, "junk.log"), []byte("log\n"), 0644))
	runGit("add", "-f", "keep.txt", "deleted.txt", "junk.log")
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.log\n"), 0644))
	runGit("add", ".gitignore")
	require.NoError(t, os.Remove(filepath.Join(dir, "deleted.txt")))

	oldDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { os.Chdir(oldDir) })

	cs, err := GetChangeset(false)
	require.NoError(t, err)

	var got []string
	for _, f := range cs.Files {
		got = append(got, f.Path)
	}
	require.ElementsMatch(t, []string{"keep.txt", ".gitignore"}, got)

	// The pruned entries must also leave the index so later groups and runs
	// start clean.
	out, err := exec.Command("git", "ls-files").CombinedOutput()
	require.NoError(t, err)
	lsFiles := string(out)
	require.Contains(t, lsFiles, "keep.txt")
	require.NotContains(t, lsFiles, "deleted.txt")
	require.NotContains(t, lsFiles, "junk.log")
}
