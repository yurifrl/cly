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

func TestParsePythonSpec(t *testing.T) {
	tests := []struct {
		line          string
		expectedPkg   string
		expectedPyVer string
	}{
		{"ruff", "ruff", ""},
		{"vectorcode[lsp,mcp]", "vectorcode[lsp,mcp]", ""},
		{"vectorcode[lsp,mcp] python=3.13", "vectorcode[lsp,mcp]", "3.13"},
		{"deepeval python=3.12", "deepeval", "3.12"},
		{"package>=1.0 python=3.11", "package>=1.0", "3.11"},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			pkg, pyVer := parsePythonSpec(tt.line)
			assert.Equal(t, tt.expectedPkg, pkg)
			assert.Equal(t, tt.expectedPyVer, pyVer)
		})
	}
}

func TestExtractTaps(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bundle-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	brewfile := filepath.Join(tmpDir, "Brewfile")

	t.Run("extracts tap lines from Brewfile", func(t *testing.T) {
		content := `tap "homebrew/cask"
tap "homebrew/core"
brew "git"
cask "firefox"
tap "some/other"
brew "vim"`
		err := os.WriteFile(brewfile, []byte(content), 0644)
		require.NoError(t, err)

		taps, err := extractTaps(brewfile)
		require.NoError(t, err)
		assert.Equal(t, []string{
			`tap "homebrew/cask"`,
			`tap "homebrew/core"`,
			`tap "some/other"`,
		}, taps)
	})

	t.Run("returns empty slice when no taps", func(t *testing.T) {
		content := `brew "git"
cask "firefox"`
		err := os.WriteFile(brewfile, []byte(content), 0644)
		require.NoError(t, err)

		taps, err := extractTaps(brewfile)
		require.NoError(t, err)
		assert.Empty(t, taps)
	})

	t.Run("handles comments and empty lines", func(t *testing.T) {
		content := `# Comment
tap "homebrew/cask"

# Another comment
brew "git"
tap "some/tap"  # inline comment`
		err := os.WriteFile(brewfile, []byte(content), 0644)
		require.NoError(t, err)

		taps, err := extractTaps(brewfile)
		require.NoError(t, err)
		assert.Equal(t, []string{
			`tap "homebrew/cask"`,
			`tap "some/tap"  # inline comment`,
		}, taps)
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		_, err := extractTaps("/nonexistent/file")
		assert.Error(t, err)
	})
}

func TestParseTapName(t *testing.T) {
	cases := map[string]string{
		`tap "homebrew/cask"`:                 "homebrew/cask",
		`tap "gromgit/brewtils"  # comment`:   "gromgit/brewtils",
		`  tap "some/other"`:                   "some/other",
		`tap "user/repo", "https://x.com/r"`:  "user/repo",
		`brew "git"`:                          "",
		`# tap "commented/out"`:                "",
	}
	for line, want := range cases {
		assert.Equal(t, want, parseTapName(line), "line: %s", line)
	}
}

func TestWriteTapsToTempFile(t *testing.T) {
	t.Run("creates temp file with tap lines", func(t *testing.T) {
		taps := []string{
			`tap "homebrew/cask"`,
			`tap "homebrew/core"`,
		}

		tmpFile, err := writeTapsToTempFile(taps)
		require.NoError(t, err)
		defer os.Remove(tmpFile)

		content, err := os.ReadFile(tmpFile)
		require.NoError(t, err)

		expected := "tap \"homebrew/cask\"\ntap \"homebrew/core\"\n"
		assert.Equal(t, expected, string(content))
	})
}

func TestFilterMasLines(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bundle-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	brewfile := filepath.Join(tmpDir, "Brewfile")

	t.Run("filters out mas lines", func(t *testing.T) {
		content := `tap "homebrew/cask"
brew "git"
cask "firefox"
mas "Xcode", id: 497799835
mas "1Password", id: 1333542190
brew "vim"`
		err := os.WriteFile(brewfile, []byte(content), 0644)
		require.NoError(t, err)

		result, err := filterMasLines(brewfile)
		require.NoError(t, err)

		expected := `tap "homebrew/cask"
brew "git"
cask "firefox"
brew "vim"`
		assert.Equal(t, expected, result)
	})

	t.Run("returns unchanged when no mas lines", func(t *testing.T) {
		content := `tap "homebrew/cask"
brew "git"
cask "firefox"`
		err := os.WriteFile(brewfile, []byte(content), 0644)
		require.NoError(t, err)

		result, err := filterMasLines(brewfile)
		require.NoError(t, err)
		assert.Equal(t, content, result)
	})

	t.Run("handles empty file", func(t *testing.T) {
		err := os.WriteFile(brewfile, []byte(""), 0644)
		require.NoError(t, err)

		result, err := filterMasLines(brewfile)
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		_, err := filterMasLines("/nonexistent/file")
		assert.Error(t, err)
	})
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

func TestParseUvToolList(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected []string
	}{
		{
			name: "parses multiple tools",
			output: `deepeval v3.7.8
- deepeval
vectorcode v0.7.20
- vectorcode
- vectorcode-mcp-server
- vectorcode-server`,
			expected: []string{"deepeval", "vectorcode"},
		},
		{
			name:     "handles empty output",
			output:   "",
			expected: nil,
		},
		{
			name: "handles single tool",
			output: `ruff v0.1.0
- ruff`,
			expected: []string{"ruff"},
		},
		{
			name:     "handles no tools message",
			output:   "No tools installed",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseUvToolList(tt.output)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDiffByBaseName(t *testing.T) {
	t.Run("always reinstalls packages with extras", func(t *testing.T) {
		desired := []string{"vectorcode[lsp,mcp]", "ruff", "black>=1.0"}
		installed := []string{"vectorcode", "ruff"}
		result := diffByBaseName(desired, installed)
		// vectorcode[lsp,mcp] always included (has extras)
		// ruff skipped (installed, no extras)
		// black>=1.0 included (not installed)
		assert.Equal(t, []string{"vectorcode[lsp,mcp]", "black>=1.0"}, result)
	})

	t.Run("finds missing when base not installed", func(t *testing.T) {
		desired := []string{"vectorcode[lsp,mcp]"}
		installed := []string{"ruff"}
		result := diffByBaseName(desired, installed)
		assert.Equal(t, []string{"vectorcode[lsp,mcp]"}, result)
	})

	t.Run("skips simple packages when installed", func(t *testing.T) {
		desired := []string{"ruff", "black"}
		installed := []string{"ruff", "black"}
		result := diffByBaseName(desired, installed)
		assert.Empty(t, result)
	})

	t.Run("version specs without extras skip if installed", func(t *testing.T) {
		desired := []string{"ruff>=1.0", "black@2.0"}
		installed := []string{"ruff", "black"}
		result := diffByBaseName(desired, installed)
		assert.Empty(t, result)
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
		installFn:   func(pkg string, verbose bool, force bool) error { return nil },
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
