package agentsession

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runCmd executes a cobra command tree with args and returns captured stdout.
func runCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := &cobra.Command{Use: "cly"}
	Register(root)
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

func TestWorkflow_SaveAndVerify(t *testing.T) {
	tmpDir := t.TempDir()
	origFn := filePathFn
	filePathFn = func() string { return filepath.Join(tmpDir, "sessions.json") }
	t.Cleanup(func() { filePathFn = origFn })

	out, err := runCmd(t, "as", "save", "my-session", "test-id-123", "-d", "a test session")
	require.NoError(t, err)
	assert.Contains(t, out, `Saved claude session "my-session"`)
	assert.Contains(t, out, "test-id-123")

	sessions, err := Load(filePathFn())
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	entry := FindByNameForProvider(sessions, "claude", "my-session")
	require.NotNil(t, entry)
	assert.Equal(t, "test-id-123", entry.ID)
	assert.Equal(t, "my-session", entry.Name)
	assert.Equal(t, "claude", entry.Provider)
	assert.Equal(t, "a test session", entry.Description)
}

func TestWorkflow_SaveUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	origFn := filePathFn
	filePathFn = func() string { return filepath.Join(tmpDir, "sessions.json") }
	t.Cleanup(func() { filePathFn = origFn })

	_, err := runCmd(t, "as", "save", "proj", "id-v1")
	require.NoError(t, err)

	out, err := runCmd(t, "as", "save", "proj", "id-v2", "-d", "updated")
	require.NoError(t, err)
	assert.Contains(t, out, "id-v2")

	sessions, err := Load(filePathFn())
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
	entry := FindByNameForProvider(sessions, "claude", "proj")
	require.NotNil(t, entry)
	assert.Equal(t, "id-v2", entry.ID)
	assert.Equal(t, "updated", entry.Description)
}

func TestWorkflow_FilterByPath(t *testing.T) {
	sessions := Sessions{
		"claude:proj-a": Entry{ID: "1", Name: "proj-a", Provider: "claude", Path: "/work/a"},
		"claude:proj-b": Entry{ID: "2", Name: "proj-b", Provider: "claude", Path: "/work/b"},
		"pi:proj-c":     Entry{ID: "3", Name: "proj-c", Provider: "pi", Path: "/work/a"},
	}

	filtered := filterByPath(sessions, "/work/a")
	assert.Len(t, filtered, 2)
	assert.Contains(t, filtered, "claude:proj-a")
	assert.Contains(t, filtered, "pi:proj-c")
	assert.NotContains(t, filtered, "claude:proj-b")
}

func TestWorkflow_FindEntry(t *testing.T) {
	sessions := Sessions{
		"claude:alpha": Entry{ID: "uuid-alpha", Name: "alpha", Provider: "claude", Path: "/a"},
		"pi:beta":      Entry{ID: "uuid-beta", Name: "beta", Provider: "pi", Path: "/b"},
	}

	byName := findEntry(sessions, "claude", "alpha")
	require.NotNil(t, byName)
	assert.Equal(t, "alpha", byName.Name)

	byID := findEntry(sessions, "pi", "uuid-beta")
	require.NotNil(t, byID)
	assert.Equal(t, "beta", byID.Name)

	notFound := findEntry(sessions, "claude", "uuid-beta")
	assert.Nil(t, notFound)
}

func TestWorkflow_ResumeNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	origFn := filePathFn
	filePathFn = func() string { return filepath.Join(tmpDir, "sessions.json") }
	t.Cleanup(func() { filePathFn = origFn })

	_, err := runCmd(t, "as", "save", "existing", "id-1")
	require.NoError(t, err)

	_, err = runCmd(t, "as", "resume", "nonexistent")
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "not found"))
}

func TestWorkflow_LsEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	origFn := filePathFn
	filePathFn = func() string { return filepath.Join(tmpDir, "sessions.json") }
	t.Cleanup(func() { filePathFn = origFn })

	out, err := runCmd(t, "as", "ls")
	require.NoError(t, err)
	assert.Contains(t, out, "No saved claude sessions")
}
