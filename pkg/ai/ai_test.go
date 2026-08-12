package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func listConfig(entries ...map[string]interface{}) map[string]interface{} {
	list := make([]interface{}, len(entries))
	for i, e := range entries {
		list[i] = e
	}
	return map[string]interface{}{"providers": list}
}

func TestResolveListSelection(t *testing.T) {
	global := listConfig(
		map[string]interface{}{"name": "work", "provider": "openai",
			"base_url": "https://gw.example/v1", "api_key": "$WORK_KEY",
			"model": "work-model", "weight": 10,
			"condition": `user == "` + buildContext().User + `"`},
		map[string]interface{}{"name": "personal", "provider": "anthropic",
			"model": "claude-sonnet-4-5-20250929", "default": true},
	)
	r, err := resolveE(global, nil)
	require.NoError(t, err)
	assert.Equal(t, "openai", r.Provider)
	assert.Equal(t, "work-model", r.Model)
	assert.Equal(t, "https://gw.example/v1", r.BaseURL)
	assert.Equal(t, "WORK_KEY", r.APIKeyEnv)
	require.NotNil(t, LastDecision())
	assert.Equal(t, "work", LastDecision().Picked)
}

func TestResolveDefaultFallback(t *testing.T) {
	global := listConfig(
		map[string]interface{}{"name": "nomatch", "provider": "openai",
			"model": "x", "condition": `user == "definitely-not-me-zzz"`},
		map[string]interface{}{"name": "fb", "provider": "bedrock",
			"model": "us.anthropic.claude-sonnet-4-5-20250929-v1:0", "default": true},
	)
	r, err := resolveE(global, nil)
	require.NoError(t, err)
	assert.Equal(t, "bedrock", r.Provider)
	assert.Equal(t, "us.anthropic.claude-sonnet-4-5-20250929-v1:0", r.Model)
}

func TestResolveNoAIBlockKeepsLibraryDefaults(t *testing.T) {
	r, err := resolveE(nil, nil)
	require.NoError(t, err)
	assert.Equal(t, defaultProvider, r.Provider)
	assert.Equal(t, defaultModel, r.Model)
	assert.Equal(t, defaultAPIKeyEnv, r.APIKeyEnv)
}

func TestResolveBadConfig(t *testing.T) {
	global := listConfig(map[string]interface{}{"model": "m"}) // no name
	r, err := resolveE(global, nil)
	assert.Error(t, err)
	assert.Nil(t, r)
	// LoadConfigWith-style behavior: nil on error
	assert.Nil(t, resolve(global, nil))
	// error retrievable for NewClientWith path
	require.Error(t, LastSelectionError())
}

func TestResolveModuleOverrideOnPickedEntry(t *testing.T) {
	global := listConfig(
		map[string]interface{}{"name": "only", "provider": "openai",
			"model": "base-model", "api_key": "$ONLY_KEY"},
	)
	override := map[string]interface{}{"model": "override-model"}
	r, err := resolveE(global, override)
	require.NoError(t, err)
	assert.Equal(t, "openai", r.Provider)      // provider from picked entry
	assert.Equal(t, "override-model", r.Model) // model from override
	assert.Equal(t, "ONLY_KEY", r.APIKeyEnv)   // key from picked entry
}

func TestResolveModuleOverrideSwitchesProviderType(t *testing.T) {
	// override provider: bedrock re-bases to the first bedrock entry
	global := listConfig(
		map[string]interface{}{"name": "oa", "provider": "openai", "model": "m1"},
		map[string]interface{}{"name": "br", "provider": "bedrock", "model": "m2"},
	)
	override := map[string]interface{}{"provider": "bedrock"}
	r, err := resolveE(global, override)
	require.NoError(t, err)
	assert.Equal(t, "bedrock", r.Provider)
	assert.Equal(t, "m2", r.Model)
}

func TestResolveOverrideDisabled(t *testing.T) {
	global := listConfig(map[string]interface{}{"name": "x", "provider": "openai", "model": "m"})
	r, err := resolveE(global, map[string]interface{}{"enabled": false})
	require.NoError(t, err)
	assert.Nil(t, r)
}
