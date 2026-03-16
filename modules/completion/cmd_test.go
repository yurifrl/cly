package completion

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterAlias(t *testing.T) {
	extraCompletions = nil

	RegisterAlias("zl", "complete -c zl -w 'cly zl'")
	assert.Len(t, extraCompletions, 1)
	assert.Equal(t, "zl", extraCompletions[0].Alias)
	assert.Equal(t, "complete -c zl -w 'cly zl'", extraCompletions[0].Lines)

	RegisterAlias("hy", "complete -c hy -w 'cly helpy'")
	assert.Len(t, extraCompletions, 2)

	extraCompletions = nil
}

func TestBuildExtraCompletions(t *testing.T) {
	extraCompletions = nil

	RegisterAlias("zl", "complete -c zl -f -a switch\ncomplete -c zl -w zellij")
	RegisterAlias("hy", "complete -c hy -w 'cly helpy'")

	out := BuildExtraCompletions()
	assert.Contains(t, out, "# Alias completions")
	assert.Contains(t, out, "complete -c zl -f -a switch")
	assert.Contains(t, out, "complete -c hy -w 'cly helpy'")

	extraCompletions = nil
}

func TestBuildExtraCompletionsEmpty(t *testing.T) {
	extraCompletions = nil
	lazyGenerators = nil
	out := BuildExtraCompletions()
	assert.Empty(t, out)
}

func TestRegisteredAliases(t *testing.T) {
	extraCompletions = nil
	RegisterAlias("zl", "...")
	RegisterAlias("mcp", "...")

	got := RegisteredAliases()
	assert.Equal(t, []string{"zl", "mcp"}, got)

	extraCompletions = nil
}

func TestRegisterLazy(t *testing.T) {
	extraCompletions = nil
	lazyGenerators = nil

	RegisterAlias("zl", "complete -c zl -w zellij")
	RegisterLazy(func() string {
		return "complete -c c -w 'cly claude'\ncomplete -c uuid -w 'cly uuid'\n"
	})

	out := BuildExtraCompletions()
	assert.Contains(t, out, "complete -c zl -w zellij")
	assert.Contains(t, out, "complete -c c -w 'cly claude'")
	assert.Contains(t, out, "complete -c uuid -w 'cly uuid'")

	extraCompletions = nil
	lazyGenerators = nil
}

func TestCompletionFishOutput(t *testing.T) {
	extraCompletions = nil
	RegisterAlias("zl", "complete -c zl -w 'cly zl'")

	root := &cobra.Command{Use: "testcli"}
	root.CompletionOptions.DisableDefaultCmd = true

	sub := &cobra.Command{Use: "hello", Short: "Say hello"}
	root.AddCommand(sub)

	Register(root)

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"completion", "fish"})
	err := root.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "__testcli_perform_completion")
	assert.Contains(t, output, "complete -c zl -w 'cly zl'")

	extraCompletions = nil
}

func TestCompletionFishInstall(t *testing.T) {
	extraCompletions = nil
	RegisterAlias("zl", "complete -c zl -w 'cly zl'")

	tmpDir := t.TempDir()
	completionsDir := filepath.Join(tmpDir, ".config", "fish", "completions")

	root := &cobra.Command{Use: "testcli"}
	root.CompletionOptions.DisableDefaultCmd = true
	Register(root)

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"completion", "fish", "install", "--dir", completionsDir})
	err := root.Execute()
	require.NoError(t, err)

	// Check file was written
	outFile := filepath.Join(completionsDir, "cly.fish")
	content, err := os.ReadFile(outFile)
	require.NoError(t, err)

	assert.Contains(t, string(content), "__testcli_perform_completion")
	assert.Contains(t, string(content), "complete -c zl -w 'cly zl'")

	extraCompletions = nil
}

func TestCompletionFishInstallCreatesDir(t *testing.T) {
	extraCompletions = nil

	tmpDir := t.TempDir()
	completionsDir := filepath.Join(tmpDir, "nested", "dir", "completions")

	root := &cobra.Command{Use: "testcli"}
	root.CompletionOptions.DisableDefaultCmd = true
	Register(root)

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"completion", "fish", "install", "--dir", completionsDir})
	err := root.Execute()
	require.NoError(t, err)

	outFile := filepath.Join(completionsDir, "cly.fish")
	_, err = os.Stat(outFile)
	assert.NoError(t, err)

	extraCompletions = nil
}

func TestCompletionFishCacheCreatesFile(t *testing.T) {
	extraCompletions = nil

	// Override HOME so CachePath resolves inside tmpDir
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	root := &cobra.Command{Use: "testcli"}
	root.CompletionOptions.DisableDefaultCmd = true
	sub := &cobra.Command{Use: "hello", Short: "Say hello"}
	root.AddCommand(sub)
	Register(root)

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"completion", "fish", "--cache"})
	err := root.Execute()
	require.NoError(t, err)

	// Verify cache file was created
	cacheFile := filepath.Join(tmpDir, ".cache", "fish_completions", "cly.fish")
	content, err := os.ReadFile(cacheFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "__testcli_perform_completion")

	extraCompletions = nil
}

func TestCompletionFishCacheSkipsIfExists(t *testing.T) {
	extraCompletions = nil

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Pre-create cache file with sentinel content
	cacheDir := filepath.Join(tmpDir, ".cache", "fish_completions")
	require.NoError(t, os.MkdirAll(cacheDir, 0755))
	cacheFile := filepath.Join(cacheDir, "cly.fish")
	require.NoError(t, os.WriteFile(cacheFile, []byte("# existing cache"), 0644))

	root := &cobra.Command{Use: "testcli"}
	root.CompletionOptions.DisableDefaultCmd = true
	Register(root)

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"completion", "fish", "--cache"})
	err := root.Execute()
	require.NoError(t, err)

	// Cache should NOT have been overwritten
	content, err := os.ReadFile(cacheFile)
	require.NoError(t, err)
	assert.Equal(t, "# existing cache", string(content))

	extraCompletions = nil
}

func TestClearCache(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cacheDir := filepath.Join(tmpDir, ".cache", "fish_completions")
	require.NoError(t, os.MkdirAll(cacheDir, 0755))
	cacheFile := filepath.Join(cacheDir, "cly.fish")
	require.NoError(t, os.WriteFile(cacheFile, []byte("# cache"), 0644))

	err := ClearCache()
	require.NoError(t, err)

	_, err = os.Stat(cacheFile)
	assert.True(t, os.IsNotExist(err))
}

func TestClearCacheNoFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Should not error when file doesn't exist
	err := ClearCache()
	assert.NoError(t, err)
}
