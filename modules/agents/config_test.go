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
		content := "ides:\n  - claude\n  - opencode\n  - crush\n"
		err := os.WriteFile(filepath.Join(dir, "agents.yaml"), []byte(content), 0644)
		require.NoError(t, err)

		cfg, err := ParseConfig(filepath.Join(dir, "agents.yaml"))
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, []string{"claude", "opencode", "crush"}, cfg.IDEs)
	})

	t.Run("missing file returns nil", func(t *testing.T) {
		cfg, err := ParseConfig("/nonexistent/agents.yaml")
		require.NoError(t, err)
		assert.Nil(t, cfg)
	})

	t.Run("empty ides uses defaults", func(t *testing.T) {
		dir := t.TempDir()
		content := "ides: []\n"
		err := os.WriteFile(filepath.Join(dir, "agents.yaml"), []byte(content), 0644)
		require.NoError(t, err)

		cfg, err := ParseConfig(filepath.Join(dir, "agents.yaml"))
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, DefaultIDEs, cfg.IDEs)
	})
}

func TestFindConfigFile(t *testing.T) {
	t.Run("finds agents.yaml", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "agents.yaml")
		require.NoError(t, os.WriteFile(p, []byte("ides:\n  - claude\n"), 0644))

		found := FindConfigFile([]string{dir})
		assert.Equal(t, p, found)
	})

	t.Run("returns empty when not found", func(t *testing.T) {
		dir := t.TempDir()
		found := FindConfigFile([]string{dir})
		assert.Empty(t, found)
	})
}

func TestResolveSourceDirs(t *testing.T) {
	t.Run("global sources", func(t *testing.T) {
		dirs := ResolveSourceDirs(true)
		assert.Len(t, dirs, 1)
		assert.Contains(t, dirs[0], ".agents")
	})

	t.Run("local sources", func(t *testing.T) {
		dirs := ResolveSourceDirs(false)
		assert.Len(t, dirs, 2)
		assert.Equal(t, ".agents", dirs[0])
	})
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

	crush := GetIDEDef("crush")
	require.NotNil(t, crush)

	unknown := GetIDEDef("nonexistent")
	assert.Nil(t, unknown)
}

func TestSpecialFiles(t *testing.T) {
	claude := GetIDEDef("claude")
	assert.Equal(t, "settings.json", claude.SpecialFiles["claude.json"])

	opencode := GetIDEDef("opencode")
	assert.Equal(t, "opencode.json", opencode.SpecialFiles["opencode.json"])
}
