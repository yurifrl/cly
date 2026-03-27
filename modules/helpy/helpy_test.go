package helpy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "expands tilde",
			input:    "~/test/file.md",
			expected: filepath.Join(home, "test/file.md"),
		},
		{
			name:     "leaves absolute path unchanged",
			input:    "/tmp/test.md",
			expected: "/tmp/test.md",
		},
		{
			name:     "leaves relative path unchanged",
			input:    "relative/path.md",
			expected: "relative/path.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandPath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFileExists(t *testing.T) {
	// Create temp file
	tmpFile, err := os.CreateTemp("", "helpy-test-*.md")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	t.Run("returns true for existing file", func(t *testing.T) {
		assert.True(t, fileExists(tmpFile.Name()))
	})

	t.Run("returns false for non-existent file", func(t *testing.T) {
		assert.False(t, fileExists("/nonexistent/path/file.md"))
	})
}

func TestReadFileContent(t *testing.T) {
	// Create temp file with content
	tmpFile, err := os.CreateTemp("", "helpy-test-*.md")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	content := "# Test Header\n\nSome content here."
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	t.Run("reads file content successfully", func(t *testing.T) {
		result, err := readFileContent(tmpFile.Name())
		require.NoError(t, err)
		assert.Equal(t, content, result)
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		_, err := readFileContent("/nonexistent/file.md")
		assert.Error(t, err)
	})
}

func TestDefaultFilePath(t *testing.T) {
	assert.Equal(t, "~/DotFiles/HELP.md", defaultFilePath)
}

func TestModelFindMatches(t *testing.T) {
	m, err := initialModel("# Header\n\nSome content\nMore content\nHeader again")
	require.NoError(t, err)

	m.rendered = "# Header\n\nSome content\nMore content\nHeader again"

	t.Run("finds matches case-insensitive", func(t *testing.T) {
		m.searchQuery = "header"
		m.findMatches()
		assert.Len(t, m.matches, 2)
		assert.Equal(t, 0, m.matches[0])
		assert.Equal(t, 4, m.matches[1])
	})

	t.Run("no matches for non-existent term", func(t *testing.T) {
		m.searchQuery = "xyz123"
		m.findMatches()
		assert.Empty(t, m.matches)
	})

	t.Run("empty query returns no matches", func(t *testing.T) {
		m.searchQuery = ""
		m.findMatches()
		assert.Empty(t, m.matches)
	})
}

func TestModelClearSearch(t *testing.T) {
	m, err := initialModel("test")
	require.NoError(t, err)

	m.searchQuery = "query"
	m.matches = []int{1, 2, 3}
	m.matchIndex = 2

	m.clearSearch()

	assert.Empty(t, m.searchQuery)
	assert.Nil(t, m.matches)
	assert.Equal(t, 0, m.matchIndex)
}

func TestExtractHeaders(t *testing.T) {
	content := `# Main Title

Some intro text.

## Section One

Content here.

### Subsection

More content.

## Section Two

Final content.
`

	headers := extractHeaders(content)

	require.Len(t, headers, 4)

	assert.Equal(t, "Main Title", headers[0].title)
	assert.Equal(t, "main-title", headers[0].slug)
	assert.Equal(t, 1, headers[0].level)
	assert.Equal(t, 0, headers[0].line)

	assert.Equal(t, "Section One", headers[1].title)
	assert.Equal(t, "section-one", headers[1].slug)
	assert.Equal(t, 2, headers[1].level)
	assert.Equal(t, 4, headers[1].line)

	assert.Equal(t, "Subsection", headers[2].title)
	assert.Equal(t, "subsection", headers[2].slug)
	assert.Equal(t, 3, headers[2].level)
	assert.Equal(t, 8, headers[2].line)

	assert.Equal(t, "Section Two", headers[3].title)
	assert.Equal(t, "section-two", headers[3].slug)
	assert.Equal(t, 2, headers[3].level)
	assert.Equal(t, 12, headers[3].line)
}

func TestExtractHeadersEmpty(t *testing.T) {
	headers := extractHeaders("no headers here")
	assert.Empty(t, headers)
}

func TestToSlug(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Fish Shell", "fish-shell"},
		{"cmux", "cmux"},
		{"Zellij Session Switching (zswitch)", "zellij-session-switching-zswitch"},
		{"pi-cmux (inside pi)", "pi-cmux-inside-pi"},
		{"I Commands (i prefix)", "i-commands-i-prefix"},
		{"Copy, Cut, Paste (Move Files)", "copy-cut-paste-move-files"},
		{"Kubernetes & Cloud", "kubernetes-cloud"},
		{"nvim-tree (File Explorer)", "nvim-tree-file-explorer"},
		{"Quit & Save", "quit-save"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, toSlug(tt.input))
		})
	}
}

func TestExtractSection(t *testing.T) {
	content := `# Main Title

Some intro text.

## Section One

Content here.

### Subsection

More content.

## Section Two

Final content.`

	t.Run("extracts section with children", func(t *testing.T) {
		section, found := extractSection(content, "section-one")
		require.True(t, found)
		assert.Contains(t, section, "## Section One")
		assert.Contains(t, section, "Content here.")
		assert.Contains(t, section, "### Subsection")
		assert.Contains(t, section, "More content.")
		assert.NotContains(t, section, "Section Two")
	})

	t.Run("extracts leaf section", func(t *testing.T) {
		section, found := extractSection(content, "subsection")
		require.True(t, found)
		assert.Contains(t, section, "### Subsection")
		assert.Contains(t, section, "More content.")
		assert.NotContains(t, section, "Section Two")
	})

	t.Run("extracts last section", func(t *testing.T) {
		section, found := extractSection(content, "section-two")
		require.True(t, found)
		assert.Contains(t, section, "## Section Two")
		assert.Contains(t, section, "Final content.")
	})

	t.Run("returns false for unknown slug", func(t *testing.T) {
		_, found := extractSection(content, "nonexistent")
		assert.False(t, found)
	})
}

func TestIdeFlag(t *testing.T) {
	t.Run("ideFlag is empty by default", func(t *testing.T) {
		assert.Empty(t, ideFlag)
	})

	t.Run("ideDefault returns pi when no env var", func(t *testing.T) {
		t.Setenv("CLY_HELPY_IDE", "")
		assert.Equal(t, "pi", ideDefault())
	})

	t.Run("ideDefault returns env var value", func(t *testing.T) {
		t.Setenv("CLY_HELPY_IDE", "claude")
		assert.Equal(t, "claude", ideDefault())
	})
}

func TestResolveAI(t *testing.T) {
	t.Skip("resolveAI was removed — AI is now handled by pkg/llm directly")
}

func TestBuildAICommand(t *testing.T) {
	t.Skip("buildAICommand was removed — AI is now handled by pkg/llm directly")
}
