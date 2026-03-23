package agentsession

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLSOutputsJSONByDefault(t *testing.T) {
	tmpDir := t.TempDir()
	origFn := filePathFn
	filePathFn = func() string { return filepath.Join(tmpDir, "sessions.json") }
	t.Cleanup(func() { filePathFn = origFn })

	sessions := Sessions{
		"pi:save-command-ai-metadata": {
			ID:          "abc-123",
			Name:        "save-command-ai-metadata",
			Provider:    "pi",
			Path:        "/tmp/project",
			Description: "Adds AI-generated session metadata for saves.",
			SavedAt:     time.Date(2026, 3, 23, 2, 30, 0, 0, time.UTC),
		},
	}
	require.NoError(t, Save(filePathFn(), sessions))

	root := &cobra.Command{Use: "cly"}
	Register(root)
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"as", "ls", "-a", "-p", "all"})

	err := root.Execute()
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, `"name": "save-command-ai-metadata"`)
	assert.Contains(t, out, `"provider": "pi"`)
	assert.Contains(t, out, `"description": "Adds AI-generated session metadata for saves."`)
}

func TestLSFiltersByDirectoryFlag(t *testing.T) {
	tmpDir := t.TempDir()
	origFn := filePathFn
	filePathFn = func() string { return filepath.Join(tmpDir, "sessions.json") }
	t.Cleanup(func() { filePathFn = origFn })

	sessions := Sessions{
		"pi:one": {ID: "1", Name: "one", Provider: "pi", Path: "/work/a"},
		"claude:two": {ID: "2", Name: "two", Provider: "claude", Path: "/work/b"},
	}
	require.NoError(t, Save(filePathFn(), sessions))

	root := &cobra.Command{Use: "cly"}
	Register(root)
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"as", "ls", "-a", "--directory", "/work/a", "-p", "all"})

	err := root.Execute()
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, `"name": "one"`)
	assert.NotContains(t, out, `"name": "two"`)
}
