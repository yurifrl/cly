package claudesession

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
		MakeKey("/tmp/project", "abc123"): Entry{
			ID:          "abc123",
			Name:        "my-session",
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
		MakeKey("/tmp", "id1"): Entry{ID: "id1", Name: "s1", Path: "/tmp"},
	}

	err := Save(path, sessions)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Dir(path))
	assert.NoError(t, err)
}

func TestFindByName(t *testing.T) {
	sessions := Sessions{
		MakeKey("/a", "1"): Entry{ID: "1", Name: "alpha", Path: "/a"},
		MakeKey("/b", "2"): Entry{ID: "2", Name: "beta", Path: "/b"},
	}

	found := FindByName(sessions, "alpha")
	require.NotNil(t, found)
	assert.Equal(t, "1", found.ID)

	notFound := FindByName(sessions, "gamma")
	assert.Nil(t, notFound)
}

func TestRemove(t *testing.T) {
	sessions := Sessions{
		MakeKey("/a", "1"): Entry{ID: "1", Name: "alpha", Path: "/a"},
		MakeKey("/b", "2"): Entry{ID: "2", Name: "beta", Path: "/b"},
	}

	result := Remove(sessions, "alpha")
	assert.Len(t, result, 1)
	assert.Nil(t, FindByName(result, "alpha"))
	assert.NotNil(t, FindByName(result, "beta"))
}

func TestRemoveNonExistent(t *testing.T) {
	sessions := Sessions{
		MakeKey("/a", "1"): Entry{ID: "1", Name: "alpha", Path: "/a"},
	}

	result := Remove(sessions, "nope")
	assert.Len(t, result, 1)
}

func TestMakeKey(t *testing.T) {
	key := MakeKey("/some/path", "myid")
	assert.Equal(t, "/some/path:myid", key)
}

func TestFindByNameAndPath(t *testing.T) {
	sessions := Sessions{
		MakeKey("/a", "1"): Entry{ID: "1", Name: "alpha", Path: "/a"},
		MakeKey("/b", "2"): Entry{ID: "2", Name: "beta", Path: "/b"},
		MakeKey("/a", "3"): Entry{ID: "3", Name: "beta", Path: "/a"},
	}

	found := FindByNameAndPath(sessions, "alpha", "/a")
	require.NotNil(t, found)
	assert.Equal(t, "1", found.ID)
	assert.Equal(t, "/a", found.Path)

	foundDifferentPath := FindByNameAndPath(sessions, "beta", "/a")
	require.NotNil(t, foundDifferentPath)
	assert.Equal(t, "3", foundDifferentPath.ID)
	assert.Equal(t, "/a", foundDifferentPath.Path)

	notFound := FindByNameAndPath(sessions, "alpha", "/b")
	assert.Nil(t, notFound)

	notFoundName := FindByNameAndPath(sessions, "gamma", "/a")
	assert.Nil(t, notFoundName)
}
