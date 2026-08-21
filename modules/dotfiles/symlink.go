package dotfiles

import (
	"fmt"
	"github.com/yurifrl/cly/pkg/mut"
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
	Mapping         Mapping
	State           LinkState
	Error           string
	RemovedExisting bool
	BackupPath      string
	CreatedDir      bool
}

// expectedDestTarget returns the string a destination symlink should carry:
// the source path itself, or the resolved target when the source leaf is a
// symlink (follows relative and absolute links, chains included, capped
// against cycles).
func expectedDestTarget(source string) string {
	cur := source
	visited := map[string]bool{}
	for hops := 0; hops < 40; hops++ {
		info, err := os.Lstat(cur)
		if err != nil || info.Mode()&os.ModeSymlink == 0 {
			return cur
		}
		if visited[cur] {
			return cur // cycle; os.Stat on such a source errors earlier
		}
		visited[cur] = true
		target, err := os.Readlink(cur)
		if err != nil {
			return cur
		}
		if filepath.IsAbs(target) {
			cur = filepath.Clean(target)
		} else {
			cur = filepath.Clean(filepath.Join(filepath.Dir(cur), target))
		}
	}
	return cur
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

	linkTarget := expectedDestTarget(m.Source)

	destInfo, err := os.Lstat(m.Destination)
	if err == nil {
		// Already-correct symlink? No-op. Don't move, don't recreate — we
		// would only end up briefly removing it and putting it right back.
		if destInfo.Mode()&os.ModeSymlink != 0 {
			if target, lerr := os.Readlink(m.Destination); lerr == nil && target == linkTarget {
				result.State = StateLinked
				return result
			}
		}
		// Otherwise force override: move whatever exists (file, dir, or
		// symlink pointing somewhere else) into the per-run backup dir.
		result.RemovedExisting = true
		_ = destInfo // kept for potential future per-type handling
		backupPath, berr := BackupExisting(m.Destination)
		if berr != nil {
			result.State = StateError
			result.Error = fmt.Sprintf("failed to back up existing destination: %v", berr)
			return result
		}
		result.BackupPath = backupPath
	} else if !os.IsNotExist(err) {
		result.State = StateError
		result.Error = err.Error()
		return result
	}

	destDir := filepath.Dir(m.Destination)
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		result.CreatedDir = true
	}
	if err := mut.MkdirAll(destDir, 0755); err != nil {
		result.State = StateError
		result.Error = fmt.Sprintf("failed to create parent directory: %v", err)
		return result
	}

	if err := mut.Symlink(linkTarget, m.Destination); err != nil {
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

		expected := expectedDestTarget(m.Source)
		if target == expected {
			result.State = StateLinked
		} else {
			result.State = StateConflict
			result.Error = fmt.Sprintf("symlink points to %s instead of %s", target, expected)
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

	if err := mut.Remove(m.Destination); err != nil {
		return false
	}

	return true
}
