// Package jsonc converts JSONC source (JSON with // and /* */ comments,
// trailing commas, and $VAR/${VAR} env placeholders) into valid,
// pretty-printed JSON. It is the single implementation shared by
// modules/dotfiles (dotfiles sync, `cly pi y settings`) and modules/agents
// (JSONC-sidecar transforms for IDE/agent configs).
//
// Behavior:
//   - Strips // line comments, /* */ block comments, and trailing commas
//     before } or ].
//   - Expands $VAR and ${VAR} from the process environment via os.Expand,
//     unless the literal string `@no-interpolation` appears in the first
//     10 lines of the source (in which case env expansion is skipped).
//   - Re-validates the output as JSON after expansion so a value containing
//     a quote or newline surfaces as an error instead of writing malformed
//     output.
//   - Pretty-prints with two-space indentation and a trailing newline.
package jsonc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

// Convert runs the full pipeline: strip comments and trailing commas,
// expand env vars (unless @no-interpolation), validate, pretty-print.
func Convert(data []byte) ([]byte, error) {
	stripped, err := Strip(data)
	if err != nil {
		return nil, err
	}
	return ExpandEnv(data, stripped)
}

// Strip removes comments and trailing commas from JSONC and pretty-prints
// the result. It does NOT expand env vars — use Convert for that, or call
// ExpandEnv with the original source if you need to keep strip and expand
// as separate steps (e.g. the dotfiles no-op fast path).
func Strip(input []byte) ([]byte, error) {
	stripped := stripComments(input)
	cleaned := removeTrailingCommas(stripped)

	var pretty bytes.Buffer
	if err := json.Indent(&pretty, cleaned, "", "  "); err != nil {
		return nil, fmt.Errorf("result is not valid JSON: %w", err)
	}
	pretty.WriteByte('\n')
	return pretty.Bytes(), nil
}

// ExpandEnv applies os.Expand ($VAR/${VAR} -> os.Getenv) to stripped JSON
// unless the original source contains @no-interpolation in its first 10
// lines. Re-validates and re-pretty-prints so expansion cannot silently
// produce malformed JSON.
func ExpandEnv(original, stripped []byte) ([]byte, error) {
	if HasNoInterpolation(original) {
		return stripped, nil
	}
	expanded := os.Expand(string(stripped), os.Getenv)

	var pretty bytes.Buffer
	if err := json.Indent(&pretty, []byte(expanded), "", "  "); err != nil {
		return nil, fmt.Errorf("expansion produced invalid JSON: %w", err)
	}
	pretty.WriteByte('\n')
	return pretty.Bytes(), nil
}

// HasNoInterpolation reports whether @no-interpolation appears in the first
// 10 lines of src. Mirrors the convention from modules/agents/transform.go.
func HasNoInterpolation(src []byte) bool {
	limit := 10
	start := 0
	for i := 0; i < limit; i++ {
		if start >= len(src) {
			break
		}
		end := bytes.IndexByte(src[start:], '\n')
		var line []byte
		if end < 0 {
			line = src[start:]
		} else {
			line = src[start : start+end]
		}
		if bytes.Contains(line, []byte("@no-interpolation")) {
			return true
		}
		if end < 0 {
			break
		}
		start += end + 1
	}
	return false
}

// stripComments removes // and /* */ comments while preserving string
// literals (so URLs and patterns containing slashes pass through untouched).
func stripComments(input []byte) []byte {
	var buf bytes.Buffer
	buf.Grow(len(input))

	i := 0
	n := len(input)
	for i < n {
		// String literal — copy verbatim, including escape sequences.
		if input[i] == '"' {
			buf.WriteByte(input[i])
			i++
			for i < n {
				if input[i] == '\\' && i+1 < n {
					buf.WriteByte(input[i])
					buf.WriteByte(input[i+1])
					i += 2
					continue
				}
				buf.WriteByte(input[i])
				if input[i] == '"' {
					i++
					break
				}
				i++
			}
			continue
		}

		// Line comment.
		if i+1 < n && input[i] == '/' && input[i+1] == '/' {
			i += 2
			for i < n && input[i] != '\n' {
				i++
			}
			continue
		}

		// Block comment.
		if i+1 < n && input[i] == '/' && input[i+1] == '*' {
			i += 2
			for i+1 < n {
				if input[i] == '*' && input[i+1] == '/' {
					i += 2
					break
				}
				i++
			}
			continue
		}

		buf.WriteByte(input[i])
		i++
	}
	return buf.Bytes()
}

// removeTrailingCommas drops commas that appear before ] or } (with
// optional whitespace between), which JSONC allows and JSON forbids.
func removeTrailingCommas(input []byte) []byte {
	var buf bytes.Buffer
	buf.Grow(len(input))

	n := len(input)
	for i := 0; i < n; i++ {
		if input[i] == ',' {
			j := i + 1
			for j < n && (input[j] == ' ' || input[j] == '\t' || input[j] == '\n' || input[j] == '\r') {
				j++
			}
			if j < n && (input[j] == ']' || input[j] == '}') {
				continue
			}
		}
		buf.WriteByte(input[i])
	}
	return buf.Bytes()
}

