package agents

import (
	"encoding/json"
	"os"
	"strings"
)

// TransformKind identifies the transformation to apply to a file.
type TransformKind int

const (
	TransformNone    TransformKind = iota // Copy as-is
	TransformJSONCSK                      // JSONC → JSON with env interpolation
	TransformSkillMD                      // Strip allowed-tools from SKILL.md frontmatter
)

// TransformJSONC converts JSONC content to valid JSON.
// It strips comments, removes trailing commas, interpolates ${VAR} env vars
// (unless @no-interpolation is present in first 10 lines), and pretty-prints.
func TransformJSONC(data []byte) ([]byte, error) {
	content := string(data)

	noInterp := hasNoInterpolation(content)

	// Strip comments and trailing commas
	content = stripJSONCComments(content)

	// Interpolate env vars unless opted out
	if !noInterp {
		content = interpolateEnv(content)
	}

	// Parse and re-marshal for clean formatting
	var v interface{}
	if err := json.Unmarshal([]byte(content), &v); err != nil {
		return nil, err
	}

	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}

	return append(out, '\n'), nil
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

// stripJSONCComments removes // and /* */ comments from JSONC, preserving strings.
// Also removes trailing commas before } and ].
func stripJSONCComments(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	i := 0
	for i < len(s) {
		// String literal — pass through unchanged
		if s[i] == '"' {
			b.WriteByte(s[i])
			i++
			for i < len(s) {
				b.WriteByte(s[i])
				if s[i] == '\\' {
					i++
					if i < len(s) {
						b.WriteByte(s[i])
					}
				} else if s[i] == '"' {
					i++
					break
				}
				i++
			}
			continue
		}

		// Single-line comment
		if i+1 < len(s) && s[i] == '/' && s[i+1] == '/' {
			for i < len(s) && s[i] != '\n' {
				i++
			}
			continue
		}

		// Block comment
		if i+1 < len(s) && s[i] == '/' && s[i+1] == '*' {
			i += 2
			for i+1 < len(s) {
				if s[i] == '*' && s[i+1] == '/' {
					i += 2
					break
				}
				i++
			}
			// consume any remaining newlines that were part of the block comment line
			for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
				i++
			}
			if i < len(s) && s[i] == '\n' {
				// keep the newline but we already consumed the content line
			}
			continue
		}

		b.WriteByte(s[i])
		i++
	}

	return removeTrailingCommas(b.String())
}

// removeTrailingCommas removes commas before } and ] (with optional whitespace/newlines between).
func removeTrailingCommas(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		if runes[i] == ',' {
			// Look ahead past whitespace/newlines for } or ]
			j := i + 1
			for j < len(runes) && (runes[j] == ' ' || runes[j] == '\t' || runes[j] == '\n' || runes[j] == '\r') {
				j++
			}
			if j < len(runes) && (runes[j] == '}' || runes[j] == ']') {
				// Skip the trailing comma
				continue
			}
		}
		b.WriteRune(runes[i])
	}

	return b.String()
}

// interpolateEnv replaces ${VAR} with os.Getenv(VAR).
func interpolateEnv(s string) string {
	return os.Expand(s, os.Getenv)
}

// hasNoInterpolation checks if @no-interpolation appears in the first 10 lines.
func hasNoInterpolation(s string) bool {
	lines := strings.SplitN(s, "\n", 12)
	limit := 10
	if len(lines) < limit {
		limit = len(lines)
	}
	for i := 0; i < limit; i++ {
		if strings.Contains(lines[i], "@no-interpolation") {
			return true
		}
	}
	return false
}
