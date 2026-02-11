package claudetasks

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")

	store := Store{
		"my-task": TaskList{Name: "my-task"},
	}

	err := Save(path, store)
	require.NoError(t, err)

	loaded, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, store, loaded)
}

func TestLoadNonExistent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nope.json")

	loaded, err := Load(path)
	require.NoError(t, err)
	assert.Empty(t, loaded)
}

func TestSaveCreatesDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "dir", "tasks.json")

	err := Save(path, Store{"x": TaskList{Name: "x"}})
	require.NoError(t, err)

	loaded, err := Load(path)
	require.NoError(t, err)
	assert.Len(t, loaded, 1)
}
