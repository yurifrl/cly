package agents

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig(t *testing.T) {
	t.Run("valid yaml config", func(t *testing.T) {
		dir := t.TempDir()
		content := "ides:\n  - claude\n  - opencode\nrepos:\n  - /tmp/repo1\n  - /tmp/repo2\n"
		err := os.WriteFile(filepath.Join(dir, "agents.yaml"), []byte(content), 0644)
		require.NoError(t, err)

		cfg, err := ParseConfig(filepath.Join(dir, "agents.yaml"))
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, []string{"claude", "opencode"}, cfg.IDEs)
		assert.Equal(t, []string{"/tmp/repo1", "/tmp/repo2"}, cfg.Repos)
	})

	t.Run("missing file returns nil", func(t *testing.T) {
		cfg, err := ParseConfig("/nonexistent/agents.yaml")
		require.NoError(t, err)
		assert.Nil(t, cfg)
	})

	t.Run("empty ides uses defaults", func(t *testing.T) {
		dir := t.TempDir()
		content := "ides: []\nrepos: []\n"
		err := os.WriteFile(filepath.Join(dir, "agents.yaml"), []byte(content), 0644)
		require.NoError(t, err)

		cfg, err := ParseConfig(filepath.Join(dir, "agents.yaml"))
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, DefaultIDEs, cfg.IDEs)
		assert.Empty(t, cfg.Repos)
	})
}

func TestLoadSaveGlobalConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := &Config{
		IDEs:  []string{"claude"},
		Repos: []string{"/b/repo", "/a/repo", "/b/repo"},
	}
	require.NoError(t, SaveGlobalConfig(cfg))

	assert.FileExists(t, GlobalConfigPath())

	loaded, err := LoadGlobalConfig()
	require.NoError(t, err)
	assert.Equal(t, []string{"claude"}, loaded.IDEs)
	assert.Equal(t, []string{"/a/repo", "/b/repo"}, loaded.Repos)
}

func TestAddRepo(t *testing.T) {
	cfg := &Config{
		IDEs:  append([]string{}, DefaultIDEs...),
		Repos: []string{"/a/repo"},
	}

	added := AddRepo(cfg, "/b/repo")
	assert.True(t, added)
	assert.Equal(t, []string{"/a/repo", "/b/repo"}, cfg.Repos)

	added = AddRepo(cfg, "/a/repo")
	assert.False(t, added)
	assert.Equal(t, []string{"/a/repo", "/b/repo"}, cfg.Repos)
}

func TestGlobalPaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	assert.Equal(t, filepath.Join(home, ".config", "cly"), GlobalConfigDir())
	assert.Equal(t, filepath.Join(home, ".config", "cly", "agents.yaml"), GlobalConfigPath())
	assert.Equal(t, filepath.Join(home, ".config", "cly", "agents.pid"), PidFilePath())
	assert.Equal(t, filepath.Join(home, ".config", "cly", "agents.log"), LogFilePath())
	assert.Equal(t, filepath.Join(home, ".config", "cly", "agents.status.yaml"), StatusFilePath())
}

func TestIDEDefs(t *testing.T) {
	claude := GetIDEDef("claude")
	require.NotNil(t, claude)
	assert.Equal(t, ".claude", claude.LocalDir)
	assert.Equal(t, "CLAUDE.md", claude.AgentsMDTarget)
	assert.False(t, claude.StripAllowedTools)

	opencode := GetIDEDef("opencode")
	require.NotNil(t, opencode)
	assert.Equal(t, ".opencode", opencode.LocalDir)
	assert.Equal(t, "AGENTS.md", opencode.AgentsMDTarget)
	assert.True(t, opencode.StripAllowedTools)
	assert.Equal(t, "command", opencode.DirRenames["commands"])
	assert.Equal(t, "agent", opencode.DirRenames["agents"])
	assert.Equal(t, "skill", opencode.DirRenames["skills"])

	unknown := GetIDEDef("nonexistent")
	assert.Nil(t, unknown)
}
