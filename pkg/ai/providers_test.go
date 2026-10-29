package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseProviders(t *testing.T) {
	global := map[string]interface{}{
		"providers": []interface{}{
			map[string]interface{}{
				"name":       "aihub",
				"provider":   "openai",
				"base_url":   "https://aihub-gateway.fbrai.dev/v1",
				"api_key":    "$AIHUB_API_KEY",
				"model":      "aihub/claude-sonnet-5",
				"weight":     10,
				"max_tokens": 4000,
				"condition":  `user == "yuri" && dir =~ "~/Workdir/Yuri/*"`,
				"default":    true,
			},
			map[string]interface{}{
				"name":     "bedrock",
				"provider": "bedrock",
				"model":    "us.anthropic.claude-sonnet-4-5-20250929-v1:0",
			},
		},
	}
	entries, err := parseProviders(global)
	require.NoError(t, err)
	require.Len(t, entries, 2)

	assert.Equal(t, "aihub", entries[0].Name)
	assert.Equal(t, "openai", entries[0].Provider)
	assert.Equal(t, "https://aihub-gateway.fbrai.dev/v1", entries[0].BaseURL)
	assert.Equal(t, "", entries[0].APIKey)
	assert.Equal(t, "AIHUB_API_KEY", entries[0].APIKeyEnv) // $ENV expanded to env name
	assert.Equal(t, "aihub/claude-sonnet-5", entries[0].Model)
	assert.Equal(t, 10, entries[0].Weight)
	assert.Equal(t, 4000, entries[0].MaxTokens)
	assert.Equal(t, 0, entries[1].MaxTokens)
	assert.True(t, entries[0].Default)
	require.NotNil(t, entries[0].cond)

	assert.Equal(t, "bedrock", entries[1].Name)
	assert.Equal(t, 0, entries[1].Weight)
	assert.False(t, entries[1].Default)
	assert.Nil(t, entries[1].cond)
}

func TestParseProvidersDefaults(t *testing.T) {
	// entry with no provider type defaults to anthropic; no api_key gets
	// the provider's conventional env var name
	global := map[string]interface{}{
		"providers": []interface{}{
			map[string]interface{}{"name": "x", "model": "m"},
		},
	}
	entries, err := parseProviders(global)
	require.NoError(t, err)
	assert.Equal(t, "anthropic", entries[0].Provider)
	assert.Equal(t, "ANTHROPIC_API_KEY", entries[0].APIKeyEnv)
}

func TestParseProvidersErrors(t *testing.T) {
	tests := []struct {
		name   string
		global map[string]interface{}
	}{
		{"empty list", map[string]interface{}{"providers": []interface{}{}}},
		{"missing name", map[string]interface{}{"providers": []interface{}{
			map[string]interface{}{"model": "m"}}}},
		{"duplicate name", map[string]interface{}{"providers": []interface{}{
			map[string]interface{}{"name": "x"},
			map[string]interface{}{"name": "x"}}}},
		{"bad condition", map[string]interface{}{"providers": []interface{}{
			map[string]interface{}{"name": "x", "condition": "user =="}}}},
		{"unknown condition field", map[string]interface{}{"providers": []interface{}{
			map[string]interface{}{"name": "x", "condition": `foo == "bar"`}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseProviders(tt.global)
			assert.Error(t, err)
		})
	}
}

func TestParseProvidersTypes(t *testing.T) {
	global := map[string]interface{}{
		"providers": []interface{}{
			map[string]interface{}{"name": "chat1", "model": "m"},
			map[string]interface{}{"name": "jev1", "model": "j", "type": "jev"},
			map[string]interface{}{"name": "chat2", "model": "m2", "type": "chat"},
		},
	}
	entries, err := parseProviders(global)
	require.NoError(t, err)
	assert.Equal(t, TypeChat, entries[0].Type)
	assert.Equal(t, TypeJev, entries[1].Type)
	assert.Equal(t, TypeChat, entries[2].Type)

	chat := providersOfType(entries, TypeChat)
	require.Len(t, chat, 2)
	assert.Equal(t, "chat1", chat[0].Name)
	assert.Equal(t, "chat2", chat[1].Name)

	jev := providersOfType(entries, TypeJev)
	require.Len(t, jev, 1)
	assert.Equal(t, "jev1", jev[0].Name)
}

func TestDecisionsEndpointsIsolation(t *testing.T) {
	// Chat entries and broken jev entries must never yield a jev endpoint.
	global := map[string]interface{}{
		"providers": []interface{}{
			map[string]interface{}{
				"name": "chatter", "model": "big-model", "type": "chat",
				"base_url": "https://chat.example", "api_key": "k",
			},
			map[string]interface{}{
				"name": "jev-good", "model": "aihub/jev-latest", "type": "jev",
				"base_url": "https://jev.example/", "api_key": "jk",
			},
			map[string]interface{}{
				"name": "jev-nokey", "model": "aihub/jev-latest", "type": "jev",
				"base_url": "https://jev2.example", "api_key": "$CLY_TEST_UNSET_KEY",
			},
		},
	}
	entries, err := parseProviders(global)
	require.NoError(t, err)
	eps := decisionsEndpoints(entries, selCtx())
	require.Len(t, eps, 1)
	assert.Equal(t, "jev-good", eps[0].name)
	assert.Equal(t, "https://jev.example", eps[0].baseURL) // trailing / trimmed
	assert.Equal(t, "jk", eps[0].apiKey)
}
