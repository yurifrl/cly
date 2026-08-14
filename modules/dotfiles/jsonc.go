package dotfiles

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yurifrl/cly/pkg/jsonc"
	"github.com/yurifrl/cly/pkg/mut"
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

// StripJSONC strips comments and trailing commas from JSONC input, with no
// env expansion. Kept as the public entry point for callers that need just
// the strip step; Convert-style expansion happens in expandEnvIfAllowed.
func StripJSONC(input []byte) ([]byte, error) {
	return jsonc.Strip(input)
}

// expandEnvIfAllowed expands $VAR/${VAR} unless the source carries
// @no-interpolation in the first 10 lines. Wraps pkg/jsonc.ExpandEnv.
func expandEnvIfAllowed(original, stripped []byte) ([]byte, error) {
	return jsonc.ExpandEnv(original, stripped)
}

// hasNoInterpolation is kept for direct callers (tests, external helpers)
// that need the marker check without going through Convert.
func hasNoInterpolation(src []byte) bool {
	return jsonc.HasNoInterpolation(src)
}
