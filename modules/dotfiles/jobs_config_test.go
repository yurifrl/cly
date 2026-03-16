package dotfiles

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig_Jobs(t *testing.T) {
	t.Run("parses startup interval and once jobs", func(t *testing.T) {
		content := `@startup claude-mem keepalive -- cd ~/.claude && node worker.js
@interval pnpm-update every=24h -- pnpm update -g -L
@once gh-dash -- gh extension install dlvhdr/gh-dash`
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "dotfiles.conf")
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		cfg, err := ParseConfig(configPath)
		require.NoError(t, err)
		require.Len(t, cfg.Jobs, 3)

		assert.Equal(t, JobRunStartup, cfg.Jobs[0].Run)
		assert.Equal(t, "claude-mem", cfg.Jobs[0].Name)
		assert.True(t, cfg.Jobs[0].KeepAlive)
		assert.Equal(t, "cd ~/.claude && node worker.js", cfg.Jobs[0].Command)

		assert.Equal(t, JobRunInterval, cfg.Jobs[1].Run)
		assert.Equal(t, "24h", cfg.Jobs[1].Every)
		assert.Equal(t, "pnpm update -g -L", cfg.Jobs[1].Command)

		assert.Equal(t, JobRunOnce, cfg.Jobs[2].Run)
		assert.Equal(t, "gh-dash", cfg.Jobs[2].Name)
	})

	t.Run("reports duplicate job names", func(t *testing.T) {
		content := `@startup same -- echo one
@once same -- echo two`
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "dotfiles.conf")
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		cfg, err := ParseConfig(configPath)
		require.NoError(t, err)
		assert.Len(t, cfg.Jobs, 1)
		require.Len(t, cfg.Errors, 1)
		assert.Contains(t, cfg.Errors[0], "duplicate job name")
	})

	t.Run("reports invalid interval syntax", func(t *testing.T) {
		content := `@interval pnpm-update 24h -- pnpm update -g -L`
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "dotfiles.conf")
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		cfg, err := ParseConfig(configPath)
		require.NoError(t, err)
		assert.Empty(t, cfg.Jobs)
		require.Len(t, cfg.Errors, 1)
		assert.Contains(t, cfg.Errors[0], "requires every=<duration>")
	})
}
