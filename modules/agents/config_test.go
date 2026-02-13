package agents

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig(t *testing.T) {
	t.Run("valid jsonc config", func(t *testing.T) {
		dir := t.TempDir()
		content := `{
  // IDE list
  ides: [claude, opencode, crush]
}`
		err := os.WriteFile(filepath.Join(dir, "ai.json"), []byte(content), 0644)
		require.NoError(t, err)

		cfg, err := ParseConfig(filepath.Join(dir, "ai.json"))
		require.NoError(t, err)
		assert.Equal(t, []string{"claude", "opencode", "crush"}, cfg.IDEs)
	})

	t.Run("missing file uses defaults", func(t *testing.T) {
		cfg, err := ParseConfig("/nonexistent/ai.json")
		require.NoError(t, err)
		assert.Equal(t, DefaultIDEs, cfg.IDEs)
	})
}

func TestResolveSourceDirs(t *testing.T) {
	t.Run("global sources", func(t *testing.T) {
		dirs := ResolveSourceDirs(true)
		assert.NotEmpty(t, dirs)
		// Should include home-based paths
		for _, d := range dirs {
			assert.True(t, filepath.IsAbs(d), "should be absolute: %s", d)
		}
	})

	t.Run("local sources", func(t *testing.T) {
		dirs := ResolveSourceDirs(false)
		assert.NotEmpty(t, dirs)
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
