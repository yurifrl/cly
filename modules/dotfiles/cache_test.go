package dotfiles

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withShellSh swaps the cache shell to /bin/sh so we don't depend on fish.
func withShellSh(t *testing.T) {
	t.Helper()
	prev := cacheShell
	cacheShell = "sh"
	t.Cleanup(func() { cacheShell = prev })
}

func TestApplyCache_HashSkipAndInvalidate(t *testing.T) {
	withShellSh(t)
	tmp := t.TempDir()
	marker := filepath.Join(tmp, "marker")
	cfg := &Config{
		BaseDir: tmp,
		CacheEntries: []CacheEntry{
			{Command: "touch " + marker},
		},
	}
	lock := &DotfilesLock{}

	// First run: marker created, lock populated.
	require.NoError(t, ApplyCache(cfg, lock, CacheApplyOptions{}))
	require.Len(t, lock.Cache, 1)
	first := lock.Cache[0]
	assert.NotEmpty(t, first.Hash)
	assert.NotEmpty(t, first.LastRun)
	assert.Equal(t, 0, first.ExitCode)
	assert.Equal(t, "touch "+marker, first.Command)
	_, err := os.Stat(marker)
	require.NoError(t, err)

	// Remove marker; second run with same command should be skipped (cache hit).
	require.NoError(t, os.Remove(marker))
	require.NoError(t, ApplyCache(cfg, lock, CacheApplyOptions{}))
	_, err = os.Stat(marker)
	assert.True(t, os.IsNotExist(err), "skip should NOT recreate marker")
	assert.Equal(t, first.LastRun, lock.Cache[0].LastRun, "metadata preserved on cache hit")

	// Edit command -> hash invalidates -> reruns.
	cfg.CacheEntries[0].Command = "touch " + marker + " && echo ok"
	require.NoError(t, ApplyCache(cfg, lock, CacheApplyOptions{}))
	// New hash entry is appended; old entry remains in lock until prune.
	require.GreaterOrEqual(t, len(lock.Cache), 1)
	var updated CacheLockEntry
	for _, e := range lock.Cache {
		if e.Command == cfg.CacheEntries[0].Command {
			updated = e
		}
	}
	require.NotEmpty(t, updated.Hash)
	assert.NotEqual(t, first.Hash, updated.Hash, "hash should change with command")
	_, err = os.Stat(marker)
	require.NoError(t, err)
}

func TestApplyCache_ExitCodeCaptured(t *testing.T) {
	withShellSh(t)
	cfg := &Config{
		BaseDir: t.TempDir(),
		CacheEntries: []CacheEntry{
			{Command: "exit 7"},
		},
	}
	lock := &DotfilesLock{}
	require.NoError(t, ApplyCache(cfg, lock, CacheApplyOptions{}))
	require.Len(t, lock.Cache, 1)
	assert.Equal(t, 7, lock.Cache[0].ExitCode)
}

func TestApplyCache_ForceBypassesHash(t *testing.T) {
	withShellSh(t)
	tmp := t.TempDir()
	counter := filepath.Join(tmp, "count")
	cfg := &Config{
		BaseDir: tmp,
		CacheEntries: []CacheEntry{
			{Command: "echo x >> " + counter},
		},
	}
	lock := &DotfilesLock{}
	require.NoError(t, ApplyCache(cfg, lock, CacheApplyOptions{}))
	require.NoError(t, ApplyCache(cfg, lock, CacheApplyOptions{}))            // skipped
	require.NoError(t, ApplyCache(cfg, lock, CacheApplyOptions{Force: true})) // re-runs
	data, err := os.ReadFile(counter)
	require.NoError(t, err)
	assert.Equal(t, "x\nx\n", string(data), "force should re-run; non-force should skip")
}

func TestApplyCache_DuplicateLinesCollapse(t *testing.T) {
	withShellSh(t)
	tmp := t.TempDir()
	counter := filepath.Join(tmp, "count")
	// Two identical @exec lines hash to the same value: the first
	// populates the lock entry, the second is a cache hit.
	cfg := &Config{
		BaseDir: tmp,
		CacheEntries: []CacheEntry{
			{Command: "echo x >> " + counter},
			{Command: "echo x >> " + counter},
		},
	}
	lock := &DotfilesLock{}
	require.NoError(t, ApplyCache(cfg, lock, CacheApplyOptions{}))
	require.Len(t, lock.Cache, 1, "duplicate identical commands collapse to one lock entry")
	data, err := os.ReadFile(counter)
	require.NoError(t, err)
	assert.Equal(t, "x\n", string(data), "duplicate identical @exec lines run only once")
}

func TestPruneStaleCacheEntries_FlagOnFirstMiss(t *testing.T) {
	cfg := &Config{}
	lock := &DotfilesLock{Cache: []CacheLockEntry{
		{Hash: "h", Command: "echo gone"},
	}}
	now := time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)
	kept, flagged, pruned := pruneStaleCacheEntries(lock, cfg, now, false)
	assert.Empty(t, kept)
	require.Len(t, flagged, 1)
	assert.Empty(t, pruned)
	assert.NotEmpty(t, flagged[0].FlaggedForDelete)
	require.Len(t, lock.Cache, 1)
	assert.NotEmpty(t, lock.Cache[0].FlaggedForDelete)
}

func TestPruneStaleCacheEntries_AfterGracePeriod(t *testing.T) {
	cfg := &Config{}
	now := time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)
	flaggedAt := now.Add(-8 * 24 * time.Hour).Format(time.RFC3339)
	lock := &DotfilesLock{Cache: []CacheLockEntry{
		{Hash: "h", Command: "echo stale", FlaggedForDelete: flaggedAt},
	}}
	kept, flagged, pruned := pruneStaleCacheEntries(lock, cfg, now, false)
	assert.Empty(t, kept)
	assert.Empty(t, flagged)
	require.Len(t, pruned, 1)
	assert.Empty(t, lock.Cache, "entry past grace should be removed from lock")
}

func TestPruneStaleCacheEntries_HardPrune(t *testing.T) {
	cfg := &Config{}
	now := time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)
	flaggedAt := now.Add(-1 * time.Hour).Format(time.RFC3339)
	lock := &DotfilesLock{Cache: []CacheLockEntry{
		{Hash: "h", Command: "echo a", FlaggedForDelete: flaggedAt},
		{Hash: "h2", Command: "echo b"},
	}}
	_, _, pruned := pruneStaleCacheEntries(lock, cfg, now, true)
	require.Len(t, pruned, 2)
	assert.Empty(t, lock.Cache)
}

func TestPruneStaleCacheEntries_KeepsConfigEntriesAndClearsFlag(t *testing.T) {
	cfg := &Config{CacheEntries: []CacheEntry{{Command: "echo alive"}}}
	aliveHash := hashCacheEntry(cfg.CacheEntries[0])
	now := time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)
	lock := &DotfilesLock{Cache: []CacheLockEntry{
		{Hash: aliveHash, Command: "echo alive", FlaggedForDelete: now.Format(time.RFC3339)},
	}}
	kept, flagged, pruned := pruneStaleCacheEntries(lock, cfg, now, false)
	require.Len(t, kept, 1)
	assert.Empty(t, flagged)
	assert.Empty(t, pruned)
	assert.Empty(t, lock.Cache[0].FlaggedForDelete, "flag cleared once entry reappears in cfg")
}

func TestCacheLockEntry_BackwardCompatRoundTrip(t *testing.T) {
	// Old schema with name+hash: name is silently dropped, hash is preserved.
	raw := `{"cache":[{"name":"old","hash":"h1"}]}`
	var lock DotfilesLock
	require.NoError(t, json.Unmarshal([]byte(raw), &lock))
	require.Len(t, lock.Cache, 1)
	assert.Equal(t, "h1", lock.Cache[0].Hash)
	assert.Equal(t, 0, lock.Cache[0].ExitCode)

	// Round-trip with new fields populated.
	lock.Cache[0].Command = "echo hi"
	lock.Cache[0].LastRun = "2026-05-25T00:00:00Z"
	lock.Cache[0].ExitCode = 0
	out, err := json.Marshal(&lock)
	require.NoError(t, err)
	var lock2 DotfilesLock
	require.NoError(t, json.Unmarshal(out, &lock2))
	assert.Equal(t, "echo hi", lock2.Cache[0].Command)
	assert.Equal(t, "2026-05-25T00:00:00Z", lock2.Cache[0].LastRun)
}

// urlCacheFixture spins up a server serving one installer script and wires a
// HOME-scoped lock path, mirroring the old @install test setup.
func urlCacheFixture(t *testing.T, script string) (*Config, string) {
	t.Helper()
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, script)
	}))
	t.Cleanup(srv.Close)

	cfg := &Config{BaseDir: tmpHome, CacheEntries: []CacheEntry{{Command: srv.URL + "/install.sh"}}}
	return cfg, script
}

func TestApplyCache_URL_RunsScript(t *testing.T) {
	script := "#!/bin/sh\necho installed\n"
	cfg, script := urlCacheFixture(t, script)
	lock := &DotfilesLock{}

	require.NoError(t, ApplyCache(cfg, lock, CacheApplyOptions{}))
	require.Len(t, lock.Cache, 1)
	want := sha256.Sum256([]byte(script))
	assert.Equal(t, hex.EncodeToString(want[:]), lock.Cache[0].Hash, "identity is the script content hash")
	assert.Equal(t, cfg.CacheEntries[0].Command, lock.Cache[0].Command)
	assert.Equal(t, 0, lock.Cache[0].ExitCode)
	assert.NotEmpty(t, lock.Cache[0].LastRun)
}

func TestApplyCache_URL_SkipsWhenContentUnchanged(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	runsPath := filepath.Join(tmpHome, "runs")
	script := "#!/bin/sh\necho x >> " + runsPath + "\n"

	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		fmt.Fprint(w, script)
	}))
	t.Cleanup(srv.Close)

	cfg := &Config{BaseDir: tmpHome, CacheEntries: []CacheEntry{{Command: srv.URL + "/install.sh"}}}
	lock := &DotfilesLock{}

	require.NoError(t, ApplyCache(cfg, lock, CacheApplyOptions{}))
	require.NoError(t, ApplyCache(cfg, lock, CacheApplyOptions{}))

	// Fetched fresh both times (content may have changed upstream), but the
	// script ran only once because its content hash matched the lock.
	assert.Equal(t, 2, hits)
	data, err := os.ReadFile(runsPath)
	require.NoError(t, err)
	assert.Equal(t, 1, strings.Count(string(data), "x\n"), "second run must be a cache hit")
	assert.Len(t, lock.Cache, 1)
}

func TestApplyCache_URL_RerunsWhenContentChanges(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	runsPath := filepath.Join(tmpHome, "runs")

	scripts := []string{
		"#!/bin/sh\necho x >> " + runsPath + "\n",
		"#!/bin/sh\necho x >> " + runsPath + "\necho y >> " + runsPath + "\n",
	}
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls < len(scripts) {
			fmt.Fprint(w, scripts[calls])
		} else {
			fmt.Fprint(w, scripts[len(scripts)-1])
		}
		calls++
	}))
	t.Cleanup(srv.Close)

	cfg := &Config{BaseDir: tmpHome, CacheEntries: []CacheEntry{{Command: srv.URL + "/install.sh"}}}
	lock := &DotfilesLock{}

	require.NoError(t, ApplyCache(cfg, lock, CacheApplyOptions{}))
	require.NoError(t, ApplyCache(cfg, lock, CacheApplyOptions{}))

	data, err := os.ReadFile(runsPath)
	require.NoError(t, err)
	assert.Equal(t, 2, strings.Count(string(data), "x\n"), "upstream change must re-run the script")
	assert.Len(t, lock.Cache, 2, "old content entry stays until prune")
}

func TestIsScriptURL(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"https://example.com/install.sh", true},
		{"http://example.com/install.sh", true},
		{"echo hi", false},
		{"curl https://example.com/install.sh | sh", false},
		{"HTTPS://EXAMPLE.COM/i.sh", true},
		{"", false},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, isScriptURL(tc.in), "input %q", tc.in)
	}
}
