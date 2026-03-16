package helpy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseFrontmatter_Valid(t *testing.T) {
	content := `---
name: "Test Doc"
description: "A test document"
url: "https://example.com"
---
# Hello World

This is the body.`

	meta, body := parseFrontmatter(content)

	assert.Equal(t, "Test Doc", meta.Name)
	assert.Equal(t, "A test document", meta.Description)
	assert.Equal(t, "https://example.com", meta.URL)
	assert.Contains(t, body, "# Hello World")
	assert.Contains(t, body, "This is the body.")
	assert.NotContains(t, body, "---")
	assert.NotContains(t, body, "name:")
}

func TestParseFrontmatter_NoFrontmatter(t *testing.T) {
	content := `# Hello World

This is just a regular markdown file.`

	meta, body := parseFrontmatter(content)

	assert.Equal(t, "", meta.Name)
	assert.Equal(t, "", meta.Description)
	assert.Equal(t, "", meta.URL)
	assert.Equal(t, content, body)
}

func TestParseFrontmatter_PartialFields(t *testing.T) {
	content := `---
name: "Only Name"
---
# Content`

	meta, body := parseFrontmatter(content)

	assert.Equal(t, "Only Name", meta.Name)
	assert.Equal(t, "", meta.Description)
	assert.Equal(t, "", meta.URL)
	assert.Contains(t, body, "# Content")
}

func TestParseFrontmatter_EmptyContent(t *testing.T) {
	meta, body := parseFrontmatter("")

	assert.Equal(t, "", meta.Name)
	assert.Equal(t, "", body)
}

func TestParseFrontmatter_OnlyDelimiters(t *testing.T) {
	content := "---\n---\n# Hello"

	meta, body := parseFrontmatter(content)

	// Empty frontmatter is valid
	assert.Equal(t, "", meta.Name)
	assert.Contains(t, body, "# Hello")
}

func TestParseFrontmatter_UnclosedDelimiter(t *testing.T) {
	content := `---
name: "Test"
# This has no closing delimiter`

	meta, body := parseFrontmatter(content)

	// Should return original content when no closing ---
	assert.Equal(t, "", meta.Name)
	assert.Equal(t, content, body)
}
