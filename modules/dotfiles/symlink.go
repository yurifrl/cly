package dotfiles

import (
	"fmt"
	"os"
	"path/filepath"
)

type LinkState string

const (
	StateLinked   LinkState = "linked"
	StateUnlinked LinkState = "unlinked"
	StateMissing  LinkState = "missing"
	StateConflict LinkState = "conflict"
	StateBroken   LinkState = "broken"
	StateError    LinkState = "error"
)

type LinkResult struct {
	Mapping        Mapping
	State          LinkState
	Error          string
	RemovedExisting bool
	CreatedDir     bool
}

func CreateSymlink(m Mapping) LinkResult {
	result := LinkResult{Mapping: m}

	info, err := os.Stat(m.Source)
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

	if info.IsDir() && !m.IsDir {
		result.State = StateError
		result.Error = fmt.Sprintf("source is a directory but missing trailing slash in config")
		return result
	}

	destInfo, err := os.Lstat(m.Destination)
	if err == nil {
		// Force override: remove whatever exists (symlink, file, or directory)
		result.RemovedExisting = true
		if destInfo.IsDir() && destInfo.Mode()&os.ModeSymlink == 0 {
			if err := os.RemoveAll(m.Destination); err != nil {
				result.State = StateError
				result.Error = fmt.Sprintf("failed to remove existing directory: %v", err)
				return result
			}
		} else {
			if err := os.Remove(m.Destination); err != nil {
				result.State = StateError
				result.Error = fmt.Sprintf("failed to remove existing file: %v", err)
				return result
			}
		}
	} else if !os.IsNotExist(err) {
		result.State = StateError
		result.Error = err.Error()
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

	if err := os.Symlink(m.Source, m.Destination); err != nil {
		result.State = StateError
		result.Error = fmt.Sprintf("failed to create symlink: %v", err)
		return result
	}

	result.State = StateLinked
	return result
}

func CheckStatus(m Mapping) LinkResult {
	result := LinkResult{Mapping: m}

	destInfo, err := os.Lstat(m.Destination)
	if os.IsNotExist(err) {
		_, srcErr := os.Stat(m.Source)
		if os.IsNotExist(srcErr) {
			result.State = StateMissing
		} else {
			result.State = StateUnlinked
		}
		return result
	}
	if err != nil {
		result.State = StateError
		result.Error = err.Error()
		return result
	}

	if destInfo.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(m.Destination)
		if err != nil {
			result.State = StateError
			result.Error = err.Error()
			return result
		}

		if _, err := os.Stat(m.Destination); os.IsNotExist(err) {
			result.State = StateBroken
			return result
		}

		if target == m.Source {
			result.State = StateLinked
		} else {
			result.State = StateConflict
			result.Error = fmt.Sprintf("symlink points to %s instead of %s", target, m.Source)
		}
		return result
	}

	result.State = StateConflict
	result.Error = "destination is not a symlink"
	return result
}

func RemoveSymlink(m Mapping) bool {
	info, err := os.Lstat(m.Destination)
	if os.IsNotExist(err) {
		return false
	}
	if err != nil {
		return false
	}

	if info.Mode()&os.ModeSymlink == 0 {
		return false
	}

	if err := os.Remove(m.Destination); err != nil {
		return false
	}

	return true
}
