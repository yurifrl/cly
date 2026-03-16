package helpy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverDocs(t *testing.T) {
	// Create temp dir with nested structure
	tmpDir := t.TempDir()

	// Root level docs
	os.WriteFile(filepath.Join(tmpDir, "readme.md"), []byte("# Readme"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "guide.md"), []byte("# Guide"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "not-a-doc.txt"), []byte("text"), 0644)

	// Nested dir
	subDir := filepath.Join(tmpDir, "pi", "extensions")
	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "oh-pi.md"), []byte("# Oh Pi"), 0644)
	os.WriteFile(filepath.Join(subDir, "pi-teams.md"), []byte("# Teams"), 0644)

	docs, err := discoverDocs(tmpDir)
	require.NoError(t, err)
	assert.Len(t, docs, 4)

	// Check names are relative without .md
	names := make(map[string]bool)
	for _, d := range docs {
		names[d.name] = true
	}
	assert.True(t, names["readme"])
	assert.True(t, names["guide"])
	assert.True(t, names[filepath.Join("pi", "extensions", "oh-pi")])
	assert.True(t, names[filepath.Join("pi", "extensions", "pi-teams")])
}

func TestDiscoverDocsEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := discoverDocs(tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no .md files")
}

func TestDiscoverDocsNonExistent(t *testing.T) {
	_, err := discoverDocs("/nonexistent/path/that/doesnt/exist")
	assert.Error(t, err)
}

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		s     string
		query string
		want  bool
	}{
		{"oh-pi", "oh", true},
		{"oh-pi", "opi", true},
		{"pi-teams", "ptm", true},
		{"pi/extensions/oh-pi", "ext", true},
		{"pi-guardrails", "xyz", false},
		{"pi-notify", "pn", true},
		{"pi-notify", "notify", true},
		{"anything", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.query, func(t *testing.T) {
			got := fuzzyMatch(tt.s, tt.query)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPickerFilterDocs(t *testing.T) {
	docs := []docEntry{
		{name: "oh-pi", path: "/a/oh-pi.md"},
		{name: "pi-teams", path: "/a/pi-teams.md"},
		{name: "pi-notify", path: "/a/pi-notify.md"},
		{name: "visual-explainer", path: "/a/visual-explainer.md"},
	}

	m := newPickerModel(docs)

	// No filter = all docs
	m.filterDocs()
	assert.Len(t, m.filtered, 4)

	// Filter "team"
	m.filterInput.SetValue("team")
	m.filterDocs()
	assert.Len(t, m.filtered, 1)
	assert.Equal(t, "pi-teams", m.filtered[0].name)

	// Filter "pi" matches multiple
	m.filterInput.SetValue("pi")
	m.filterDocs()
	assert.True(t, len(m.filtered) >= 3) // oh-pi, pi-teams, pi-notify

	// Filter no match
	m.filterInput.SetValue("zzzzz")
	m.filterDocs()
	assert.Len(t, m.filtered, 0)

	// Verify selectedIdx clamped
	m.selectedIdx = 5
	m.filterInput.SetValue("vis")
	m.filterDocs()
	assert.Equal(t, 0, m.selectedIdx)
}

func TestNewPickerModel(t *testing.T) {
	docs := []docEntry{
		{name: "test", path: "/test.md"},
	}
	m := newPickerModel(docs)
	assert.Len(t, m.docs, 1)
	assert.Len(t, m.filtered, 1)
	assert.Equal(t, 0, m.selectedIdx)
	assert.False(t, m.viewing)
}
