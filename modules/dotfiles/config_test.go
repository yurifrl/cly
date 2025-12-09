package dotfiles

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig(t *testing.T) {
	t.Run("parses valid mapping", func(t *testing.T) {
		content := `./home/.gitconfig -> ~/.gitconfig`
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "dotfiles.conf")
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		cfg, err := ParseConfig(configPath)
		require.NoError(t, err)
		assert.Len(t, cfg.Mappings, 1)
		assert.Equal(t, filepath.Join(tmpDir, "home/.gitconfig"), cfg.Mappings[0].Source)
		assert.Contains(t, cfg.Mappings[0].Destination, ".gitconfig")
		assert.False(t, cfg.Mappings[0].IsDir)
	})

	t.Run("parses directory mapping with trailing slash", func(t *testing.T) {
		content := `./home/.config/nvim/ -> ~/.config/nvim/`
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "dotfiles.conf")
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		cfg, err := ParseConfig(configPath)
		require.NoError(t, err)
		assert.Len(t, cfg.Mappings, 1)
		assert.True(t, cfg.Mappings[0].IsDir)
	})

	t.Run("skips comments", func(t *testing.T) {
		content := `# This is a comment
./home/.gitconfig -> ~/.gitconfig
# Another comment`
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "dotfiles.conf")
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		cfg, err := ParseConfig(configPath)
		require.NoError(t, err)
		assert.Len(t, cfg.Mappings, 1)
	})

	t.Run("skips empty lines", func(t *testing.T) {
		content := `./home/.gitconfig -> ~/.gitconfig

./home/.bashrc -> ~/.bashrc`
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "dotfiles.conf")
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		cfg, err := ParseConfig(configPath)
		require.NoError(t, err)
		assert.Len(t, cfg.Mappings, 2)
	})

	t.Run("parses install commands", func(t *testing.T) {
		content := `!echo "hello world"
./home/.gitconfig -> ~/.gitconfig
!launchctl load ~/Library/LaunchAgents/foo.plist`
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "dotfiles.conf")
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		cfg, err := ParseConfig(configPath)
		require.NoError(t, err)
		assert.Len(t, cfg.Mappings, 1)
		assert.Len(t, cfg.InstallCommands, 2)
		assert.Equal(t, `echo "hello world"`, cfg.InstallCommands[0])
	})

	t.Run("reports invalid format with line number", func(t *testing.T) {
		content := `./home/.gitconfig -> ~/.gitconfig
invalid line without arrow
./home/.bashrc -> ~/.bashrc`
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "dotfiles.conf")
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		cfg, err := ParseConfig(configPath)
		require.NoError(t, err)
		assert.Len(t, cfg.Mappings, 2)
		assert.Len(t, cfg.Errors, 1)
		assert.Contains(t, cfg.Errors[0], "line 2")
	})

	t.Run("expands tilde in destination", func(t *testing.T) {
		content := `./home/.gitconfig -> ~/.gitconfig`
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "dotfiles.conf")
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		cfg, err := ParseConfig(configPath)
		require.NoError(t, err)
		home, _ := os.UserHomeDir()
		assert.Equal(t, filepath.Join(home, ".gitconfig"), cfg.Mappings[0].Destination)
	})

	t.Run("resolves relative source paths", func(t *testing.T) {
		content := `./home/.gitconfig -> ~/.gitconfig`
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "dotfiles.conf")
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		cfg, err := ParseConfig(configPath)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(tmpDir, "home/.gitconfig"), cfg.Mappings[0].Source)
	})

	t.Run("handles whitespace around arrow", func(t *testing.T) {
		content := `./home/.gitconfig   ->   ~/.gitconfig`
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "dotfiles.conf")
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		cfg, err := ParseConfig(configPath)
		require.NoError(t, err)
		assert.Len(t, cfg.Mappings, 1)
	})
}
