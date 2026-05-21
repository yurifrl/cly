package dotfiles

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/yurifrl/cly/pkg/mut"
)

// Backup destinations replaced by `cly dotfiles` are moved (not deleted) to a
// timestamped directory under $XDG_DATA_HOME/cly/backups (default
// ~/.local/share/cly/backups). The original absolute path is preserved beneath
// that root so a backup can be located later by mirroring its real path.
//
// Example:
//   /Users/yuri/.hammerspoon/init.lua
//     -> ~/.local/share/cly/backups/dotfiles-20261120-143015/Users/yuri/.hammerspoon/init.lua

var (
	backupMu      sync.Mutex
	backupRoot    string
	backupErr     error
	backupComputed bool // path is computed (cheap)
	backupCreated  bool // dir is created on disk
)

// backupRootPath computes (and caches) the per-run backup root path WITHOUT
// creating the directory. Use this when previewing planned backups.
func backupRootPath() (string, error) {
	backupMu.Lock()
	defer backupMu.Unlock()
	if backupComputed {
		return backupRoot, backupErr
	}
	backupComputed = true

	base, err := backupBaseDir()
	if err != nil {
		backupErr = err
		return "", backupErr
	}
	ts := time.Now().Format("20060102-150405")
	backupRoot = filepath.Join(base, fmt.Sprintf("dotfiles-%s", ts))
	return backupRoot, nil
}

// backupRootDir returns the per-run backup directory, creating it on disk if
// needed. Subsequent calls reuse the same path.
func backupRootDir() (string, error) {
	root, err := backupRootPath()
	if err != nil {
		return "", err
	}
	backupMu.Lock()
	defer backupMu.Unlock()
	if backupCreated {
		return root, nil
	}
	if err := mut.MkdirAll(root, 0755); err != nil {
		return "", fmt.Errorf("create backup dir: %w", err)
	}
	backupCreated = true
	return root, nil
}

// PlanBackupTarget returns the absolute path where `path` would be moved to
// if it were backed up right now. It does not touch the filesystem and does
// not create the backup directory. Returns ("", nil) when `path` does not
// exist and therefore would not be backed up.
func PlanBackupTarget(path string) (string, error) {
	if _, err := os.Lstat(path); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	root, err := backupRootPath()
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	return filepath.Join(root, abs), nil
}

// resetBackupForTest clears the lazy backup state so each test gets a fresh
// directory. Only meant to be called from _test.go files.
func resetBackupForTest() {
	backupMu.Lock()
	defer backupMu.Unlock()
	backupComputed = false
	backupCreated = false
	backupRoot = ""
	backupErr = nil
}

func backupBaseDir() (string, error) {
	if v := os.Getenv("CLY_BACKUP_DIR"); v != "" {
		return v, nil
	}
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return filepath.Join(v, "cly", "backups"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "cly", "backups"), nil
}

// BackupExisting moves `path` (a file, directory, or symlink) into the
// per-run backup directory, preserving its original absolute layout. Use it
// in place of os.Remove/os.RemoveAll whenever an existing destination must be
// cleared before writing a new symlink or file.
//
// In dry-run mode the move is logged via mut and no filesystem changes occur.
// If `path` does not exist, BackupExisting is a no-op and returns ("", nil).
// On success it returns the absolute path of the backup copy so callers can
// surface it to the user.
func BackupExisting(path string) (string, error) {
	if _, err := os.Lstat(path); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	root, err := backupRootDir()
	if err != nil {
		return "", err
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	target := filepath.Join(root, abs)

	if err := mut.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return "", fmt.Errorf("create backup parent: %w", err)
	}

	// If a previous backup already occupies the slot (e.g. same path backed up
	// twice in one run), suffix with a counter so nothing is clobbered.
	target = uniquePath(target)

	if err := mut.Rename(path, target); err != nil {
		// Cross-device fallback: shell out to `mv`, which transparently
		// handles EXDEV by copying then unlinking.
		if isCrossDeviceErr(err) {
			if mvErr := mut.Exec("mv", path, target); mvErr == nil {
				return target, nil
			} else {
				return "", fmt.Errorf("backup move failed: %w", mvErr)
			}
		}
		return "", fmt.Errorf("backup move failed: %w", err)
	}
	return target, nil
}

func uniquePath(p string) string {
	if _, err := os.Lstat(p); os.IsNotExist(err) {
		return p
	}
	for i := 1; i < 10000; i++ {
		candidate := fmt.Sprintf("%s.%d", p, i)
		if _, err := os.Lstat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
	return p
}

func isCrossDeviceErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "cross-device") || strings.Contains(msg, "EXDEV")
}
