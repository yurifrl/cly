package agentsession

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterByName(t *testing.T) {
	sessions := Sessions{
		"claude:deploy-prod":    Entry{ID: "1", Name: "deploy-prod", Provider: "claude"},
		"claude:deploy-staging": Entry{ID: "2", Name: "deploy-staging", Provider: "claude"},
		"pi:unrelated":          Entry{ID: "3", Name: "unrelated", Provider: "pi"},
	}

	t.Run("matches substring", func(t *testing.T) {
		filtered := filterByName(sessions, "deploy")
		assert.Len(t, filtered, 2)
		assert.Contains(t, filtered, "claude:deploy-prod")
		assert.Contains(t, filtered, "claude:deploy-staging")
	})

	t.Run("case insensitive", func(t *testing.T) {
		filtered := filterByName(sessions, "DEPLOY")
		assert.Len(t, filtered, 2)
	})

	t.Run("no match", func(t *testing.T) {
		filtered := filterByName(sessions, "nonexistent")
		assert.Len(t, filtered, 0)
	})

	t.Run("empty filter returns all", func(t *testing.T) {
		filtered := filterByName(sessions, "")
		assert.Len(t, filtered, 3)
	})
}

func TestFindByIDAny(t *testing.T) {
	sessions := Sessions{
		"claude:alpha": Entry{ID: "uuid-1", Name: "alpha", Provider: "claude"},
		"pi:beta":      Entry{ID: "uuid-2", Name: "beta", Provider: "pi"},
	}

	t.Run("finds across providers", func(t *testing.T) {
		found := FindByIDAny(sessions, "uuid-2")
		require.NotNil(t, found)
		assert.Equal(t, "beta", found.Name)
		assert.Equal(t, "pi", found.Provider)
	})

	t.Run("not found", func(t *testing.T) {
		found := FindByIDAny(sessions, "nope")
		assert.Nil(t, found)
	})

	t.Run("defaults provider for legacy entries", func(t *testing.T) {
		sessions := Sessions{
			"legacy": Entry{ID: "uuid-legacy", Name: "legacy"},
		}
		found := FindByIDAny(sessions, "uuid-legacy")
		require.NotNil(t, found)
		assert.Equal(t, defaultProviderFallback, found.Provider)
	})
}

func TestMetaSerialization(t *testing.T) {
	tmpDir := t.TempDir()
	origFn := filePathFn
	filePathFn = func() string { return tmpDir + "/sessions.json" }
	t.Cleanup(func() { filePathFn = origFn })

	sessions := Sessions{
		"claude:proj": Entry{
			ID:       "id-1",
			Name:     "proj",
			Provider: "claude",
			Path:     "/tmp",
			Meta: map[string]string{
				"env":  "prod",
				"team": "infra",
			},
		},
	}

	require.NoError(t, Save(filePathFn(), sessions))

	loaded, err := Load(filePathFn())
	require.NoError(t, err)

	entry := loaded["claude:proj"]
	assert.Equal(t, "prod", entry.Meta["env"])
	assert.Equal(t, "infra", entry.Meta["team"])
}

func TestParseMeta(t *testing.T) {
	t.Run("set flags", func(t *testing.T) {
		meta, err := parseMeta([]string{"key=value", "env=prod"}, "")
		require.NoError(t, err)
		assert.Equal(t, "value", meta["key"])
		assert.Equal(t, "prod", meta["env"])
	})

	t.Run("meta json", func(t *testing.T) {
		meta, err := parseMeta(nil, `{"key":"value","env":"prod"}`)
		require.NoError(t, err)
		assert.Equal(t, "value", meta["key"])
		assert.Equal(t, "prod", meta["env"])
	})

	t.Run("set overrides meta json", func(t *testing.T) {
		meta, err := parseMeta([]string{"env=staging"}, `{"env":"prod","team":"infra"}`)
		require.NoError(t, err)
		assert.Equal(t, "staging", meta["env"])
		assert.Equal(t, "infra", meta["team"])
	})

	t.Run("invalid set format", func(t *testing.T) {
		_, err := parseMeta([]string{"noequals"}, "")
		assert.Error(t, err)
	})

	t.Run("invalid meta json", func(t *testing.T) {
		_, err := parseMeta(nil, "not json")
		assert.Error(t, err)
	})

	t.Run("empty returns nil", func(t *testing.T) {
		meta, err := parseMeta(nil, "")
		require.NoError(t, err)
		assert.Nil(t, meta)
	})
}
