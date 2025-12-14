package bundle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yurifrl/cly/pkg/store"
)

func TestParseFile(t *testing.T) {
	// Create temp file
	tmpDir, err := os.MkdirTemp("", "bundle-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	bundleFile := filepath.Join(tmpDir, "testfile")

	t.Run("parses packages ignoring comments and blanks", func(t *testing.T) {
		content := `# This is a comment
package1
package2

# Another comment
package3
   # indented comment

package4`
		err := os.WriteFile(bundleFile, []byte(content), 0644)
		require.NoError(t, err)

		packages, err := parseFile(bundleFile)
		require.NoError(t, err)
		assert.Equal(t, []string{"package1", "package2", "package3", "package4"}, packages)
	})

	t.Run("returns empty slice for empty file", func(t *testing.T) {
		err := os.WriteFile(bundleFile, []byte(""), 0644)
		require.NoError(t, err)

		packages, err := parseFile(bundleFile)
		require.NoError(t, err)
		assert.Empty(t, packages)
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		_, err := parseFile("/nonexistent/file")
		assert.Error(t, err)
	})
}

func TestDiff(t *testing.T) {
	t.Run("finds items in a not in b", func(t *testing.T) {
		a := []string{"a", "b", "c"}
		b := []string{"b", "d"}
		result := diff(a, b)
		assert.ElementsMatch(t, []string{"a", "c"}, result)
	})

	t.Run("returns empty for identical slices", func(t *testing.T) {
		a := []string{"a", "b"}
		b := []string{"a", "b"}
		result := diff(a, b)
		assert.Empty(t, result)
	})
}

func TestGetBinaryName(t *testing.T) {
	tests := []struct {
		pkg      string
		expected string
	}{
		{"github.com/foo/bar", "bar"},
		{"github.com/foo/bar/cmd/baz", "baz"},
		{"github.com/foo/bar@v1.0.0", "bar"},
		{"github.com/foo/bar/cmd/baz@latest", "baz"},
		{"golang.org/x/tools/gopls", "gopls"},
	}

	for _, tt := range tests {
		t.Run(tt.pkg, func(t *testing.T) {
			result := getBinaryName(tt.pkg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizePackage(t *testing.T) {
	tests := []struct {
		pkg      string
		expected string
	}{
		{"lodash", "lodash"},
		{"@scope/package", "@scope/package"},
		{"user/repo", "github:user/repo"},
		{"user/repo@v1.0.0", "github:user/repo#v1.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.pkg, func(t *testing.T) {
			result := normalizePackage(tt.pkg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractBasePkg(t *testing.T) {
	tests := []struct {
		pkg      string
		expected string
	}{
		{"ruff", "ruff"},
		{"vectorcode[lsp,mcp]", "vectorcode"},
		{"vectorcode[lsp]<1.0.0", "vectorcode"},
		{"package>=1.0.0", "package"},
		{"package@1.0.0", "package"},
	}

	for _, tt := range tests {
		t.Run(tt.pkg, func(t *testing.T) {
			result := extractBasePkg(tt.pkg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOpenEditor(t *testing.T) {
	t.Run("uses EDITOR env var when set", func(t *testing.T) {
		editor := getEditor()
		// Should return something (either EDITOR env or fallback)
		assert.NotEmpty(t, editor)
	})

	t.Run("falls back to vim when EDITOR not set", func(t *testing.T) {
		orig := os.Getenv("EDITOR")
		os.Unsetenv("EDITOR")
		defer func() {
			if orig != "" {
				os.Setenv("EDITOR", orig)
			}
		}()

		editor := getEditor()
		assert.Equal(t, "vim", editor)
	})
}

func TestBaseBundlerCheck(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bundle-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create store
	dbPath := filepath.Join(tmpDir, "test.db")
	s, err := store.New(dbPath)
	require.NoError(t, err)
	defer s.Close()

	// Create bundle file
	bundleFile := filepath.Join(tmpDir, "testbundle")
	err = os.WriteFile(bundleFile, []byte("pkg1\npkg2\npkg3"), 0644)
	require.NoError(t, err)

	bundler := &baseBundler{
		name:        "test",
		defaultFile: bundleFile,
		store:       s,
		installFn:   func(pkg string, verbose bool) error { return nil },
		uninstallFn: func(pkg string, verbose bool) error { return nil },
	}

	t.Run("check shows changes needed when store empty", func(t *testing.T) {
		err := bundler.Check(bundleFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "changes needed")
	})

	t.Run("check shows in sync when all installed", func(t *testing.T) {
		s.Add("test", "pkg1")
		s.Add("test", "pkg2")
		s.Add("test", "pkg3")

		err := bundler.Check(bundleFile)
		assert.NoError(t, err)
	})

	t.Run("check shows changes when extra in store", func(t *testing.T) {
		s.Add("test", "pkg4")

		err := bundler.Check(bundleFile)
		assert.Error(t, err)
	})
}
