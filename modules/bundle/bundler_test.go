package bundle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBundleFile(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "empty file",
			content:  "",
			expected: nil,
		},
		{
			name:     "simple packages",
			content:  "pkg1\npkg2\npkg3",
			expected: []string{"pkg1", "pkg2", "pkg3"},
		},
		{
			name:     "with comments",
			content:  "# comment\npkg1\n# another comment\npkg2",
			expected: []string{"pkg1", "pkg2"},
		},
		{
			name:     "with blank lines",
			content:  "pkg1\n\n\npkg2\n   \npkg3",
			expected: []string{"pkg1", "pkg2", "pkg3"},
		},
		{
			name:     "with whitespace",
			content:  "  pkg1  \n\tpkg2\t\n pkg3 ",
			expected: []string{"pkg1", "pkg2", "pkg3"},
		},
		{
			name:     "mixed",
			content:  "# Header\npkg1\n\n# Section\npkg2\n   \n# End",
			expected: []string{"pkg1", "pkg2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := filepath.Join(t.TempDir(), "Testfile")
			err := os.WriteFile(tmpFile, []byte(tt.content), 0644)
			require.NoError(t, err)

			packages, err := ParseBundleFile(tmpFile)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, packages)
		})
	}
}

func TestLoadState(t *testing.T) {
	t.Run("file not found returns nil", func(t *testing.T) {
		packages, err := LoadState("/nonexistent/path")
		require.NoError(t, err)
		assert.Nil(t, packages)
	})

	t.Run("reads state file", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "state")
		err := os.WriteFile(tmpFile, []byte("pkg1\npkg2\npkg3\n"), 0644)
		require.NoError(t, err)

		packages, err := LoadState(tmpFile)
		require.NoError(t, err)
		assert.Equal(t, []string{"pkg1", "pkg2", "pkg3"}, packages)
	})
}

func TestSaveState(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "state")
	packages := []string{"pkg1", "pkg2", "pkg3"}

	err := SaveState(tmpFile, packages)
	require.NoError(t, err)

	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, "pkg1\npkg2\npkg3\n", string(content))
}

func TestDiffPackages(t *testing.T) {
	tests := []struct {
		name            string
		desired         []string
		installed       []string
		expectInstall   []string
		expectRemove    []string
	}{
		{
			name:          "all new",
			desired:       []string{"a", "b", "c"},
			installed:     nil,
			expectInstall: []string{"a", "b", "c"},
			expectRemove:  nil,
		},
		{
			name:          "all removed",
			desired:       nil,
			installed:     []string{"a", "b", "c"},
			expectInstall: nil,
			expectRemove:  []string{"a", "b", "c"},
		},
		{
			name:          "no changes",
			desired:       []string{"a", "b"},
			installed:     []string{"a", "b"},
			expectInstall: nil,
			expectRemove:  nil,
		},
		{
			name:          "mixed changes",
			desired:       []string{"a", "b", "c"},
			installed:     []string{"a", "d", "e"},
			expectInstall: []string{"b", "c"},
			expectRemove:  []string{"d", "e"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toInstall, toRemove := DiffPackages(tt.desired, tt.installed)
			assert.Equal(t, tt.expectInstall, toInstall)
			assert.Equal(t, tt.expectRemove, toRemove)
		})
	}
}

func TestNormalizePackage(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"@scope/package", "@scope/package"},
		{"@org/pkg", "@org/pkg"},
		{"user/repo", "github:user/repo"},
		{"user/repo@v1.0", "github:user/repo#v1.0"},
		{"simple-pkg", "simple-pkg"},
		{"lodash", "lodash"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizePackage(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
