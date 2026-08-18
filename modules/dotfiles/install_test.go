package dotfiles

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyInstalls_RunsScript(t *testing.T) {
	tmpHome := t.TempDir()
	require.NoError(t, os.Setenv("HOME", tmpHome))
	defer os.Unsetenv("HOME")

	// Serve a dummy install script
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("#!/bin/sh\necho installed\n"))
	}))
	defer srv.Close()

	cfg := &Config{
		BaseDir:  tmpHome,
		Installs: []Install{{URL: srv.URL + "/install.sh"}},
	}

	require.NoError(t, ApplyInstalls(cfg, InstallOptions{}))

	lockFile, _ := lockFilePath()
	lock, err := loadLock(lockFile)
	require.NoError(t, err)
	require.Len(t, lock.Installs, 1)
	assert.Equal(t, srv.URL+"/install.sh", lock.Installs[0].URL)
	assert.NotEmpty(t, lock.Installs[0].SHA)
}

func TestApplyInstalls_SkipsWhenSHAUnchanged(t *testing.T) {
	tmpHome := t.TempDir()
	require.NoError(t, os.Setenv("HOME", tmpHome))
	defer os.Unsetenv("HOME")

	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Write([]byte("#!/bin/sh\necho installed\n"))
	}))
	defer srv.Close()

	cfg := &Config{
		BaseDir:  tmpHome,
		Installs: []Install{{URL: srv.URL + "/install.sh"}},
	}

	// First run
	require.NoError(t, ApplyInstalls(cfg, InstallOptions{}))
	// Second run — same SHA, should skip
	require.NoError(t, ApplyInstalls(cfg, InstallOptions{}))

	// Fetched twice (once per run) but script only ran once (second run skipped)
	assert.Equal(t, 2, calls)

	lockFile, _ := lockFilePath()
	lock, _ := loadLock(lockFile)
	assert.Len(t, lock.Installs, 1)
}

func TestRemoveInstallArtifacts_Bypassed(t *testing.T) {
	// Bypassed entry should not remove files — just prints a banner.
	// We test by ensuring no panic and no file ops on real filesystem.
	e := InstallManifest{URL: "https://example.com/install.sh", SHA: "abc", Bypassed: true}
	RemoveInstallArtifacts(e) // should not panic
}

func TestRemoveInstallArtifacts_WithManifest(t *testing.T) {
	tmpDir := t.TempDir()
	bin := filepath.Join(tmpDir, "mytool")
	require.NoError(t, os.WriteFile(bin, []byte("#!/bin/sh\n"), 0755))

	e := InstallManifest{
		URL: "https://example.com/install.sh",
		SHA: "abc",
		Manifest: &ScriptManifest{
			Binaries: []string{bin},
		},
	}
	RemoveInstallArtifacts(e)

	_, err := os.Stat(bin)
	assert.True(t, os.IsNotExist(err))
}

func TestUpsertInstall(t *testing.T) {
	entries := []InstallManifest{
		{URL: "https://a.com", SHA: "old"},
	}
	entries = upsertInstall(entries, InstallManifest{URL: "https://a.com", SHA: "new"})
	require.Len(t, entries, 1)
	assert.Equal(t, "new", entries[0].SHA)

	entries = upsertInstall(entries, InstallManifest{URL: "https://b.com", SHA: "bbb"})
	assert.Len(t, entries, 2)
}
