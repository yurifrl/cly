package zl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadZlConfig(t *testing.T) {
	t.Run("returns defaults when no config", func(t *testing.T) {
		oldGet := getConfigFunc
		getConfigFunc = func() map[string]interface{} {
			return nil
		}
		defer func() { getConfigFunc = oldGet }()

		cfg := LoadZlConfig()
		assert.True(t, cfg.AutoZoxide)
		assert.True(t, cfg.UpdateZoxide)
		assert.Empty(t, cfg.SessionDirs)
	})

	t.Run("loads custom values from config", func(t *testing.T) {
		oldGet := getConfigFunc
		getConfigFunc = func() map[string]interface{} {
			return map[string]interface{}{
				"auto_zoxide":   false,
				"update_zoxide": false,
				"session_dirs": map[string]interface{}{
					"work": "/home/user/work",
					"cly":  "/home/user/cly",
				},
			}
		}
		defer func() { getConfigFunc = oldGet }()

		cfg := LoadZlConfig()
		assert.False(t, cfg.AutoZoxide)
		assert.False(t, cfg.UpdateZoxide)
		assert.Equal(t, "/home/user/work", cfg.SessionDirs["work"])
		assert.Equal(t, "/home/user/cly", cfg.SessionDirs["cly"])
	})

	t.Run("handles partial config with defaults", func(t *testing.T) {
		oldGet := getConfigFunc
		getConfigFunc = func() map[string]interface{} {
			return map[string]interface{}{
				"session_dirs": map[string]interface{}{
					"work": "/home/user/work",
				},
			}
		}
		defer func() { getConfigFunc = oldGet }()

		cfg := LoadZlConfig()
		assert.True(t, cfg.AutoZoxide) // default
		assert.True(t, cfg.UpdateZoxide) // default
		assert.Equal(t, "/home/user/work", cfg.SessionDirs["work"])
	})
}

func TestSaveSessionMapping(t *testing.T) {
	t.Run("saves mapping to config", func(t *testing.T) {
		oldSet := setConfigFunc
		var savedKey, savedValue string
		setConfigFunc = func(key string, value interface{}) error {
			savedKey = key
			if m, ok := value.(map[string]string); ok {
				savedValue = m["test"]
			}
			return nil
		}
		defer func() { setConfigFunc = oldSet }()

		oldGet := getConfigFunc
		getConfigFunc = func() map[string]interface{} {
			return map[string]interface{}{
				"session_dirs": map[string]interface{}{},
			}
		}
		defer func() { getConfigFunc = oldGet }()

		err := SaveSessionMapping("test", "/home/user/test")
		require.NoError(t, err)
		assert.Equal(t, "modules.zl.session_dirs", savedKey)
		assert.Equal(t, "/home/user/test", savedValue)
	})

	t.Run("merges with existing mappings", func(t *testing.T) {
		oldSet := setConfigFunc
		var savedMappings map[string]string
		setConfigFunc = func(key string, value interface{}) error {
			if m, ok := value.(map[string]string); ok {
				savedMappings = m
			}
			return nil
		}
		defer func() { setConfigFunc = oldSet }()

		oldGet := getConfigFunc
		getConfigFunc = func() map[string]interface{} {
			return map[string]interface{}{
				"session_dirs": map[string]interface{}{
					"work": "/home/user/work",
				},
			}
		}
		defer func() { getConfigFunc = oldGet }()

		err := SaveSessionMapping("cly", "/home/user/cly")
		require.NoError(t, err)
		assert.Equal(t, "/home/user/work", savedMappings["work"])
		assert.Equal(t, "/home/user/cly", savedMappings["cly"])
	})
}
