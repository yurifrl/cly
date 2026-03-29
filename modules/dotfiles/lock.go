package dotfiles

import (
	"encoding/json"
	"os"
	"path/filepath"

	pkgconfig "github.com/yurifrl/cly/pkg/config"
)

// LockEntry represents a source→destination pair that was applied.
type LockEntry struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

// DotfilesLock records everything applied during the last dotfiles run.
type DotfilesLock struct {
	Symlinks        []LockEntry `json:"symlinks"`
	JsoncCopies     []LockEntry `json:"jsonc_copies"`
	Jobs            []string    `json:"jobs"`
	InstallCommands []string    `json:"install_commands"`
	OpMappings      []LockEntry `json:"op_mappings"`
}

// LockDiff holds items that exist in the old lock but not in the new one.
type LockDiff struct {
	RemovedSymlinks        []LockEntry
	RemovedJsoncCopies     []LockEntry
	RemovedJobs            []string
	RemovedInstallCommands []string
	RemovedOpMappings      []LockEntry
}

// lockFilePath returns the path for the dotfiles lock file.
func lockFilePath() (string, error) {
	dataDir := pkgconfig.GetString("app.data_dir")
	if dataDir == "" {
		dataDir = "~/.local/share/cly"
	}
	dataDir = expandTilde(dataDir)
	return filepath.Join(dataDir, "dotfiles/dotfiles.lock"), nil
}

// loadLock reads the lock file. Returns an empty lock if the file does not exist.
func loadLock(path string) (*DotfilesLock, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &DotfilesLock{}, nil
	}
	if err != nil {
		return nil, err
	}
	var lock DotfilesLock
	if err := json.Unmarshal(data, &lock); err != nil {
		// Corrupt lock — start fresh rather than failing hard.
		return &DotfilesLock{}, nil
	}
	return &lock, nil
}

// saveLock writes the lock file to disk.
func saveLock(path string, lock *DotfilesLock) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}

// buildLock creates a new lock snapshot from the current config.
// It only records entries that were actually applied (all mappings present in config).
func buildLock(cfg *Config) *DotfilesLock {
	lock := &DotfilesLock{}

	for _, m := range cfg.Mappings {
		entry := LockEntry{Source: m.Source, Destination: m.Destination}
		if IsJsoncToJson(m) {
			lock.JsoncCopies = append(lock.JsoncCopies, entry)
		} else {
			lock.Symlinks = append(lock.Symlinks, entry)
		}
	}

	for _, j := range cfg.Jobs {
		lock.Jobs = append(lock.Jobs, j.Name)
	}

	lock.InstallCommands = append(lock.InstallCommands, cfg.InstallCommands...)

	for _, op := range cfg.OpMappings {
		lock.OpMappings = append(lock.OpMappings, LockEntry{Source: op.Source, Destination: op.Destination})
	}

	return lock
}

// diffLocks returns items present in old but absent from new (removed entries).
func diffLocks(old, new *DotfilesLock) LockDiff {
	var diff LockDiff
	diff.RemovedSymlinks = removedEntries(old.Symlinks, new.Symlinks)
	diff.RemovedJsoncCopies = removedEntries(old.JsoncCopies, new.JsoncCopies)
	diff.RemovedJobs = removedStrings(old.Jobs, new.Jobs)
	diff.RemovedInstallCommands = removedStrings(old.InstallCommands, new.InstallCommands)
	diff.RemovedOpMappings = removedEntries(old.OpMappings, new.OpMappings)
	return diff
}

func removedEntries(old, new []LockEntry) []LockEntry {
	newSet := make(map[string]bool, len(new))
	for _, e := range new {
		newSet[e.Destination] = true
	}
	var removed []LockEntry
	for _, e := range old {
		if !newSet[e.Destination] {
			removed = append(removed, e)
		}
	}
	return removed
}

func removedStrings(old, new []string) []string {
	newSet := make(map[string]bool, len(new))
	for _, s := range new {
		newSet[s] = true
	}
	var removed []string
	for _, s := range old {
		if !newSet[s] {
			removed = append(removed, s)
		}
	}
	return removed
}
