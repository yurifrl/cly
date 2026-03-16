package llm

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_Anthropic(t *testing.T) {
	// Set a fake key for testing client creation
	t.Setenv("ANTHROPIC_API_KEY", "test-key-123")

	client, err := NewClient(Config{
		Provider: ProviderAnthropic,
		Model:    "claude-sonnet-4-20250514",
	})

	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewClient_OpenAI(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key-123")

	client, err := NewClient(Config{
		Provider: ProviderOpenAI,
		Model:    "gpt-4o",
	})

	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewClient_UnknownProvider(t *testing.T) {
	client, err := NewClient(Config{
		Provider: "unknown",
		APIKey:   "test-key",
	})

	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "unknown provider")
}

func TestNewClient_NoAPIKey(t *testing.T) {
	// Ensure env vars are unset
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Unsetenv("OPENAI_API_KEY")

	client, err := NewClient(Config{
		Provider: ProviderAnthropic,
	})

	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "no API key")
}

func TestResolveAPIKey_DirectConfig(t *testing.T) {
	key := resolveAPIKey(Config{
		Provider: ProviderAnthropic,
		APIKey:   "direct-key",
	})
	assert.Equal(t, "direct-key", key)
}

func TestResolveAPIKey_CustomEnv(t *testing.T) {
	t.Setenv("MY_CUSTOM_KEY", "custom-env-key")

	key := resolveAPIKey(Config{
		Provider:  ProviderAnthropic,
		APIKeyEnv: "MY_CUSTOM_KEY",
	})
	assert.Equal(t, "custom-env-key", key)
}

func TestResolveAPIKey_DefaultEnv(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "default-anthropic-key")

	key := resolveAPIKey(Config{
		Provider: ProviderAnthropic,
	})
	assert.Equal(t, "default-anthropic-key", key)
}

func TestResolveAPIKey_Priority(t *testing.T) {
	// Direct key takes priority over env vars
	t.Setenv("ANTHROPIC_API_KEY", "env-key")

	key := resolveAPIKey(Config{
		Provider: ProviderAnthropic,
		APIKey:   "direct-key",
	})
	assert.Equal(t, "direct-key", key)
}
