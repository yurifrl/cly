package dotfiles

import (
	"encoding/json"
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
		InstallCommands: []InstallCommand{{Command: "brew install fzf"}, {Command: "brew install ripgrep"}},
		CacheEntries: []CacheEntry{
			{Command: "echo fzf"},
		},
		OpMappings: []OpMapping{
			{Source: "/dotfiles/.env.op", Destination: "/home/user/.env"},
		},
	}

	lock := buildLock(cfg, nil)

	require.Len(t, lock.Symlinks, 1)
	assert.Equal(t, "/dotfiles/.zshrc", lock.Symlinks[0].Source)
	assert.Equal(t, "/home/user/.zshrc", lock.Symlinks[0].Destination)

	require.Len(t, lock.JsoncCopies, 1)
	assert.Equal(t, "/dotfiles/settings.jsonc", lock.JsoncCopies[0].Source)
	assert.Equal(t, "/home/user/settings.json", lock.JsoncCopies[0].Destination)

	// Cache entries with no prior lock state are not pre-seeded; they are
	// inserted by ApplyCache after the command runs.
	assert.Empty(t, lock.Cache)

	require.Len(t, lock.InstallCommands, 2)
	assert.Equal(t, "brew install fzf", lock.InstallCommands[0])

	require.Len(t, lock.OpMappings, 1)
	assert.Equal(t, "/home/user/.env", lock.OpMappings[0].Destination)
}

func TestBuildLock_PreservesInactiveOverlayMappings(t *testing.T) {
	old := &DotfilesLock{Symlinks: []LockEntry{
		{Source: "/base/old", Destination: "/home/user/base", Config: "dotfiles.conf"},
		{Source: "/alice/old", Destination: "/home/user/alice", Config: "dotfiles.alice.conf"},
	}}
	cfg := &Config{
		ConfigPaths: []string{"dotfiles.conf", "dotfiles.bob.conf"},
		Mappings: []Mapping{
			{Source: "/base/new", Destination: "/home/user/base", ConfigPath: "dotfiles.conf"},
			{Source: "/bob/new", Destination: "/home/user/bob", ConfigPath: "dotfiles.bob.conf"},
		},
	}

	lock := buildLock(cfg, old)

	assert.ElementsMatch(t, []LockEntry{
		{Source: "/base/new", Destination: "/home/user/base", Config: "dotfiles.conf"},
		{Source: "/bob/new", Destination: "/home/user/bob", Config: "dotfiles.bob.conf"},
		{Source: "/alice/old", Destination: "/home/user/alice", Config: "dotfiles.alice.conf"},
	}, lock.Symlinks)
	assert.Empty(t, diffLocks(old, lock).RemovedSymlinks,
		"switching from alice to bob must not schedule alice's links for removal")
}

func TestBuildLock_PreservesLegacyUnownedMappings(t *testing.T) {
	old := &DotfilesLock{Symlinks: []LockEntry{
		{Source: "/legacy/source", Destination: "/home/user/legacy"},
	}}
	cfg := &Config{ConfigPaths: []string{"dotfiles.conf"}}

	lock := buildLock(cfg, old)

	assert.Equal(t, old.Symlinks, lock.Symlinks)
	assert.Empty(t, diffLocks(old, lock).RemovedSymlinks,
		"unowned legacy entries must not be deleted during the ownership migration")
}

func TestBuildLock_PreservesHashes(t *testing.T) {
	jobA := CacheEntry{Command: "echo job-a"}
	jobB := CacheEntry{Command: "echo job-b"}
	cfg := &Config{
		CacheEntries: []CacheEntry{jobA, jobB},
	}
	hashA := hashCacheEntry(jobA)
	hashB := hashCacheEntry(jobB)
	prev := &DotfilesLock{
		Cache: []CacheLockEntry{
			{Hash: hashA, Command: "echo job-a", LastRun: "2026-01-01T00:00:00Z"},
			{Hash: "def456stale", Command: "echo stale"},
		},
	}
	lock := buildLock(cfg, prev)

	// New behavior: cfg entries with prev metadata are carried forward;
	// new cfg entries (job-b) are NOT pre-seeded — ApplyCache inserts
	// them after running. Stale prev entries are kept around so
	// pruneStaleCacheEntries can flag/age them.
	require.Len(t, lock.Cache, 2)
	byHash := make(map[string]CacheLockEntry)
	for _, e := range lock.Cache {
		byHash[e.Hash] = e
	}
	assert.Equal(t, "2026-01-01T00:00:00Z", byHash[hashA].LastRun, "prev metadata carried forward")
	_, hasB := byHash[hashB]
	assert.False(t, hasB, "new entry not pre-seeded; ApplyCache will insert")
	assert.Equal(t, "echo stale", byHash["def456stale"].Command, "stale entry carried forward for prune accounting")
}

func TestDiffLocks_NoChanges(t *testing.T) {
	lock := &DotfilesLock{
		Symlinks:        []LockEntry{{Source: "/src/.zshrc", Destination: "/dst/.zshrc"}},
		Cache:           []CacheLockEntry{{Hash: "abc", Command: "echo my-job"}},
		InstallCommands: []string{"brew install fzf"},
	}

	diff := diffLocks(lock, lock)

	assert.Empty(t, diff.RemovedSymlinks)
	assert.Empty(t, diff.RemovedCache)
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

func TestDiffLocks_RemovedCache(t *testing.T) {
	old := &DotfilesLock{Cache: []CacheLockEntry{{Hash: "h-a"}, {Hash: "h-b"}}}
	new := &DotfilesLock{Cache: []CacheLockEntry{{Hash: "h-a"}}}

	diff := diffLocks(old, new)

	require.Len(t, diff.RemovedCache, 1)
	assert.Equal(t, "h-b", diff.RemovedCache[0])
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
	assert.Empty(t, lock.Cache)
}

func TestSaveLock_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dotfiles.lock")

	original := &DotfilesLock{
		Symlinks:        []LockEntry{{Source: "/src/.zshrc", Destination: "/dst/.zshrc"}},
		Cache:           []CacheLockEntry{{Hash: "abc123", Command: "echo my-job"}},
		InstallCommands: []string{"brew install fzf"},
	}

	err := saveLock(path, original)
	require.NoError(t, err)

	loaded, err := loadLock(path)
	require.NoError(t, err)

	assert.Equal(t, original.Symlinks, loaded.Symlinks)
	assert.Equal(t, original.Cache, loaded.Cache)
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

func TestCacheLock_LegacyStringUnmarshal(t *testing.T) {
	// Legacy bare-string entries had no hash — they're orphaned and dropped.
	raw := `{"jobs": ["job-a", "job-b"], "symlinks": []}`
	var lock DotfilesLock
	require.NoError(t, json.Unmarshal([]byte(raw), &lock))
	assert.Empty(t, lock.Cache, "legacy bare-string entries are orphaned and dropped")
}

func TestCacheLock_LegacyJobsObjectUnmarshal(t *testing.T) {
	// Old name+hash form: hash is kept, name is silently dropped.
	raw := `{"jobs": [{"name": "job-a", "hash": "abc123"}]}`
	var lock DotfilesLock
	require.NoError(t, json.Unmarshal([]byte(raw), &lock))

	require.Len(t, lock.Cache, 1)
	assert.Equal(t, "abc123", lock.Cache[0].Hash)
}

func TestCacheLock_LegacyJobsObjectDropsNameOnlyEntries(t *testing.T) {
	// Old name-only entries (no hash) are orphaned legacy and get dropped.
	raw := `{"cache": [{"name": "orphan"}, {"name": "keep", "hash": "h1"}]}`
	var lock DotfilesLock
	require.NoError(t, json.Unmarshal([]byte(raw), &lock))
	require.Len(t, lock.Cache, 1)
	assert.Equal(t, "h1", lock.Cache[0].Hash)
}

func TestCacheLock_NewFormatUnmarshal(t *testing.T) {
	raw := `{"cache": [{"hash": "abc123", "command": "echo job-a"}]}`
	var lock DotfilesLock
	require.NoError(t, json.Unmarshal([]byte(raw), &lock))

	require.Len(t, lock.Cache, 1)
	assert.Equal(t, "abc123", lock.Cache[0].Hash)
	assert.Equal(t, "echo job-a", lock.Cache[0].Command)
}

func TestLoadLock_MigratesLegacyJobsState(t *testing.T) {
	tmpHome := t.TempDir()
	oldHome := os.Getenv("HOME")
	require.NoError(t, os.Setenv("HOME", tmpHome))
	defer os.Setenv("HOME", oldHome)

	// Write a legacy jobs-state.json in the default data dir.
	stateDir := filepath.Join(tmpHome, ".local/share/cly/dotfiles")
	require.NoError(t, os.MkdirAll(stateDir, 0755))
	stateData := `{"jobs": {"once-job": "hashABC"}}`
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "jobs-state.json"), []byte(stateData), 0644))

	lock, err := loadLock("/nonexistent/path/dotfiles.lock")
	require.NoError(t, err)

	commands := lockCacheToMap(lock.Cache)
	// Legacy jobs-state hashes are carried forward keyed by hash.
	_, ok := commands["hashABC"]
	assert.True(t, ok, "legacy jobs-state hash should be migrated into the lock")

	_, statErr := os.Stat(filepath.Join(stateDir, "jobs-state.json"))
	assert.True(t, os.IsNotExist(statErr))
}
