package dotfiles

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateSymlink(t *testing.T) {
	t.Run("creates symlink for file", func(t *testing.T) {
		tmpDir := t.TempDir()
		source := filepath.Join(tmpDir, "source.txt")
		dest := filepath.Join(tmpDir, "dest.txt")
		require.NoError(t, os.WriteFile(source, []byte("content"), 0644))

		mapping := Mapping{Source: source, Destination: dest, IsDir: false}
		result := CreateSymlink(mapping)

		assert.Equal(t, StateLinked, result.State)
		assert.Empty(t, result.Error)

		link, err := os.Readlink(dest)
		require.NoError(t, err)
		assert.Equal(t, source, link)
	})

	t.Run("creates symlink for directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		source := filepath.Join(tmpDir, "sourcedir")
		dest := filepath.Join(tmpDir, "destdir")
		require.NoError(t, os.Mkdir(source, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(source, "file.txt"), []byte("content"), 0644))

		mapping := Mapping{Source: source, Destination: dest, IsDir: true}
		result := CreateSymlink(mapping)

		assert.Equal(t, StateLinked, result.State)
		link, err := os.Readlink(dest)
		require.NoError(t, err)
		assert.Equal(t, source, link)
	})

	t.Run("creates parent directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		source := filepath.Join(tmpDir, "source.txt")
		dest := filepath.Join(tmpDir, "nested", "deep", "dest.txt")
		require.NoError(t, os.WriteFile(source, []byte("content"), 0644))

		mapping := Mapping{Source: source, Destination: dest, IsDir: false}
		result := CreateSymlink(mapping)

		assert.Equal(t, StateLinked, result.State)
		_, err := os.Stat(filepath.Dir(dest))
		assert.NoError(t, err)
	})

	t.Run("replaces existing symlink", func(t *testing.T) {
		tmpDir := t.TempDir()
		source1 := filepath.Join(tmpDir, "source1.txt")
		source2 := filepath.Join(tmpDir, "source2.txt")
		dest := filepath.Join(tmpDir, "dest.txt")
		require.NoError(t, os.WriteFile(source1, []byte("content1"), 0644))
		require.NoError(t, os.WriteFile(source2, []byte("content2"), 0644))
		require.NoError(t, os.Symlink(source1, dest))

		mapping := Mapping{Source: source2, Destination: dest, IsDir: false}
		result := CreateSymlink(mapping)

		assert.Equal(t, StateLinked, result.State)
		link, err := os.Readlink(dest)
		require.NoError(t, err)
		assert.Equal(t, source2, link)
	})

	t.Run("links to real file when source is a symlink", func(t *testing.T) {
		tmpDir := t.TempDir()
		realSource := filepath.Join(tmpDir, "real.txt")
		source := filepath.Join(tmpDir, "source.txt")
		dest := filepath.Join(tmpDir, "dest.txt")
		require.NoError(t, os.WriteFile(realSource, []byte("real"), 0644))
		require.NoError(t, os.Symlink(realSource, source))

		mapping := Mapping{Source: source, Destination: dest, IsDir: false}
		result := CreateSymlink(mapping)

		assert.Equal(t, StateLinked, result.State)
		link, err := os.Readlink(dest)
		require.NoError(t, err)
		assert.Equal(t, realSource, link)
	})

	t.Run("resolves relative and chained source symlinks", func(t *testing.T) {
		tmpDir := t.TempDir()
		realSource := filepath.Join(tmpDir, "a.txt")
		mid := filepath.Join(tmpDir, "b.txt")
		source := filepath.Join(tmpDir, "c.txt")
		dest := filepath.Join(tmpDir, "dest.txt")
		require.NoError(t, os.WriteFile(realSource, []byte("a"), 0644))
		require.NoError(t, os.Symlink("a.txt", mid))
		require.NoError(t, os.Symlink("b.txt", source))

		mapping := Mapping{Source: source, Destination: dest, IsDir: false}
		result := CreateSymlink(mapping)

		assert.Equal(t, StateLinked, result.State)
		link, err := os.Readlink(dest)
		require.NoError(t, err)
		assert.Equal(t, realSource, link)
	})

	t.Run("replaces dest symlink pointing at the link name", func(t *testing.T) {
		tmpDir := t.TempDir()
		realSource := filepath.Join(tmpDir, "real.txt")
		source := filepath.Join(tmpDir, "source.txt")
		dest := filepath.Join(tmpDir, "dest.txt")
		require.NoError(t, os.WriteFile(realSource, []byte("real"), 0644))
		require.NoError(t, os.Symlink(realSource, source))
		require.NoError(t, os.Symlink(source, dest)) // old behavior target

		mapping := Mapping{Source: source, Destination: dest, IsDir: false}
		result := CreateSymlink(mapping)

		assert.Equal(t, StateLinked, result.State)
		assert.True(t, result.RemovedExisting)
		link, err := os.Readlink(dest)
		require.NoError(t, err)
		assert.Equal(t, realSource, link)
	})

	t.Run("force overrides existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		source := filepath.Join(tmpDir, "source.txt")
		dest := filepath.Join(tmpDir, "dest.txt")
		require.NoError(t, os.WriteFile(source, []byte("source"), 0644))
		require.NoError(t, os.WriteFile(dest, []byte("existing"), 0644))

		mapping := Mapping{Source: source, Destination: dest, IsDir: false}
		result := CreateSymlink(mapping)

		assert.Equal(t, StateLinked, result.State)
		link, err := os.Readlink(dest)
		require.NoError(t, err)
		assert.Equal(t, source, link)
	})

	t.Run("force overrides existing directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		source := filepath.Join(tmpDir, "sourcedir")
		dest := filepath.Join(tmpDir, "destdir")
		require.NoError(t, os.Mkdir(source, 0755))
		require.NoError(t, os.Mkdir(dest, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dest, "file.txt"), []byte("content"), 0644))

		mapping := Mapping{Source: source, Destination: dest, IsDir: true}
		result := CreateSymlink(mapping)

		assert.Equal(t, StateLinked, result.State)
		link, err := os.Readlink(dest)
		require.NoError(t, err)
		assert.Equal(t, source, link)
	})

	t.Run("warns on missing source", func(t *testing.T) {
		tmpDir := t.TempDir()
		source := filepath.Join(tmpDir, "nonexistent.txt")
		dest := filepath.Join(tmpDir, "dest.txt")

		mapping := Mapping{Source: source, Destination: dest, IsDir: false}
		result := CreateSymlink(mapping)

		assert.Equal(t, StateMissing, result.State)
	})

	t.Run("errors when source is directory but IsDir is false", func(t *testing.T) {
		tmpDir := t.TempDir()
		source := filepath.Join(tmpDir, "sourcedir")
		dest := filepath.Join(tmpDir, "dest")
		require.NoError(t, os.Mkdir(source, 0755))

		mapping := Mapping{Source: source, Destination: dest, IsDir: false}
		result := CreateSymlink(mapping)

		assert.Equal(t, StateError, result.State)
		assert.Contains(t, result.Error, "trailing slash")
	})
}

func TestCheckStatus(t *testing.T) {
	t.Run("linked when symlink points to source", func(t *testing.T) {
		tmpDir := t.TempDir()
		source := filepath.Join(tmpDir, "source.txt")
		dest := filepath.Join(tmpDir, "dest.txt")
		require.NoError(t, os.WriteFile(source, []byte("content"), 0644))
		require.NoError(t, os.Symlink(source, dest))

		mapping := Mapping{Source: source, Destination: dest}
		result := CheckStatus(mapping)

		assert.Equal(t, StateLinked, result.State)
	})

	t.Run("linked when dest symlink points at resolved source target", func(t *testing.T) {
		tmpDir := t.TempDir()
		realSource := filepath.Join(tmpDir, "real.txt")
		source := filepath.Join(tmpDir, "source.txt")
		dest := filepath.Join(tmpDir, "dest.txt")
		require.NoError(t, os.WriteFile(realSource, []byte("real"), 0644))
		require.NoError(t, os.Symlink(realSource, source))
		require.NoError(t, os.Symlink(realSource, dest))

		mapping := Mapping{Source: source, Destination: dest}
		result := CheckStatus(mapping)

		assert.Equal(t, StateLinked, result.State)
	})

	t.Run("conflict when dest symlink points at the source link name", func(t *testing.T) {
		tmpDir := t.TempDir()
		realSource := filepath.Join(tmpDir, "real.txt")
		source := filepath.Join(tmpDir, "source.txt")
		dest := filepath.Join(tmpDir, "dest.txt")
		require.NoError(t, os.WriteFile(realSource, []byte("real"), 0644))
		require.NoError(t, os.Symlink(realSource, source))
		require.NoError(t, os.Symlink(source, dest))

		mapping := Mapping{Source: source, Destination: dest}
		result := CheckStatus(mapping)

		assert.Equal(t, StateConflict, result.State)
		assert.Contains(t, result.Error, "instead of")
	})

	t.Run("missing when source does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		source := filepath.Join(tmpDir, "nonexistent.txt")
		dest := filepath.Join(tmpDir, "dest.txt")

		mapping := Mapping{Source: source, Destination: dest}
		result := CheckStatus(mapping)

		assert.Equal(t, StateMissing, result.State)
	})

	t.Run("conflict when destination is regular file", func(t *testing.T) {
		tmpDir := t.TempDir()
		source := filepath.Join(tmpDir, "source.txt")
		dest := filepath.Join(tmpDir, "dest.txt")
		require.NoError(t, os.WriteFile(source, []byte("source"), 0644))
		require.NoError(t, os.WriteFile(dest, []byte("existing"), 0644))

		mapping := Mapping{Source: source, Destination: dest}
		result := CheckStatus(mapping)

		assert.Equal(t, StateConflict, result.State)
	})

	t.Run("broken when symlink target missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		source := filepath.Join(tmpDir, "nonexistent.txt")
		dest := filepath.Join(tmpDir, "dest.txt")
		require.NoError(t, os.Symlink(source, dest))

		mapping := Mapping{Source: source, Destination: dest}
		result := CheckStatus(mapping)

		assert.Equal(t, StateBroken, result.State)
	})

	t.Run("unlinked when destination does not exist but source does", func(t *testing.T) {
		tmpDir := t.TempDir()
		source := filepath.Join(tmpDir, "source.txt")
		dest := filepath.Join(tmpDir, "dest.txt")
		require.NoError(t, os.WriteFile(source, []byte("content"), 0644))

		mapping := Mapping{Source: source, Destination: dest}
		result := CheckStatus(mapping)

		assert.Equal(t, StateUnlinked, result.State)
	})
}

func TestRemoveSymlink(t *testing.T) {
	t.Run("removes existing symlink", func(t *testing.T) {
		tmpDir := t.TempDir()
		source := filepath.Join(tmpDir, "source.txt")
		dest := filepath.Join(tmpDir, "dest.txt")
		require.NoError(t, os.WriteFile(source, []byte("content"), 0644))
		require.NoError(t, os.Symlink(source, dest))

		mapping := Mapping{Source: source, Destination: dest}
		removed := RemoveSymlink(mapping)

		assert.True(t, removed)
		_, err := os.Lstat(dest)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("skips non-symlink files", func(t *testing.T) {
		tmpDir := t.TempDir()
		source := filepath.Join(tmpDir, "source.txt")
		dest := filepath.Join(tmpDir, "dest.txt")
		require.NoError(t, os.WriteFile(source, []byte("source"), 0644))
		require.NoError(t, os.WriteFile(dest, []byte("existing"), 0644))

		mapping := Mapping{Source: source, Destination: dest}
		removed := RemoveSymlink(mapping)

		assert.False(t, removed)
		_, err := os.Stat(dest)
		assert.NoError(t, err)
	})

	t.Run("skips non-existent destination", func(t *testing.T) {
		tmpDir := t.TempDir()
		source := filepath.Join(tmpDir, "source.txt")
		dest := filepath.Join(tmpDir, "dest.txt")

		mapping := Mapping{Source: source, Destination: dest}
		removed := RemoveSymlink(mapping)

		assert.False(t, removed)
	})
}
