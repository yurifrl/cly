package zs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildSessionItems_MatchesReferenceShape(t *testing.T) {
	home := t.TempDir()
	oldHome := os.Getenv("HOME")
	t.Cleanup(func() {
		_ = os.Setenv("HOME", oldHome)
	})
	_ = os.Setenv("HOME", home)

	items := buildSessionItems(
		[]string{"work", "dev"},
		[]string{filepath.Join(home, "code", "cly"), filepath.Join(home, "code", "cly"), "/tmp/demo"},
	)

	if assert.Len(t, items, 5) {
		assert.Equal(t, pickerItem{Label: "work", Value: "work", Kind: selectionKindSession}, items[0])
		assert.Equal(t, pickerItem{Label: "dev", Value: "dev", Kind: selectionKindSession}, items[1])
		assert.Equal(t, pickerItem{Label: "~/code/cly", Value: filepath.Join(home, "code", "cly"), Kind: selectionKindDir}, items[2])
		assert.Equal(t, pickerItem{Label: "~/code/cly", Value: filepath.Join(home, "code", "cly"), Kind: selectionKindDir}, items[3])
		assert.Equal(t, pickerItem{Label: "/tmp/demo", Value: "/tmp/demo", Kind: selectionKindDir}, items[4])
	}
}

func TestBuildDirectoryItems_MatchesReferenceShape(t *testing.T) {
	home := t.TempDir()
	oldHome := os.Getenv("HOME")
	t.Cleanup(func() {
		_ = os.Setenv("HOME", oldHome)
	})
	_ = os.Setenv("HOME", home)

	items := buildDirectoryItems([]string{filepath.Join(home, "code", "cly"), "/tmp/demo"})
	assert.Equal(t, []pickerItem{
		{Label: "~/code/cly", Value: filepath.Join(home, "code", "cly"), Kind: selectionKindDir},
		{Label: "/tmp/demo", Value: "/tmp/demo", Kind: selectionKindDir},
	}, items)
}

func TestShortenHome(t *testing.T) {
	home := t.TempDir()
	oldHome := os.Getenv("HOME")
	t.Cleanup(func() {
		_ = os.Setenv("HOME", oldHome)
	})
	_ = os.Setenv("HOME", home)

	assert.Equal(t, "~", shortenHome(home))
	assert.Equal(t, "~/code/cly", shortenHome(filepath.Join(home, "code", "cly")))
	assert.Equal(t, "/tmp/demo", shortenHome("/tmp/demo"))
}
