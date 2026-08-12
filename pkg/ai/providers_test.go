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
				"name":      "aihub",
				"provider":  "openai",
				"base_url":  "https://aihub-gateway.fbrai.dev/v1",
				"api_key":   "$AIHUB_API_KEY",
				"model":     "aihub/claude-sonnet-5",
				"weight":    10,
				"condition": `user == "yuri" && dir =~ "~/Workdir/Yuri/*"`,
				"default":   true,
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
