package dotfiles

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildLock(t *testing.T) {
	cfg := &Config{
		Mappings: []Mapping{
			{Source: "/dotfiles/.zshrc", Destination: "/home/user/.zshrc"},
			{Source: "/dotfiles/settings.jsonc", Destination: "/home/user/settings.json"},
		},
		InstallCommands: []string{"brew install fzf", "brew install ripgrep"},
		Jobs: []Job{
			{Name: "sync-dotfiles", Run: JobRunStartup},
		},
		OpMappings: []OpMapping{
			{Source: "/dotfiles/.env.op", Destination: "/home/user/.env"},
		},
	}

	lock := buildLock(cfg)

	require.Len(t, lock.Symlinks, 1)
	assert.Equal(t, "/dotfiles/.zshrc", lock.Symlinks[0].Source)
	assert.Equal(t, "/home/user/.zshrc", lock.Symlinks[0].Destination)

	require.Len(t, lock.JsoncCopies, 1)
	assert.Equal(t, "/dotfiles/settings.jsonc", lock.JsoncCopies[0].Source)
	assert.Equal(t, "/home/user/settings.json", lock.JsoncCopies[0].Destination)

	require.Len(t, lock.Jobs, 1)
	assert.Equal(t, "sync-dotfiles", lock.Jobs[0])

	require.Len(t, lock.InstallCommands, 2)
	assert.Equal(t, "brew install fzf", lock.InstallCommands[0])

	require.Len(t, lock.OpMappings, 1)
	assert.Equal(t, "/home/user/.env", lock.OpMappings[0].Destination)
}

func TestDiffLocks_NoChanges(t *testing.T) {
	lock := &DotfilesLock{
		Symlinks:        []LockEntry{{Source: "/src/.zshrc", Destination: "/dst/.zshrc"}},
		Jobs:            []string{"my-job"},
		InstallCommands: []string{"brew install fzf"},
	}

	diff := diffLocks(lock, lock)

	assert.Empty(t, diff.RemovedSymlinks)
	assert.Empty(t, diff.RemovedJobs)
	assert.Empty(t, diff.RemovedInstallCommands)
}

func TestDiffLocks_RemovedSymlink(t *testing.T) {
	old := &DotfilesLock{
		Symlinks: []LockEntry{
			{Source: "/src/.zshrc", Destination: "/dst/.zshrc"},
			{Source: "/src/.vimrc", Destination: "/dst/.vimrc"},
		},
	}
	new := &DotfilesLock{
		Symlinks: []LockEntry{
			{Source: "/src/.zshrc", Destination: "/dst/.zshrc"},
		},
	}

	diff := diffLocks(old, new)

	require.Len(t, diff.RemovedSymlinks, 1)
	assert.Equal(t, "/dst/.vimrc", diff.RemovedSymlinks[0].Destination)
	assert.Empty(t, diff.RemovedJsoncCopies)
}

func TestDiffLocks_RemovedJob(t *testing.T) {
	old := &DotfilesLock{Jobs: []string{"job-a", "job-b"}}
	new := &DotfilesLock{Jobs: []string{"job-a"}}

	diff := diffLocks(old, new)

	require.Len(t, diff.RemovedJobs, 1)
	assert.Equal(t, "job-b", diff.RemovedJobs[0])
}

func TestDiffLocks_RemovedInstallCommand(t *testing.T) {
	old := &DotfilesLock{InstallCommands: []string{"brew install fzf", "brew install bat"}}
	new := &DotfilesLock{InstallCommands: []string{"brew install fzf"}}

	diff := diffLocks(old, new)

	require.Len(t, diff.RemovedInstallCommands, 1)
	assert.Equal(t, "brew install bat", diff.RemovedInstallCommands[0])
}

func TestDiffLocks_RemovedOpMapping(t *testing.T) {
	old := &DotfilesLock{
		OpMappings: []LockEntry{{Source: "/src/.env.op", Destination: "/dst/.env"}},
	}
	new := &DotfilesLock{}

	diff := diffLocks(old, new)

	require.Len(t, diff.RemovedOpMappings, 1)
	assert.Equal(t, "/dst/.env", diff.RemovedOpMappings[0].Destination)
}

func TestLoadLock_NotExist(t *testing.T) {
	lock, err := loadLock("/nonexistent/path/dotfiles.lock")
	require.NoError(t, err)
	assert.Empty(t, lock.Symlinks)
	assert.Empty(t, lock.Jobs)
}

func TestSaveLock_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dotfiles.lock")

	original := &DotfilesLock{
		Symlinks:        []LockEntry{{Source: "/src/.zshrc", Destination: "/dst/.zshrc"}},
		Jobs:            []string{"my-job"},
		InstallCommands: []string{"brew install fzf"},
	}

	err := saveLock(path, original)
	require.NoError(t, err)

	loaded, err := loadLock(path)
	require.NoError(t, err)

	assert.Equal(t, original.Symlinks, loaded.Symlinks)
	assert.Equal(t, original.Jobs, loaded.Jobs)
	assert.Equal(t, original.InstallCommands, loaded.InstallCommands)
}

func TestSaveLock_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "dotfiles.lock")

	err := saveLock(path, &DotfilesLock{})
	require.NoError(t, err)

	_, err = os.Stat(path)
	assert.NoError(t, err)
}
