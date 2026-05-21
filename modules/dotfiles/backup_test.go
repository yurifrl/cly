package dotfiles

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetBackupState resets the package-level lazy backup root so each test
// gets a fresh, isolated backup directory.
func resetBackupState() { resetBackupForTest() }

func TestCreateSymlinkBacksUpExistingFile(t *testing.T) {
	t.Setenv("CLY_BACKUP_DIR", t.TempDir())
	resetBackupState()

	tmp := t.TempDir()
	src := filepath.Join(tmp, "src.txt")
	dst := filepath.Join(tmp, "dst.txt")

	require.NoError(t, os.WriteFile(src, []byte("source"), 0644))
	require.NoError(t, os.WriteFile(dst, []byte("ORIGINAL CONTENT"), 0644))

	res := CreateSymlink(Mapping{Source: src, Destination: dst})
	require.Equal(t, StateLinked, res.State, "error: %s", res.Error)
	assert.True(t, res.RemovedExisting)
	require.NotEmpty(t, res.BackupPath, "expected backup path to be set")

	// Backup file should still hold the original bytes.
	got, err := os.ReadFile(res.BackupPath)
	require.NoError(t, err)
	assert.Equal(t, "ORIGINAL CONTENT", string(got))

	// Destination should now be a symlink pointing at the source.
	target, err := os.Readlink(dst)
	require.NoError(t, err)
	assert.Equal(t, src, target)
}

func TestCreateSymlinkBacksUpExistingDirectory(t *testing.T) {
	t.Setenv("CLY_BACKUP_DIR", t.TempDir())
	resetBackupState()

	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "src") + string(os.PathSeparator)
	dstDir := filepath.Join(tmp, "dst")

	require.NoError(t, os.MkdirAll(strings.TrimSuffix(srcDir, string(os.PathSeparator)), 0755))
	require.NoError(t, os.MkdirAll(dstDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dstDir, "keep.txt"), []byte("keep me"), 0644))

	res := CreateSymlink(Mapping{
		Source:      strings.TrimSuffix(srcDir, string(os.PathSeparator)),
		Destination: dstDir,
		IsDir:       true,
	})
	require.Equal(t, StateLinked, res.State, "error: %s", res.Error)
	require.NotEmpty(t, res.BackupPath, "expected backup path to be set")

	backedUp, err := os.ReadFile(filepath.Join(res.BackupPath, "keep.txt"))
	require.NoError(t, err)
	assert.Equal(t, "keep me", string(backedUp))
}

func TestBackupExistingNoopWhenMissing(t *testing.T) {
	t.Setenv("CLY_BACKUP_DIR", t.TempDir())
	resetBackupState()

	got, err := BackupExisting(filepath.Join(t.TempDir(), "does-not-exist"))
	require.NoError(t, err)
	assert.Empty(t, got)
}
