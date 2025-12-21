package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockResolver implements SecretResolver for testing walker
type mockResolver struct {
	secrets map[string]string
	errors  map[string]error
}

func (m *mockResolver) Resolve(ctx context.Context, ref string) (string, error) {
	if err, ok := m.errors[ref]; ok {
		return "", err
	}
	if val, ok := m.secrets[ref]; ok {
		return val, nil
	}
	return "", errors.New("secret not found")
}

// setupMockOpCLI creates a mock op binary for testing
func setupMockOpCLI(t *testing.T) string {
	tmpDir := t.TempDir()
	mockOpPath := filepath.Join(tmpDir, "op")

	script := `#!/bin/bash
if [[ "$2" == "op://test/item/password" ]]; then
    echo "secret123"
    exit 0
elif [[ "$2" == "op://test/item/token" ]]; then
    echo "token456"
    exit 0
elif [[ "$2" == "op://test/slow/field" ]]; then
    sleep 15
    exit 0
fi
echo "secret not found" >&2
exit 1
`
	err := os.WriteFile(mockOpPath, []byte(script), 0755)
	require.NoError(t, err)
	return mockOpPath
}

// Phase 1: OpResolver Tests

func TestOpResolver_ValidFormat(t *testing.T) {
	mockPath := setupMockOpCLI(t)
	resolver := &OpResolver{cliPath: mockPath}
	ctx := context.Background()

	_, err := resolver.Resolve(ctx, "op://test/item/password")
	assert.NoError(t, err)
}

func TestOpResolver_MissingPrefix(t *testing.T) {
	resolver := &OpResolver{cliPath: "op"}
	ctx := context.Background()

	_, err := resolver.Resolve(ctx, "test/item/password")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must start with op://")
}

func TestOpResolver_IncompletePath(t *testing.T) {
	resolver := &OpResolver{cliPath: "op"}
	ctx := context.Background()

	tests := []struct {
		name string
		ref  string
	}{
		{"only vault", "op://vault"},
		{"vault and item", "op://vault/item"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := resolver.Resolve(ctx, tt.ref)
			assert.Error(t, err)
		})
	}
}

func TestOpResolver_CLIExecution(t *testing.T) {
	mockPath := setupMockOpCLI(t)
	resolver := &OpResolver{cliPath: mockPath}
	ctx := context.Background()

	secret, err := resolver.Resolve(ctx, "op://test/item/password")
	require.NoError(t, err)
	assert.Equal(t, "secret123", secret)
}

func TestOpResolver_MultipleSecrets(t *testing.T) {
	mockPath := setupMockOpCLI(t)
	resolver := &OpResolver{cliPath: mockPath}
	ctx := context.Background()

	secret1, err := resolver.Resolve(ctx, "op://test/item/password")
	require.NoError(t, err)
	assert.Equal(t, "secret123", secret1)

	secret2, err := resolver.Resolve(ctx, "op://test/item/token")
	require.NoError(t, err)
	assert.Equal(t, "token456", secret2)
}

func TestOpResolver_Timeout(t *testing.T) {
	mockPath := setupMockOpCLI(t)
	resolver := &OpResolver{cliPath: mockPath}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := resolver.Resolve(ctx, "op://test/slow/field")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestOpResolver_CLINotFound(t *testing.T) {
	resolver := &OpResolver{cliPath: "/nonexistent/op"}
	ctx := context.Background()

	_, err := resolver.Resolve(ctx, "op://test/item/password")
	assert.Error(t, err)
}

func TestOpResolver_SecretNotFound(t *testing.T) {
	mockPath := setupMockOpCLI(t)
	resolver := &OpResolver{cliPath: mockPath}
	ctx := context.Background()

	_, err := resolver.Resolve(ctx, "op://test/missing/field")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve")
}

func TestOpResolver_NoSecretInError(t *testing.T) {
	mockPath := setupMockOpCLI(t)
	resolver := &OpResolver{cliPath: mockPath}
	ctx := context.Background()

	_, err := resolver.Resolve(ctx, "op://test/missing/field")
	require.Error(t, err)
	// Error should contain reference, not secret value
	assert.Contains(t, err.Error(), "op://test/missing/field")
}

// Phase 2: Walker Tests

func TestResolveSecretsInPlace_FlatMap(t *testing.T) {
	resolver := &mockResolver{
		secrets: map[string]string{
			"op://test/api/key": "secret123",
		},
	}

	data := map[string]interface{}{
		"api_key": "op://test/api/key",
		"url":     "https://api.example.com",
	}

	err := resolveSecretsInPlace(context.Background(), resolver, data)
	require.NoError(t, err)

	assert.Equal(t, "secret123", data["api_key"])
	assert.Equal(t, "https://api.example.com", data["url"])
}

func TestResolveSecretsInPlace_NestedMap(t *testing.T) {
	resolver := &mockResolver{
		secrets: map[string]string{
			"op://test/api/key":    "secret123",
			"op://test/api/secret": "secret456",
		},
	}

	data := map[string]interface{}{
		"service": map[string]interface{}{
			"api_key": "op://test/api/key",
			"nested": map[string]interface{}{
				"secret": "op://test/api/secret",
			},
		},
		"url": "https://api.example.com",
	}

	err := resolveSecretsInPlace(context.Background(), resolver, data)
	require.NoError(t, err)

	service := data["service"].(map[string]interface{})
	assert.Equal(t, "secret123", service["api_key"])

	nested := service["nested"].(map[string]interface{})
	assert.Equal(t, "secret456", nested["secret"])
}

func TestResolveSecretsInPlace_MixedValues(t *testing.T) {
	resolver := &mockResolver{
		secrets: map[string]string{
			"op://test/api/key": "secret123",
		},
	}

	data := map[string]interface{}{
		"api_key": "op://test/api/key",
		"enabled": true,
		"port":    8080,
		"rate":    1.5,
		"tags":    []string{"prod", "api"},
	}

	err := resolveSecretsInPlace(context.Background(), resolver, data)
	require.NoError(t, err)

	assert.Equal(t, "secret123", data["api_key"])
	assert.Equal(t, true, data["enabled"])
	assert.Equal(t, 8080, data["port"])
	assert.Equal(t, 1.5, data["rate"])
}

func TestResolveSecretsInPlace_EmptyMap(t *testing.T) {
	resolver := &mockResolver{}
	data := map[string]interface{}{}

	err := resolveSecretsInPlace(context.Background(), resolver, data)
	require.NoError(t, err)
}

func TestResolveSecretsInPlace_FailFast(t *testing.T) {
	resolver := &mockResolver{
		secrets: map[string]string{
			"op://test/api/key": "secret123",
		},
		errors: map[string]error{
			"op://test/api/missing": errors.New("secret not found"),
		},
	}

	data := map[string]interface{}{
		"api_key":    "op://test/api/key",
		"api_secret": "op://test/api/missing",
		"other_key":  "op://test/api/other",
	}

	err := resolveSecretsInPlace(context.Background(), resolver, data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "secret not found")
}

func TestResolveSecretsInPlace_NonSecretStrings(t *testing.T) {
	resolver := &mockResolver{
		secrets: map[string]string{
			"op://test/api/key": "secret123",
		},
	}

	data := map[string]interface{}{
		"api_key": "op://test/api/key",
		"url":     "https://op://looks-like-secret.com",
		"comment": "use op:// for secrets",
		"path":    "/op/data",
	}

	err := resolveSecretsInPlace(context.Background(), resolver, data)
	require.NoError(t, err)

	assert.Equal(t, "secret123", data["api_key"])
	assert.Equal(t, "https://op://looks-like-secret.com", data["url"])
	assert.Equal(t, "use op:// for secrets", data["comment"])
	assert.Equal(t, "/op/data", data["path"])
}
