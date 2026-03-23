package claudesession

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

	out, err := runCmd(t, "cs", "save", "my-session", "test-id-123", "-d", "a test session")
	require.NoError(t, err)
	assert.Contains(t, out, `Saved session "my-session"`)
	assert.Contains(t, out, "test-id-123")

	sessions, err := Load(filePathFn())
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	entry := sessions["my-session"]
	assert.Equal(t, "test-id-123", entry.ID)
	assert.Equal(t, "my-session", entry.Name)
	assert.Equal(t, "a test session", entry.Description)
}

func TestWorkflow_SaveUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	origFn := filePathFn
	filePathFn = func() string { return filepath.Join(tmpDir, "sessions.json") }
	t.Cleanup(func() { filePathFn = origFn })

	_, err := runCmd(t, "cs", "save", "proj", "id-v1")
	require.NoError(t, err)

	out, err := runCmd(t, "cs", "save", "proj", "id-v2", "-d", "updated")
	require.NoError(t, err)
	assert.Contains(t, out, "id-v2")

	sessions, err := Load(filePathFn())
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
	assert.Equal(t, "id-v2", sessions["proj"].ID)
	assert.Equal(t, "updated", sessions["proj"].Description)
}

func TestWorkflow_FilterByPath(t *testing.T) {
	sessions := Sessions{
		"proj-a": Entry{ID: "1", Name: "proj-a", Path: "/work/a"},
		"proj-b": Entry{ID: "2", Name: "proj-b", Path: "/work/b"},
		"proj-c": Entry{ID: "3", Name: "proj-c", Path: "/work/a"},
	}

	filtered := filterByPath(sessions, "/work/a")
	assert.Len(t, filtered, 2)
	assert.Contains(t, filtered, "proj-a")
	assert.Contains(t, filtered, "proj-c")
	assert.NotContains(t, filtered, "proj-b")
}

func TestWorkflow_FindEntry(t *testing.T) {
	sessions := Sessions{
		"alpha": Entry{ID: "uuid-alpha", Name: "alpha", Path: "/a"},
		"beta":  Entry{ID: "uuid-beta", Name: "beta", Path: "/b"},
	}

	byName := findEntry(sessions, "alpha")
	require.NotNil(t, byName)
	assert.Equal(t, "alpha", byName.Name)

	byID := findEntry(sessions, "uuid-beta")
	require.NotNil(t, byID)
	assert.Equal(t, "beta", byID.Name)

	notFound := findEntry(sessions, "nope")
	assert.Nil(t, notFound)
}

func TestWorkflow_ResumeNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	origFn := filePathFn
	filePathFn = func() string { return filepath.Join(tmpDir, "sessions.json") }
	t.Cleanup(func() { filePathFn = origFn })

	_, err := runCmd(t, "cs", "save", "existing", "id-1")
	require.NoError(t, err)

	_, err = runCmd(t, "cs", "resume", "nonexistent")
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "not found"))
}

func TestWorkflow_LsEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	origFn := filePathFn
	filePathFn = func() string { return filepath.Join(tmpDir, "sessions.json") }
	t.Cleanup(func() { filePathFn = origFn })

	out, err := runCmd(t, "cs", "ls")
	require.NoError(t, err)
	assert.Contains(t, out, "No saved sessions")
}

func TestWorkflow_SaveFindByID(t *testing.T) {
	tmpDir := t.TempDir()
	origFn := filePathFn
	filePathFn = func() string { return filepath.Join(tmpDir, "sessions.json") }
	t.Cleanup(func() { filePathFn = origFn })

	// Save with name "old-name" and id "abc-123"
	_, err := runCmd(t, "cs", "save", "old-name", "abc-123", "-d", "original")
	require.NoError(t, err)

	// Save with new name but same --id should find existing and rename
	out, err := runCmd(t, "cs", "save", "new-name", "--id", "abc-123", "-d", "updated")
	require.NoError(t, err)
	assert.Contains(t, out, "new-name")
	assert.Contains(t, out, "abc-123")

	sessions, err := Load(filePathFn())
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
	assert.Nil(t, FindByName(sessions, "old-name"))
	entry := FindByName(sessions, "new-name")
	require.NotNil(t, entry)
	assert.Equal(t, "abc-123", entry.ID)
	assert.Equal(t, "updated", entry.Description)
}

func TestWorkflow_SaveFindByIDNoExisting(t *testing.T) {
	tmpDir := t.TempDir()
	origFn := filePathFn
	filePathFn = func() string { return filepath.Join(tmpDir, "sessions.json") }
	t.Cleanup(func() { filePathFn = origFn })

	// --id with no existing entry should create new
	out, err := runCmd(t, "cs", "save", "fresh", "--id", "new-id-456", "-d", "brand new")
	require.NoError(t, err)
	assert.Contains(t, out, "fresh")
	assert.Contains(t, out, "new-id-456")

	sessions, err := Load(filePathFn())
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
	entry := sessions["fresh"]
	assert.Equal(t, "new-id-456", entry.ID)
	assert.Equal(t, "brand new", entry.Description)
}
