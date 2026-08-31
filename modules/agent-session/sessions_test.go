package agentsession

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "sessions.json")

	sessions := Sessions{
		"claude:my-session": Entry{
			ID:          "abc123",
			Name:        "my-session",
			Provider:    "claude",
			Path:        "/tmp/project",
			Description: "test desc",
		},
	}

	err := Save(path, sessions)
	require.NoError(t, err)

	loaded, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, sessions, loaded)
}

func TestLoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nonexistent.json")

	loaded, err := Load(path)
	require.NoError(t, err)
	assert.Empty(t, loaded)
}

func TestLoadCreatesDir(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "sub", "dir", "sessions.json")

	sessions := Sessions{
		"claude:s1": Entry{ID: "id1", Name: "s1", Provider: "claude", Path: "/tmp"},
	}

	err := Save(path, sessions)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Dir(path))
	assert.NoError(t, err)
}

func TestFindByName_DefaultProvider(t *testing.T) {
	sessions := Sessions{
		"alpha":       Entry{ID: "1", Name: "alpha", Path: "/a"}, // legacy entry, implicit provider
		"pi:alpha":    Entry{ID: "2", Name: "alpha", Provider: "pi", Path: "/b"},
		"claude:beta": Entry{ID: "3", Name: "beta", Provider: "claude", Path: "/c"},
	}

	found := FindByName(sessions, "alpha")
	require.NotNil(t, found)
	assert.Equal(t, "1", found.ID)
	assert.Equal(t, defaultProviderFallback, found.Provider)

	notFound := FindByName(sessions, "gamma")
	assert.Nil(t, notFound)
}

func TestFindByNameForProvider(t *testing.T) {
	sessions := Sessions{
		"claude:alpha": Entry{ID: "1", Name: "alpha", Provider: "claude", Path: "/a"},
		"pi:alpha":     Entry{ID: "2", Name: "alpha", Provider: "pi", Path: "/b"},
	}

	found := FindByNameForProvider(sessions, "pi", "alpha")
	require.NotNil(t, found)
	assert.Equal(t, "2", found.ID)
}

func TestRemoveForProvider(t *testing.T) {
	sessions := Sessions{
		"claude:alpha": Entry{ID: "1", Name: "alpha", Provider: "claude", Path: "/a"},
		"pi:alpha":     Entry{ID: "2", Name: "alpha", Provider: "pi", Path: "/b"},
	}

	result := RemoveForProvider(sessions, "claude", "alpha")
	assert.Len(t, result, 1)
	assert.Nil(t, FindByNameForProvider(result, "claude", "alpha"))
	assert.NotNil(t, FindByNameForProvider(result, "pi", "alpha"))
}

func TestFindByIDForProvider(t *testing.T) {
	sessions := Sessions{
		"claude:alpha": Entry{ID: "uuid-1", Name: "alpha", Provider: "claude", Path: "/a"},
		"pi:beta":      Entry{ID: "uuid-2", Name: "beta", Provider: "pi", Path: "/b"},
	}

	found := FindByIDForProvider(sessions, "pi", "uuid-2")
	require.NotNil(t, found)
	assert.Equal(t, "beta", found.Name)

	notFound := FindByIDForProvider(sessions, "claude", "uuid-2")
	assert.Nil(t, notFound)
}

func TestUpsertEntryMigratesLegacyKey(t *testing.T) {
	sessions := Sessions{
		"alpha": Entry{ID: "old", Name: "alpha", Path: "/a"},
	}

	upsertEntry(sessions, Entry{ID: "new", Name: "alpha", Provider: defaultProviderFallback, Path: "/a"})
	assert.NotContains(t, sessions, "alpha")
	assert.Contains(t, sessions, defaultProviderFallback+":alpha")
	assert.Equal(t, "new", sessions[defaultProviderFallback+":alpha"].ID)
}
