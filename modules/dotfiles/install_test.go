package dotfiles

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yurifrl/cly/pkg/llm"
)

// mockLLMClient returns a fixed JSON analysis response.
type mockLLMClient struct {
	response string
}

func (m *mockLLMClient) Stream(_ context.Context, _ string, _ []llm.Message) (<-chan llm.StreamChunk, error) {
	ch := make(chan llm.StreamChunk, 1)
	ch <- llm.StreamChunk{Text: m.response, Done: true}
	close(ch)
	return ch, nil
}

func (m *mockLLMClient) Complete(_ context.Context, _ string, _ []llm.Message) (string, error) {
	return m.response, nil
}

func mockAnalysisJSON(risk string, binaries []string) string {
	r := analysisResult{}
	r.Manifest.Binaries = binaries
	r.Security.Risk = risk
	r.MessageToUser = "test install"
	data, _ := json.Marshal(r)
	return string(data)
}

func TestApplyInstalls_BypassAI(t *testing.T) {
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

	require.NoError(t, ApplyInstalls(cfg, InstallOptions{BypassAI: true}))

	lockFile, _ := lockFilePath()
	lock, err := loadLock(lockFile)
	require.NoError(t, err)
	require.Len(t, lock.Installs, 1)
	assert.Equal(t, srv.URL+"/install.sh", lock.Installs[0].URL)
	assert.True(t, lock.Installs[0].Bypassed)
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
	require.NoError(t, ApplyInstalls(cfg, InstallOptions{BypassAI: true}))
	// Second run — same SHA, should skip
	require.NoError(t, ApplyInstalls(cfg, InstallOptions{BypassAI: true}))

	// Fetched twice (once per run) but script only ran once (second run skipped)
	assert.Equal(t, 2, calls)

	lockFile, _ := lockFilePath()
	lock, _ := loadLock(lockFile)
	assert.Len(t, lock.Installs, 1)
}

func TestApplyInstalls_WithLLMAnalysis(t *testing.T) {
	tmpHome := t.TempDir()
	require.NoError(t, os.Setenv("HOME", tmpHome))
	defer os.Unsetenv("HOME")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("#!/bin/sh\nmkdir -p ~/.mytool\n"))
	}))
	defer srv.Close()

	// Override LLM client and stdin
	mockResp := mockAnalysisJSON("low", []string{"~/.local/bin/mytool"})
	oldClient := newInstallLLMClient
	newInstallLLMClient = func() (llm.Client, error) {
		return &mockLLMClient{response: mockResp}, nil
	}
	defer func() { newInstallLLMClient = oldClient }()

	// Provide auto-approve via stdin pipe
	r, w, _ := os.Pipe()
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()
	w.WriteString("y\n")
	w.Close()

	cfg := &Config{
		BaseDir:  tmpHome,
		Installs: []Install{{URL: srv.URL + "/install.sh"}},
	}

	require.NoError(t, ApplyInstalls(cfg, InstallOptions{}))

	lockFile, _ := lockFilePath()
	lock, err := loadLock(lockFile)
	require.NoError(t, err)
	require.Len(t, lock.Installs, 1)
	assert.False(t, lock.Installs[0].Bypassed)
	require.NotNil(t, lock.Installs[0].Manifest)
	assert.Equal(t, []string{"~/.local/bin/mytool"}, lock.Installs[0].Manifest.Binaries)
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
