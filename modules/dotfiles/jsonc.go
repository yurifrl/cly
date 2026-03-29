package dotfiles

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RemoveJsoncCopy deletes the generated JSON destination file for a JSONC mapping.
// Returns true if the file was removed, false if it didn't exist or couldn't be removed.
func RemoveJsoncCopy(m Mapping) bool {
	if err := os.Remove(m.Destination); err != nil {
		return false
	}
	return true
}

// IsJsoncToJson returns true if the mapping is a JSONC source to JSON destination.
func IsJsoncToJson(m Mapping) bool {
	return strings.HasSuffix(m.Source, ".jsonc") && strings.HasSuffix(m.Destination, ".json")
}

// CopyJsoncToJson reads a JSONC file, strips comments, and writes valid JSON to the destination.
func CopyJsoncToJson(m Mapping) LinkResult {
	result := LinkResult{Mapping: m}

	data, err := os.ReadFile(m.Source)
	if os.IsNotExist(err) {
		result.State = StateMissing
		result.Error = "source does not exist"
		return result
	}
	if err != nil {
		result.State = StateError
		result.Error = err.Error()
		return result
	}

	stripped, err := StripJSONC(data)
	if err != nil {
		result.State = StateError
		result.Error = fmt.Sprintf("failed to strip JSONC comments: %v", err)
		return result
	}

	destDir := filepath.Dir(m.Destination)
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		result.CreatedDir = true
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		result.State = StateError
		result.Error = fmt.Sprintf("failed to create parent directory: %v", err)
		return result
	}

	// Check if destination already exists (file or symlink)
	if _, err := os.Lstat(m.Destination); err == nil {
		result.RemovedExisting = true
	}

	if err := os.WriteFile(m.Destination, stripped, 0644); err != nil {
		result.State = StateError
		result.Error = fmt.Sprintf("failed to write JSON: %v", err)
		return result
	}

	result.State = StateLinked
	return result
}

// StripJSONC removes // line comments, /* */ block comments, and trailing commas from JSONC input.
func StripJSONC(input []byte) ([]byte, error) {
	var buf bytes.Buffer
	i := 0
	n := len(input)

	for i < n {
		// String literal — copy verbatim
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

		// Line comment
		if i+1 < n && input[i] == '/' && input[i+1] == '/' {
			i += 2
			for i < n && input[i] != '\n' {
				i++
			}
			continue
		}

		// Block comment
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

	// Remove trailing commas before ] or }
	cleaned := removeTrailingCommas(buf.Bytes())

	// Pretty-print for readability
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, cleaned, "", "  "); err != nil {
		return nil, fmt.Errorf("result is not valid JSON: %w", err)
	}

	pretty.WriteByte('\n')
	return pretty.Bytes(), nil
}

// removeTrailingCommas removes commas that appear before ] or } (with optional whitespace between).
func removeTrailingCommas(input []byte) []byte {
	var buf bytes.Buffer
	n := len(input)

	for i := 0; i < n; i++ {
		if input[i] == ',' {
			// Look ahead past whitespace for ] or }
			j := i + 1
			for j < n && (input[j] == ' ' || input[j] == '\t' || input[j] == '\n' || input[j] == '\r') {
				j++
			}
			if j < n && (input[j] == ']' || input[j] == '}') {
				// Skip the comma, keep the whitespace
				continue
			}
		}
		buf.WriteByte(input[i])
	}

	return buf.Bytes()
}
