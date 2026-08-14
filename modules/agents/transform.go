package agents

import (
	"strings"

	"github.com/yurifrl/cly/pkg/jsonc"
)

// TransformKind identifies the transformation to apply to a file.
type TransformKind int

const (
	TransformNone    TransformKind = iota // Copy as-is
	TransformJSONCSK                      // JSONC → JSON with env interpolation
	TransformSkillMD                      // Strip allowed-tools from SKILL.md frontmatter
)

// TransformJSONC converts JSONC content to valid JSON.
// Strips comments and trailing commas, interpolates $VAR/${VAR} env vars
// (unless @no-interpolation appears in the first 10 lines), and pretty-prints.
func TransformJSONC(data []byte) ([]byte, error) {
	return jsonc.Convert(data)
}

// StripAllowedTools removes the allowed-tools: block from YAML frontmatter.
func StripAllowedTools(data []byte) []byte {
	lines := strings.SplitAfter(string(data), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return data
	}

	var result []string
	inFrontmatter := false
	inAllowedTools := false

	for _, line := range lines {
		trimmed := strings.TrimRight(line, "\n\r")

		if trimmed == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				result = append(result, line)
				continue
			}
			// Closing frontmatter
			inFrontmatter = false
			inAllowedTools = false
			result = append(result, line)
			continue
		}

		if inFrontmatter {
			if strings.HasPrefix(trimmed, "allowed-tools:") {
				inAllowedTools = true
				continue
			}

			if inAllowedTools {
				// Still in the allowed-tools block if line is indented or empty
				if len(trimmed) == 0 || trimmed[0] == ' ' || trimmed[0] == '\t' {
					continue
				}
				// Non-indented line = end of allowed-tools block
				inAllowedTools = false
			}
		}

		result = append(result, line)
	}

	out := strings.Join(result, "")
	// Remove trailing extra newline that splitAfter may produce
	if strings.HasSuffix(out, "\n\n") && !strings.HasSuffix(string(data), "\n\n") {
		out = strings.TrimSuffix(out, "\n")
	}
	return []byte(out)
}
