package helpy

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// DocMeta holds parsed YAML frontmatter from a doc.
type DocMeta struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	URL         string `yaml:"url"`
}

// parseFrontmatter extracts YAML frontmatter from markdown content.
// Returns the metadata and the content with frontmatter stripped.
// If no frontmatter is found, returns empty metadata and the original content.
func parseFrontmatter(content string) (DocMeta, string) {
	var meta DocMeta

	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "---") {
		return meta, content
	}

	// Find the closing ---
	rest := trimmed[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return meta, content
	}

	frontmatter := strings.TrimSpace(rest[:idx])
	body := rest[idx+4:] // skip \n---

	// Parse YAML
	_ = yaml.Unmarshal([]byte(frontmatter), &meta)

	// Preserve leading newline for body
	body = strings.TrimLeft(body, "\r\n")

	return meta, body
}
