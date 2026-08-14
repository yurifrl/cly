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
		assert.Equal(t, `echo "hello world"`, cfg.InstallCommands[0].Command)
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

	t.Run("expands env vars in destination", func(t *testing.T) {
		t.Setenv("CLY_TEST_CONFIG_DIR", "/tmp/myconfig")

		content := `./home/foo.txt -> $CLY_TEST_CONFIG_DIR/foo.txt
./home/bar.txt -> ${CLY_TEST_CONFIG_DIR}/bar.txt
./home/baz.txt -> $HOME/.config/baz.txt`
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "dotfiles.conf")
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		cfg, err := ParseConfig(configPath)
		require.NoError(t, err)
		require.Len(t, cfg.Mappings, 3)
		assert.Equal(t, "/tmp/myconfig/foo.txt", cfg.Mappings[0].Destination)
		assert.Equal(t, "/tmp/myconfig/bar.txt", cfg.Mappings[1].Destination)
		home, _ := os.UserHomeDir()
		assert.Equal(t, filepath.Join(home, ".config", "baz.txt"), cfg.Mappings[2].Destination)
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

	t.Run("expands glob pattern to individual file mappings", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Create source files
		binDir := filepath.Join(tmpDir, "home", ".local", "bin")
		require.NoError(t, os.MkdirAll(binDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(binDir, "script-a"), []byte("#!/bin/sh"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(binDir, "script-b"), []byte("#!/bin/sh"), 0755))
		// Create a subdirectory too
		require.NoError(t, os.MkdirAll(filepath.Join(binDir, "subdir"), 0755))

		content := `./home/.local/bin/* -> ~/.local/bin/`
		configPath := filepath.Join(tmpDir, "dotfiles.conf")
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		cfg, err := ParseConfig(configPath)
		require.NoError(t, err)
		assert.Len(t, cfg.Mappings, 3) // script-a, script-b, subdir

		// Check individual mappings
		home, _ := os.UserHomeDir()
		names := map[string]bool{}
		for _, m := range cfg.Mappings {
			name := filepath.Base(m.Source)
			names[name] = true
			assert.Equal(t, filepath.Join(home, ".local", "bin", name), m.Destination)
		}
		assert.True(t, names["script-a"])
		assert.True(t, names["script-b"])
		assert.True(t, names["subdir"])
	})

	t.Run("glob with no matches adds warning", func(t *testing.T) {
		tmpDir := t.TempDir()
		content := `./nonexistent/* -> ~/.local/bin/`
		configPath := filepath.Join(tmpDir, "dotfiles.conf")
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		cfg, err := ParseConfig(configPath)
		require.NoError(t, err)
		assert.Len(t, cfg.Mappings, 0)
		assert.Len(t, cfg.Errors, 1)
		assert.Contains(t, cfg.Errors[0], "glob pattern matched no files")
	})
}
