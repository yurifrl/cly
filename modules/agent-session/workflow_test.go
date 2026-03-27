package agentsession

import (
	"bytes"
	"encoding/json"
	"path/filepath"
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

func TestWorkflow_UpsertAndVerify(t *testing.T) {
	tmpDir := t.TempDir()
	origFn := filePathFn
	filePathFn = func() string { return filepath.Join(tmpDir, "sessions.json") }
	t.Cleanup(func() { filePathFn = origFn })

	out, err := runCmd(t, "as", "upsert", "test-id-123", "my-session", "a test session")
	require.NoError(t, err)

	// Output should be JSON
	var result Entry
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, "test-id-123", result.ID)
	assert.Equal(t, "my-session", result.Name)
	assert.Equal(t, "claude", result.Provider)
	assert.Equal(t, "a test session", result.Description)

	sessions, err := Load(filePathFn())
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	entry := FindByNameForProvider(sessions, "claude", "my-session")
	require.NotNil(t, entry)
	assert.Equal(t, "test-id-123", entry.ID)
}

func TestWorkflow_UpsertUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	origFn := filePathFn
	filePathFn = func() string { return filepath.Join(tmpDir, "sessions.json") }
	t.Cleanup(func() { filePathFn = origFn })

	_, err := runCmd(t, "as", "upsert", "id-v1", "proj")
	require.NoError(t, err)

	out, err := runCmd(t, "as", "upsert", "id-v2", "proj", "updated")
	require.NoError(t, err)

	var result Entry
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, "id-v2", result.ID)
	assert.Equal(t, "updated", result.Description)

	sessions, err := Load(filePathFn())
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
}

func TestWorkflow_UpsertWithMeta(t *testing.T) {
	tmpDir := t.TempDir()
	origFn := filePathFn
	filePathFn = func() string { return filepath.Join(tmpDir, "sessions.json") }
	t.Cleanup(func() { filePathFn = origFn })

	out, err := runCmd(t, "as", "upsert", "id-1", "--name", "proj", "--set", "env=prod", "--set", "team=infra")
	require.NoError(t, err)

	var result Entry
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, "prod", result.Meta["env"])
	assert.Equal(t, "infra", result.Meta["team"])
}

func TestWorkflow_UpsertWithMetaJSON(t *testing.T) {
	tmpDir := t.TempDir()
	origFn := filePathFn
	filePathFn = func() string { return filepath.Join(tmpDir, "sessions.json") }
	t.Cleanup(func() { filePathFn = origFn })

	out, err := runCmd(t, "as", "upsert", "id-1", "--name", "proj", "--meta", `{"env":"staging","region":"us-east"}`)
	require.NoError(t, err)

	var result Entry
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, "staging", result.Meta["env"])
	assert.Equal(t, "us-east", result.Meta["region"])
}

func TestWorkflow_UpsertMetaMerge(t *testing.T) {
	tmpDir := t.TempDir()
	origFn := filePathFn
	filePathFn = func() string { return filepath.Join(tmpDir, "sessions.json") }
	t.Cleanup(func() { filePathFn = origFn })

	// First upsert with some meta
	_, err := runCmd(t, "as", "upsert", "id-1", "--name", "proj", "--set", "env=prod", "--set", "team=infra")
	require.NoError(t, err)

	// Second upsert: update env, add new key, team preserved
	out, err := runCmd(t, "as", "upsert", "id-1", "--set", "env=staging", "--set", "version=2")
	require.NoError(t, err)

	var result Entry
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, "staging", result.Meta["env"])
	assert.Equal(t, "infra", result.Meta["team"])
	assert.Equal(t, "2", result.Meta["version"])
}

func TestWorkflow_UpsertByIDOnly(t *testing.T) {
	tmpDir := t.TempDir()
	origFn := filePathFn
	filePathFn = func() string { return filepath.Join(tmpDir, "sessions.json") }
	t.Cleanup(func() { filePathFn = origFn })

	// Create with just ID
	out, err := runCmd(t, "as", "upsert", "id-only")
	require.NoError(t, err)

	var result Entry
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, "id-only", result.ID)
	assert.Equal(t, "claude", result.Provider)
	assert.NotEmpty(t, result.Path)
}

func TestWorkflow_SaveAliasWorks(t *testing.T) {
	tmpDir := t.TempDir()
	origFn := filePathFn
	filePathFn = func() string { return filepath.Join(tmpDir, "sessions.json") }
	t.Cleanup(func() { filePathFn = origFn })

	out, err := runCmd(t, "as", "save", "id-alias", "my-session")
	require.NoError(t, err)

	var result Entry
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, "id-alias", result.ID)
	assert.Equal(t, "my-session", result.Name)
}

func TestWorkflow_RmByName(t *testing.T) {
	tmpDir := t.TempDir()
	origFn := filePathFn
	filePathFn = func() string { return filepath.Join(tmpDir, "sessions.json") }
	t.Cleanup(func() { filePathFn = origFn })

	_, err := runCmd(t, "as", "upsert", "id-1", "to-delete")
	require.NoError(t, err)

	out, err := runCmd(t, "as", "rm", "to-delete")
	require.NoError(t, err)
	assert.Contains(t, out, `"deleted"`)
	assert.Contains(t, out, `"to-delete"`)

	sessions, err := Load(filePathFn())
	require.NoError(t, err)
	assert.Len(t, sessions, 0)
}

func TestWorkflow_RmByFilter(t *testing.T) {
	tmpDir := t.TempDir()
	origFn := filePathFn
	filePathFn = func() string { return filepath.Join(tmpDir, "sessions.json") }
	t.Cleanup(func() { filePathFn = origFn })

	_, err := runCmd(t, "as", "upsert", "id-1", "old-deploy-1")
	require.NoError(t, err)
	_, err = runCmd(t, "as", "upsert", "id-2", "old-deploy-2")
	require.NoError(t, err)
	_, err = runCmd(t, "as", "upsert", "id-3", "new-session")
	require.NoError(t, err)

	out, err := runCmd(t, "as", "rm", "--filter", "old-deploy")
	require.NoError(t, err)
	assert.Contains(t, out, `"deleted"`)

	sessions, err := Load(filePathFn())
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
	assert.NotNil(t, FindByNameForProvider(sessions, "claude", "new-session"))
}

func TestWorkflow_RmDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	origFn := filePathFn
	filePathFn = func() string { return filepath.Join(tmpDir, "sessions.json") }
	t.Cleanup(func() { filePathFn = origFn })

	_, err := runCmd(t, "as", "upsert", "id-1", "target-session")
	require.NoError(t, err)

	out, err := runCmd(t, "as", "rm", "--filter", "target", "--dry-run")
	require.NoError(t, err)
	assert.Contains(t, out, `"would_delete"`)
	assert.Contains(t, out, `"target-session"`)

	// Session should still exist
	sessions, err := Load(filePathFn())
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
}

func TestWorkflow_RmErrorBothNameAndFilter(t *testing.T) {
	tmpDir := t.TempDir()
	origFn := filePathFn
	filePathFn = func() string { return filepath.Join(tmpDir, "sessions.json") }
	t.Cleanup(func() { filePathFn = origFn })

	_, err := runCmd(t, "as", "rm", "some-name", "--filter", "some-filter")
	require.Error(t, err)
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

	_, err := runCmd(t, "as", "upsert", "id-1", "existing")
	require.NoError(t, err)

	_, err = runCmd(t, "as", "resume", "nonexistent")
	require.Error(t, err)
}

func TestWorkflow_LsEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	origFn := filePathFn
	filePathFn = func() string { return filepath.Join(tmpDir, "sessions.json") }
	t.Cleanup(func() { filePathFn = origFn })

	out, err := runCmd(t, "as", "ls")
	require.NoError(t, err)
	assert.Contains(t, out, "[]")
}

func TestWorkflow_LsWithFilter(t *testing.T) {
	tmpDir := t.TempDir()
	origFn := filePathFn
	filePathFn = func() string { return filepath.Join(tmpDir, "sessions.json") }
	t.Cleanup(func() { filePathFn = origFn })

	_, err := runCmd(t, "as", "upsert", "id-1", "deploy-prod")
	require.NoError(t, err)
	_, err = runCmd(t, "as", "upsert", "id-2", "deploy-staging")
	require.NoError(t, err)
	_, err = runCmd(t, "as", "upsert", "id-3", "unrelated")
	require.NoError(t, err)

	out, err := runCmd(t, "as", "ls", "-a", "--filter", "deploy")
	require.NoError(t, err)
	assert.Contains(t, out, "deploy-prod")
	assert.Contains(t, out, "deploy-staging")
	assert.NotContains(t, out, "unrelated")
}

func TestWorkflow_UpsertNameNotOverriddenByDefault(t *testing.T) {
	tmpDir := t.TempDir()
	origFn := filePathFn
	filePathFn = func() string { return filepath.Join(tmpDir, "sessions.json") }
	t.Cleanup(func() { filePathFn = origFn })

	// Create entry with name "original"
	_, err := runCmd(t, "as", "upsert", "id-1", "--name", "original")
	require.NoError(t, err)

	// Update same ID with a different name — should NOT change the name
	out, err := runCmd(t, "as", "upsert", "id-1", "--name", "renamed")
	require.NoError(t, err)

	var result Entry
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, "original", result.Name, "name should not change without --override")
}

func TestWorkflow_UpsertNameOverriddenWithFlag(t *testing.T) {
	tmpDir := t.TempDir()
	origFn := filePathFn
	filePathFn = func() string { return filepath.Join(tmpDir, "sessions.json") }
	t.Cleanup(func() { filePathFn = origFn })

	// Create entry with name "original"
	_, err := runCmd(t, "as", "upsert", "id-1", "--name", "original")
	require.NoError(t, err)

	// Update same ID with --override — should change the name
	out, err := runCmd(t, "as", "upsert", "id-1", "--name", "renamed", "--override")
	require.NoError(t, err)

	var result Entry
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, "renamed", result.Name, "name should change with --override")
}

func TestWorkflow_UpsertNameSetWhenEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	origFn := filePathFn
	filePathFn = func() string { return filepath.Join(tmpDir, "sessions.json") }
	t.Cleanup(func() { filePathFn = origFn })

	// Create entry with NO name (ID only)
	_, err := runCmd(t, "as", "upsert", "id-1")
	require.NoError(t, err)

	// Update same ID with --name — should set the name (no --override needed)
	out, err := runCmd(t, "as", "upsert", "id-1", "--name", "now-named")
	require.NoError(t, err)

	var result Entry
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, "now-named", result.Name, "name should be set when entry had no name")
}
