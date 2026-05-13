package dotfiles

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig_Op(t *testing.T) {
	t.Run("parses @op mapping", func(t *testing.T) {
		content := `@op ./home/.env.op -> ~/.env`
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "dotfiles.conf")
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		cfg, err := ParseConfig(configPath)
		require.NoError(t, err)
		require.Len(t, cfg.OpMappings, 1)
		assert.Equal(t, filepath.Join(tmpDir, "home/.env.op"), cfg.OpMappings[0].Source)
		assert.Contains(t, cfg.OpMappings[0].Destination, ".env")
		assert.Empty(t, cfg.OpMappings[0].Account)
	})

	t.Run("parses @op with account override", func(t *testing.T) {
		content := `@op account=nsx-team.1password.com ./home/.env.op -> ~/.env`
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "dotfiles.conf")
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		cfg, err := ParseConfig(configPath)
		require.NoError(t, err)
		require.Len(t, cfg.OpMappings, 1)
		assert.Equal(t, "nsx-team.1password.com", cfg.OpMappings[0].Account)
		assert.Equal(t, filepath.Join(tmpDir, "home/.env.op"), cfg.OpMappings[0].Source)
	})

	t.Run("reports error for missing arrow", func(t *testing.T) {
		content := `@op ./home/.env.op`
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "dotfiles.conf")
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		cfg, err := ParseConfig(configPath)
		require.NoError(t, err)
		assert.Empty(t, cfg.OpMappings)
		require.Len(t, cfg.Errors, 1)
		assert.Contains(t, cfg.Errors[0], "requires source -> destination")
	})

	t.Run("parses multiple @op mappings", func(t *testing.T) {
		content := `@op ./home/.env.op -> ~/.env
@op account=my.1password.com ./home/.aicommits -> ~/.aicommits`
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "dotfiles.conf")
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		cfg, err := ParseConfig(configPath)
		require.NoError(t, err)
		require.Len(t, cfg.OpMappings, 2)
		assert.Empty(t, cfg.OpMappings[0].Account)
		assert.Equal(t, "my.1password.com", cfg.OpMappings[1].Account)
	})

	t.Run("parses @op with op:// secret reference", func(t *testing.T) {
		content := `@op account=my.1password.com op://Private/Item/field.json -> ~/.config/thing.json`
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "dotfiles.conf")
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		cfg, err := ParseConfig(configPath)
		require.NoError(t, err)
		require.Len(t, cfg.OpMappings, 1)
		m := cfg.OpMappings[0]
		assert.True(t, m.IsReference)
		assert.Equal(t, "op://Private/Item/field.json", m.Source)
		assert.Equal(t, "my.1password.com", m.Account)
		assert.Contains(t, m.Destination, ".config/thing.json")
	})

	t.Run("parses @op with quoted op:// reference containing spaces", func(t *testing.T) {
		content := `@op "op://Private/Some Item/field" -> ~/out.txt`
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "dotfiles.conf")
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		cfg, err := ParseConfig(configPath)
		require.NoError(t, err)
		require.Len(t, cfg.OpMappings, 1)
		m := cfg.OpMappings[0]
		assert.True(t, m.IsReference)
		assert.Equal(t, "op://Private/Some Item/field", m.Source)
	})
}
