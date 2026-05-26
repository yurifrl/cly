package dotfiles

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDotfilesIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	binary := buildBinary(t)

	t.Run("sync creates symlinks", func(t *testing.T) {
		tmpDir := t.TempDir()
		sourceDir := filepath.Join(tmpDir, "source")
		destDir := filepath.Join(tmpDir, "dest")
		require.NoError(t, os.MkdirAll(sourceDir, 0755))
		require.NoError(t, os.MkdirAll(destDir, 0755))

		sourceFile := filepath.Join(sourceDir, "test.txt")
		require.NoError(t, os.WriteFile(sourceFile, []byte("content"), 0644))

		configContent := sourceFile + " -> " + filepath.Join(destDir, "test.txt")
		configPath := filepath.Join(tmpDir, "dotfiles.conf")
		require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

		cmd := exec.Command(binary, "dotfiles", "--config", configPath)
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "output: %s", output)

		destFile := filepath.Join(destDir, "test.txt")
		link, err := os.Readlink(destFile)
		require.NoError(t, err)
		assert.Equal(t, sourceFile, link)
	})

	t.Run("sync skips install commands without -i flag", func(t *testing.T) {
		tmpDir := t.TempDir()
		configContent := `!echo "should not run"`
		configPath := filepath.Join(tmpDir, "dotfiles.conf")
		require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

		cmd := exec.Command(binary, "dotfiles", "--config", configPath)
		output, err := cmd.CombinedOutput()
		require.NoError(t, err)
		assert.Contains(t, string(output), "skipped")
		assert.NotContains(t, string(output), "should not run")
	})

	t.Run("sync runs install commands with -i flag", func(t *testing.T) {
		tmpDir := t.TempDir()
		markerFile := filepath.Join(tmpDir, "marker.txt")
		configContent := `!touch ` + markerFile
		configPath := filepath.Join(tmpDir, "dotfiles.conf")
		require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

		cmd := exec.Command(binary, "dotfiles", "-i", "--config", configPath)
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "output: %s", output)

		_, err = os.Stat(markerFile)
		assert.NoError(t, err, "marker file should exist")
	})

	t.Run("status shows mapping states", func(t *testing.T) {
		tmpDir := t.TempDir()
		sourceFile := filepath.Join(tmpDir, "source.txt")
		destFile := filepath.Join(tmpDir, "dest.txt")
		require.NoError(t, os.WriteFile(sourceFile, []byte("content"), 0644))
		require.NoError(t, os.Symlink(sourceFile, destFile))

		configContent := sourceFile + " -> " + destFile
		configPath := filepath.Join(tmpDir, "dotfiles.conf")
		require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

		cmd := exec.Command(binary, "dotfiles", "status", "--config", configPath)
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "output: %s", output)
		assert.Contains(t, string(output), "✓")
	})

	t.Run("unlink removes symlinks", func(t *testing.T) {
		tmpDir := t.TempDir()
		sourceFile := filepath.Join(tmpDir, "source.txt")
		destFile := filepath.Join(tmpDir, "dest.txt")
		require.NoError(t, os.WriteFile(sourceFile, []byte("content"), 0644))
		require.NoError(t, os.Symlink(sourceFile, destFile))

		configContent := sourceFile + " -> " + destFile
		configPath := filepath.Join(tmpDir, "dotfiles.conf")
		require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

		cmd := exec.Command(binary, "dotfiles", "unlink", "--config", configPath)
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "output: %s", output)

		_, err = os.Lstat(destFile)
		assert.True(t, os.IsNotExist(err), "symlink should be removed")
	})

	t.Run("errors on missing config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "nonexistent.conf")

		cmd := exec.Command(binary, "dotfiles", "--config", configPath)
		output, _ := cmd.CombinedOutput()
		assert.Contains(t, string(output), "config not found")
	})

	t.Run("force overrides existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		sourceFile := filepath.Join(tmpDir, "source.txt")
		destFile := filepath.Join(tmpDir, "dest.txt")
		require.NoError(t, os.WriteFile(sourceFile, []byte("source"), 0644))
		require.NoError(t, os.WriteFile(destFile, []byte("existing"), 0644))

		configContent := sourceFile + " -> " + destFile
		configPath := filepath.Join(tmpDir, "dotfiles.conf")
		require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

		cmd := exec.Command(binary, "dotfiles", "--config", configPath)
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "output: %s", output)

		// Should have created symlink, overriding existing file
		link, err := os.Readlink(destFile)
		require.NoError(t, err)
		assert.Equal(t, sourceFile, link)
	})
}

func TestParseConfig_RemovedDirectivesAreErrors(t *testing.T) {
	content := `@startup foo -- echo
@interval foo every=1h -- echo
@once foo -- echo`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "dotfiles.conf")
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	cfg, err := ParseConfig(configPath)
	require.NoError(t, err)
	assert.Empty(t, cfg.CacheEntries)
	require.Len(t, cfg.Errors, 3)
	assert.Contains(t, cfg.Errors[0], "@startup is removed; migrate background processes to process-compose.yaml")
	assert.Contains(t, cfg.Errors[1], "@interval is removed; migrate scheduled tasks to process-compose.yaml")
	assert.Contains(t, cfg.Errors[2], "@once is removed; use @cache instead")
}

func TestParseConfig_CacheNewForm(t *testing.T) {
	content := "@cache echo hello\n@cache git pull --ff-only\n"
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "dotfiles.conf")
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	cfg, err := ParseConfig(configPath)
	require.NoError(t, err)
	assert.Empty(t, cfg.Errors)
	require.Len(t, cfg.CacheEntries, 2)
	assert.Equal(t, "echo hello", cfg.CacheEntries[0].Command)
	assert.Equal(t, "git pull --ff-only", cfg.CacheEntries[1].Command)
}

func TestParseConfig_CacheLegacyFormRejected(t *testing.T) {
	content := "@cache foo -- echo hi\n"
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "dotfiles.conf")
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	cfg, err := ParseConfig(configPath)
	require.NoError(t, err)
	assert.Empty(t, cfg.CacheEntries)
	require.Len(t, cfg.Errors, 1)
	assert.Contains(t, cfg.Errors[0], "@cache no longer takes a name; use '@cache <command>' (the command itself is the identity)")
}

func TestParseConfig_CacheEmptyCommand(t *testing.T) {
	content := "@cache\n"
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "dotfiles.conf")
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	cfg, err := ParseConfig(configPath)
	require.NoError(t, err)
	require.Len(t, cfg.Errors, 1)
	assert.Contains(t, cfg.Errors[0], "@cache requires a command")
}

func buildBinary(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "cly")
	cmd := exec.Command("go", "build", "-o", binary, "../../.")
	cmd.Dir = filepath.Join(mustGetWd(t), "modules", "dotfiles")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to build: %s", output)
	return binary
}

func mustGetWd(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	for !strings.HasSuffix(wd, "cly") && wd != "/" {
		wd = filepath.Dir(wd)
	}
	return wd
}
