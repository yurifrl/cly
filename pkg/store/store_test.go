package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore(t *testing.T) {
	// Create temp directory for test database
	tmpDir, err := os.MkdirTemp("", "store-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	store, err := New(dbPath)
	require.NoError(t, err)
	defer store.Close()

	t.Run("List empty namespace returns empty slice", func(t *testing.T) {
		keys, err := store.List("empty")
		require.NoError(t, err)
		assert.Empty(t, keys)
	})

	t.Run("Add and List", func(t *testing.T) {
		err := store.Add("go", "github.com/foo/bar")
		require.NoError(t, err)

		err = store.Add("go", "github.com/baz/qux")
		require.NoError(t, err)

		keys, err := store.List("go")
		require.NoError(t, err)
		assert.Len(t, keys, 2)
		assert.Contains(t, keys, "github.com/foo/bar")
		assert.Contains(t, keys, "github.com/baz/qux")
	})

	t.Run("Add is idempotent", func(t *testing.T) {
		err := store.Add("js", "lodash")
		require.NoError(t, err)

		err = store.Add("js", "lodash")
		require.NoError(t, err)

		keys, err := store.List("js")
		require.NoError(t, err)
		assert.Len(t, keys, 1)
	})

	t.Run("Remove", func(t *testing.T) {
		err := store.Add("python", "ruff")
		require.NoError(t, err)

		err = store.Add("python", "black")
		require.NoError(t, err)

		err = store.Remove("python", "ruff")
		require.NoError(t, err)

		keys, err := store.List("python")
		require.NoError(t, err)
		assert.Len(t, keys, 1)
		assert.Contains(t, keys, "black")
	})

	t.Run("Remove is idempotent", func(t *testing.T) {
		err := store.Remove("python", "nonexistent")
		require.NoError(t, err)
	})

	t.Run("Namespaces are isolated", func(t *testing.T) {
		err := store.Add("ns1", "key")
		require.NoError(t, err)

		keys, err := store.List("ns2")
		require.NoError(t, err)
		assert.Empty(t, keys)

		keys, err = store.List("ns1")
		require.NoError(t, err)
		assert.Len(t, keys, 1)
	})
}

func TestNew_CreatesDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "store-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	nestedPath := filepath.Join(tmpDir, "nested", "dir", "test.db")
	store, err := New(nestedPath)
	require.NoError(t, err)
	defer store.Close()

	// Verify directory was created
	_, err = os.Stat(filepath.Dir(nestedPath))
	assert.NoError(t, err)
}
