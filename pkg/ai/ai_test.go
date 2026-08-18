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

func TestResolveAIBlockWithoutProvidersKeepsLibraryDefaults(t *testing.T) {
	lastDecision = nil
	lastSelErr = nil
	r, err := resolveE(map[string]interface{}{}, nil)
	require.NoError(t, err)
	assert.Equal(t, defaultProvider, r.Provider)
	assert.Equal(t, defaultModel, r.Model)
	assert.Nil(t, LastDecision())
}

func TestResolveBadConfig(t *testing.T) {
	global := listConfig(map[string]interface{}{"model": "m"})
	r, err := resolveE(global, nil)
	assert.Error(t, err)
	assert.Nil(t, r)
	assert.Nil(t, resolve(global, nil))
	require.Error(t, LastSelectionError())
}

func TestResolveModuleOverrideOnPickedEntry(t *testing.T) {
	global := listConfig(
		map[string]interface{}{"name": "only", "provider": "openai", "model": "base-model", "api_key": "$ONLY_KEY"},
	)
	override := map[string]interface{}{"model": "override-model"}
	r, err := resolveE(global, override)
	require.NoError(t, err)
	assert.Equal(t, "openai", r.Provider)
	assert.Equal(t, "override-model", r.Model)
	assert.Equal(t, "ONLY_KEY", r.APIKeyEnv)
}

func TestResolveModuleOverrideSwitchesProviderType(t *testing.T) {
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

func TestOrderProvidersKeepsLocationMatchFirstAndIncludesAllOthers(t *testing.T) {
	ctx := buildContext()
	entries := []Entry{
		{Name: "default", Default: true},
		{Name: "low-match", Weight: 1, Condition: `user == "` + ctx.User + `"`},
		{Name: "high-match", Weight: 10, Condition: `user == "` + ctx.User + `"`},
		{Name: "unmatched", Condition: `user == "definitely-not-me-zzz"`},
	}
	for i := range entries {
		if entries[i].Condition != "" {
			cond, err := parseCondition(entries[i].Condition)
			require.NoError(t, err)
			entries[i].cond = cond
		}
	}

	ordered := orderProviders(entries, ctx)

	assert.Equal(t, []string{"high-match", "low-match", "default", "unmatched"}, entryNames(ordered))
}
