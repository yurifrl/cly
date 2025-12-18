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
