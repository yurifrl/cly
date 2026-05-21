package dotfiles

import (
	"github.com/yurifrl/cly/pkg/mut"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"

	pkgconfig "github.com/yurifrl/cly/pkg/config"
)

// hashFile returns the hex-encoded SHA-256 of the file at `path`. Returns an
// empty string when the file cannot be read — callers treat that as "no
// hash available" rather than an error.
func hashFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// LockEntry represents a source→destination pair that was applied.
// SourceHash is the SHA-256 of the source file at apply time and is currently
// only populated for .jsonc -> .json copies, where it lets us decide whether
// the generated JSON is still in sync without re-parsing the source.
type LockEntry struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	SourceHash  string `json:"source_hash,omitempty"`
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

// lockFilePath returns the path for the dotfiles lock file. The lock lives
// next to dotfiles.conf inside the user's dotfiles repo. Contents are JSON;
// the filename uses the conventional `.lock` suffix.
func lockFilePath() (string, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(configPath), "dotfiles.lock"), nil
}

// legacyLockPaths returns previous filenames in priority order so we can
// migrate them on first read. Order matters: the first existing file wins.
func legacyLockPaths() []string {
	dataDir := pkgconfig.GetString("app.data_dir")
	if dataDir == "" {
		dataDir = "~/.local/share/cly"
	}
	dataDir = expandTilde(dataDir)
	dir := filepath.Join(dataDir, "dotfiles")
	paths := []string{
		filepath.Join(dir, "dotfiles.lock.json"),
		filepath.Join(dir, "dotfiles.lock"),
		filepath.Join(dir, "dotfiles.json"),
	}
	// Also migrate from a repo-side `dotfiles.lock.json` that an interim
	// version of this binary may have written.
	if configPath, err := getConfigPath(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(configPath), "dotfiles.lock.json"))
	}
	return paths
}

// loadLock reads the lock file. Returns an empty lock if the file does not
// exist. Migrates legacy filenames (dotfiles.lock, dotfiles.json) to the
// current `.lock.json` location on first read.
func loadLock(path string) (*DotfilesLock, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		for _, legacy := range legacyLockPaths() {
			if _, lstatErr := os.Stat(legacy); lstatErr == nil {
				if err := mut.MkdirAll(filepath.Dir(path), 0755); err == nil {
					if rerr := mut.Rename(legacy, path); rerr == nil {
						break
					}
				}
			}
		}
	}

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
	if err := mut.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return mut.WriteFile(path, data, 0644)
}

// buildLock creates a new lock snapshot from the current config.
// It only records entries that were actually applied (all mappings present in config).
func buildLock(cfg *Config) *DotfilesLock {
	lock := &DotfilesLock{}

	for _, m := range cfg.Mappings {
		entry := LockEntry{Source: m.Source, Destination: m.Destination}
		if IsJsoncToJson(m) {
			entry.SourceHash = hashFile(m.Source)
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
