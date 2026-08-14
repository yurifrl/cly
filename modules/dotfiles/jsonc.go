package dotfiles

import (
	"github.com/yurifrl/cly/pkg/mut"
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
	if err := mut.Remove(m.Destination); err != nil {
		return false
	}
	return true
}

// IsJsoncToJson returns true if the mapping is a JSONC source to JSON destination.
func IsJsoncToJson(m Mapping) bool {
	return strings.HasSuffix(m.Source, ".jsonc") && strings.HasSuffix(m.Destination, ".json")
}

// CopyJsoncToJson reads a JSONC file, strips comments, interpolates $VAR
// and ${VAR} env references (unless @no-interpolation appears in the first
// 10 lines), and writes valid JSON to the destination.
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

	expanded, err := expandEnvIfAllowed(data, stripped)
	if err != nil {
		result.State = StateError
		result.Error = fmt.Sprintf("failed to expand env vars: %v", err)
		return result
	}
	stripped = expanded

	destDir := filepath.Dir(m.Destination)
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		result.CreatedDir = true
	}
	if err := mut.MkdirAll(destDir, 0755); err != nil {
		result.State = StateError
		result.Error = fmt.Sprintf("failed to create parent directory: %v", err)
		return result
	}

	// If the existing destination already has identical content, this is a
	// pure no-op: don't move the file into the backup dir, don't rewrite it.
	if existing, err := os.ReadFile(m.Destination); err == nil && bytes.Equal(existing, stripped) {
		result.State = StateLinked
		return result
	}

	// Move the existing destination (file, dir, or symlink) into the per-run
	// backup directory so we never silently clobber user content. WriteFile
	// would otherwise follow a symlink back into the source .jsonc.
	if _, err := os.Lstat(m.Destination); err == nil {
		result.RemovedExisting = true
		backupPath, berr := BackupExisting(m.Destination)
		if berr != nil {
			result.State = StateError
			result.Error = fmt.Sprintf("failed to back up existing destination: %v", berr)
			return result
		}
		result.BackupPath = backupPath
	}

	if err := mut.WriteFile(m.Destination, stripped, 0644); err != nil {
		result.State = StateError
		result.Error = fmt.Sprintf("failed to write JSON: %v", err)
		return result
	}

	result.State = StateLinked
	return result
}

// StripJSONC removes // line comments, /* */ block comments, and trailing commas from JSONC input.
//
// Note: StripJSONC does NOT expand $VAR/${VAR} env references — that happens
// in CopyJsoncToJson / jsoncContentMatches via expandEnvIfAllowed, after the
// @no-interpolation check.
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

// expandEnvIfAllowed applies os.Expand ($VAR / ${VAR} -> os.Getenv) to the
// stripped JSON unless the original source contains @no-interpolation in the
// first 10 lines. Expanded output is re-validated and re-pretty-printed so
// callers can hand it back to JSON tooling.
func expandEnvIfAllowed(original, stripped []byte) ([]byte, error) {
	if hasNoInterpolation(original) {
		return stripped, nil
	}
	expanded := os.Expand(string(stripped), os.Getenv)

	// Re-validate: expansion may inject characters that break JSON
	// (quotes, backslashes, control chars). Resurface as an error rather
	// than silently writing malformed content.
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, []byte(expanded), "", "  "); err != nil {
		return nil, fmt.Errorf("expansion produced invalid JSON: %w", err)
	}
	pretty.WriteByte('\n')
	return pretty.Bytes(), nil
}

// hasNoInterpolation returns true when @no-interpolation appears in the
// first 10 lines of the raw file, mirroring modules/agents/transform.go.
func hasNoInterpolation(src []byte) bool {
	limit := 10
	start := 0
	for i := 0; i < limit; i++ {
		end := bytes.IndexByte(src[start:], '\n')
		if end < 0 {
			end = len(src) - start
		}
		if bytes.Contains(src[start:start+end], []byte("@no-interpolation")) {
			return true
		}
		if end == len(src)-start && start+end >= len(src) {
			break
		}
		start += end + 1
	}
	return false
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
