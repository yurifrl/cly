package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 3: Integration Tests

func TestLoad_WithoutSecrets(t *testing.T) {
	// Test that existing Load() behavior unchanged
	cfg, err := Load()
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "cly", cfg.App.Name)
}

func TestLoad_WithSecrets(t *testing.T) {
	// Create mock op binary
	tmpDir := t.TempDir()
	mockOpPath := filepath.Join(tmpDir, "op")
	script := `#!/bin/bash
if [[ "$2" == "op://test/backup/token" ]]; then
    echo "backup-token-123"
    exit 0
fi
exit 1
`
	err := os.WriteFile(mockOpPath, []byte(script), 0755)
	require.NoError(t, err)

	// Create temp config with secrets
	configDir := filepath.Join(tmpDir, ".config", "cly")
	err = os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	configContent := `app:
  name: cly
  debug: false

theme:
  style: charm

modules:
  backup:
    token: op://test/backup/token
    enabled: true
`
	configPath := filepath.Join(configDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Set HOME to temp dir and change to temp dir
	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", tmpDir)

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Temporarily replace global resolver for testing
	originalNewOpResolver := newOpResolverFunc
	defer func() { newOpResolverFunc = originalNewOpResolver }()
	newOpResolverFunc = func() *OpResolver {
		return &OpResolver{cliPath: mockOpPath}
	}

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify backup module exists
	backup, exists := cfg.Modules["backup"]
	require.True(t, exists, "backup module should exist in config")
	require.NotNil(t, backup)

	// Verify secret was resolved
	token, ok := backup["token"].(string)
	require.True(t, ok, "token should be string, got %T", backup["token"])
	assert.Equal(t, "backup-token-123", token)

	// Verify non-secret values unchanged
	enabled, ok := backup["enabled"].(bool)
	require.True(t, ok)
	assert.Equal(t, true, enabled)
}

func TestLoad_SecretResolutionFailure(t *testing.T) {
	// Create mock op binary that always fails
	tmpDir := t.TempDir()
	mockOpPath := filepath.Join(tmpDir, "op")
	script := `#!/bin/bash
echo "secret not found" >&2
exit 1
`
	err := os.WriteFile(mockOpPath, []byte(script), 0755)
	require.NoError(t, err)

	// Create temp config with invalid secret
	configDir := filepath.Join(tmpDir, ".config", "cly")
	err = os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	configContent := `app:
  name: cly

modules:
  backup:
    token: op://test/missing/token
`
	configPath := filepath.Join(configDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Set HOME to temp dir and change to temp dir
	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", tmpDir)

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Temporarily replace global resolver
	originalNewOpResolver := newOpResolverFunc
	defer func() { newOpResolverFunc = originalNewOpResolver }()
	newOpResolverFunc = func() *OpResolver {
		return &OpResolver{cliPath: mockOpPath}
	}

	_, err = Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve secret")
	// Verify no secret values in error
	assert.NotContains(t, err.Error(), "backup-token")
}

func TestLoad_ExpandsEnvVarsInAppAndModules(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "cly")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	configContent := `app:
  name: cly
  config_dir: $HOME/.config/cly
  data_dir: ${HOME}/.local/share/cly
  dotfiles_dir: $HOME/DotFiles

modules:
  bundle:
    go_file: $HOME/.config/cly/bundles/Gofile
    paths:
      - $HOME/foo
      - ${HOME}/bar
  memwatch:
    message: "Free ${FREE_MEM}%"
`
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0644))

	t.Setenv("HOME", tmpDir)
	t.Setenv("FREE_MEM", "42")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, tmpDir+"/.config/cly", cfg.App.ConfigDir)
	assert.Equal(t, tmpDir+"/.local/share/cly", cfg.App.DataDir)
	assert.Equal(t, tmpDir+"/DotFiles", cfg.App.DotFilesDir)

	bundle := cfg.Modules["bundle"]
	require.NotNil(t, bundle)
	assert.Equal(t, tmpDir+"/.config/cly/bundles/Gofile", bundle["go_file"])

	paths, ok := bundle["paths"].([]interface{})
	require.True(t, ok, "paths should be a slice, got %T", bundle["paths"])
	assert.Equal(t, tmpDir+"/foo", paths[0])
	assert.Equal(t, tmpDir+"/bar", paths[1])

	memwatch := cfg.Modules["memwatch"]
	require.NotNil(t, memwatch)
	assert.Equal(t, "Free 42%", memwatch["message"])

	// Reset global so subsequent tests do not see the override.
	globalConfig = nil
}

func TestLoad_ExpandsTildeInPaths(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "cly")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	configContent := `app:
  name: cly
  config_dir: ~/.config/cly
  data_dir: ~/.local/share/cly
  dotfiles_dir: ~/DotFiles
`
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0644))

	t.Setenv("HOME", tmpDir)

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(tmpDir, ".config", "cly"), cfg.App.ConfigDir)
	assert.Equal(t, filepath.Join(tmpDir, ".local", "share", "cly"), cfg.App.DataDir)
	assert.Equal(t, filepath.Join(tmpDir, "DotFiles"), cfg.App.DotFilesDir)

	globalConfig = nil
}

func TestGetString_ExpandsEnvVars(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "cly")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	configContent := `app:
  data_dir: $HOME/.local/share/cly
`
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0644))

	t.Setenv("HOME", tmpDir)

	got := GetString("app.data_dir")
	assert.Equal(t, tmpDir+"/.local/share/cly", got)
}
