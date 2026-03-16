package zs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildLayoutItems(t *testing.T) {
	items := buildLayoutItems([]string{"/tmp/default.kdl", "/tmp/dev.kdl"})

	if assert.Len(t, items, 3) {
		assert.Equal(t, "default", items[0].Display)
		assert.Equal(t, "/tmp/default.kdl", items[0].Path)
		assert.Equal(t, "dev", items[1].Display)
		assert.Equal(t, "default", items[2].Path)
	}
}

func TestFilterTabLayouts(t *testing.T) {
	dir := t.TempDir()
	plain := filepath.Join(dir, "plain.kdl")
	withTab := filepath.Join(dir, "tabbed.kdl")

	require.NoError(t, os.WriteFile(plain, []byte("pane size=1"), 0o644))
	require.NoError(t, os.WriteFile(withTab, []byte("tab name=\"main\""), 0o644))

	filtered, err := filterTabLayouts([]string{plain, withTab})
	require.NoError(t, err)
	assert.Equal(t, []string{plain}, filtered)
}

func TestDetectLayoutDirPattern(t *testing.T) {
	match := layoutDirPattern.FindStringSubmatch(`LAYOUT DIR: "/tmp/layouts"`)
	require.Len(t, match, 2)
	assert.Equal(t, "/tmp/layouts", match[1])
}
