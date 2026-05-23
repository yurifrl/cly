package dotfiles

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"

	pkgconfig "github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/mut"
)

func hashFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// LockEntry represents a source→destination pair that was applied.
type LockEntry struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	SourceHash  string `json:"source_hash,omitempty"`
}

// JobLockEntry stores a job name and its definition hash (used to detect reruns).
type JobLockEntry struct {
	Name string `json:"name"`
	Hash string `json:"hash,omitempty"`
}

// ScriptManifest records what an install script installs so it can be reversed.
type ScriptManifest struct {
	Binaries       []string `json:"binaries,omitempty"`
	Dirs           []string `json:"dirs,omitempty"`
	Files          []string `json:"files,omitempty"`
	ShellRCChanges []string `json:"shell_rc_changes,omitempty"`
	FetchesOther   bool     `json:"fetches_other_scripts,omitempty"`
	NeedsSudo      bool     `json:"needs_sudo,omitempty"`
}

// InstallManifest is the lock entry for an @install directive.
type InstallManifest struct {
	URL      string          `json:"url"`
	SHA      string          `json:"sha"`
	Bypassed bool            `json:"bypassed,omitempty"`
	Manifest *ScriptManifest `json:"manifest,omitempty"`
}

// DotfilesLock records everything applied during the last dotfiles run.
type DotfilesLock struct {
	Symlinks        []LockEntry       `json:"symlinks"`
	JsoncCopies     []LockEntry       `json:"jsonc_copies"`
	Jobs            []JobLockEntry    `json:"jobs"`
	InstallCommands []string          `json:"install_commands"`
	Installs        []InstallManifest `json:"installs,omitempty"`
	OpMappings      []LockEntry       `json:"op_mappings"`
}

// UnmarshalJSON handles the legacy []string format for the Jobs field.
func (d *DotfilesLock) UnmarshalJSON(data []byte) error {
	type Alias DotfilesLock
	aux := &struct {
		Jobs json.RawMessage `json:"jobs"`
		*Alias
	}{Alias: (*Alias)(d)}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	if len(aux.Jobs) == 0 || string(aux.Jobs) == "null" {
		return nil
	}
	var entries []JobLockEntry
	if err := json.Unmarshal(aux.Jobs, &entries); err == nil {
		d.Jobs = entries
		return nil
	}
	// Legacy: ["name", ...]
	var names []string
	if err := json.Unmarshal(aux.Jobs, &names); err != nil {
		return err
	}
	d.Jobs = make([]JobLockEntry, len(names))
	for i, n := range names {
		d.Jobs[i] = JobLockEntry{Name: n}
	}
	return nil
}

// LockDiff holds items present in the old lock but absent from the new one.
type LockDiff struct {
	RemovedSymlinks        []LockEntry
	RemovedJsoncCopies     []LockEntry
	RemovedJobs            []string
	RemovedInstallCommands []string
	RemovedInstalls        []InstallManifest
	RemovedOpMappings      []LockEntry
}

func lockFilePath() (string, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(configPath), "dotfiles.lock"), nil
}

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
	if configPath, err := getConfigPath(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(configPath), "dotfiles.lock.json"))
	}
	return paths
}

// legacyJobsStatePath returns the path for the old jobs-state.json file.
func legacyJobsStatePath() string {
	dataDir := pkgconfig.GetString("app.data_dir")
	if dataDir == "" {
		dataDir = "~/.local/share/cly"
	}
	return filepath.Join(expandTilde(dataDir), "dotfiles/jobs-state.json")
}

// loadLock reads the lock file. Returns an empty lock if the file does not
// exist. Migrates legacy filenames and merges the old jobs-state.json.
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
		lock := &DotfilesLock{}
		migrateJobsState(lock)
		return lock, nil
	}
	if err != nil {
		return nil, err
	}
	var lock DotfilesLock
	if err := json.Unmarshal(data, &lock); err != nil {
		lock = DotfilesLock{}
	}
	migrateJobsState(&lock)
	return &lock, nil
}

// migrateJobsState merges hashes from the legacy jobs-state.json into the lock
// and deletes the legacy file.
func migrateJobsState(lock *DotfilesLock) {
	path := legacyJobsStatePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var legacy struct {
		Jobs map[string]string `json:"jobs"`
	}
	if json.Unmarshal(data, &legacy) != nil || len(legacy.Jobs) == 0 {
		_ = os.Remove(path)
		return
	}
	existing := make(map[string]bool, len(lock.Jobs))
	for _, e := range lock.Jobs {
		existing[e.Name] = true
	}
	for name, hash := range legacy.Jobs {
		if !existing[name] {
			lock.Jobs = append(lock.Jobs, JobLockEntry{Name: name, Hash: hash})
		}
	}
	_ = os.Remove(path)
}

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
// prev is used to carry job hashes forward (jobs that still exist keep their hash).
func buildLock(cfg *Config, prev *DotfilesLock) *DotfilesLock {
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

	prevHashes := lockJobsToMap(nil)
	if prev != nil {
		prevHashes = lockJobsToMap(prev.Jobs)
	}
	for _, j := range cfg.Jobs {
		lock.Jobs = append(lock.Jobs, JobLockEntry{Name: j.Name, Hash: prevHashes[j.Name]})
	}

	lock.InstallCommands = append(lock.InstallCommands, cfg.InstallCommands...)

	if prev != nil {
		prevInstalls := make(map[string]InstallManifest, len(prev.Installs))
		for _, e := range prev.Installs {
			prevInstalls[e.URL] = e
		}
		for _, inst := range cfg.Installs {
			if e, ok := prevInstalls[inst.URL]; ok {
				lock.Installs = append(lock.Installs, e)
			}
		}
	}

	for _, op := range cfg.OpMappings {
		lock.OpMappings = append(lock.OpMappings, LockEntry{Source: op.Source, Destination: op.Destination})
	}

	return lock
}

func diffLocks(old, new *DotfilesLock) LockDiff {
	var diff LockDiff
	diff.RemovedSymlinks = removedEntries(old.Symlinks, new.Symlinks)
	diff.RemovedJsoncCopies = removedEntries(old.JsoncCopies, new.JsoncCopies)
	diff.RemovedJobs = removedJobEntries(old.Jobs, new.Jobs)
	diff.RemovedInstallCommands = removedStrings(old.InstallCommands, new.InstallCommands)
	diff.RemovedInstalls = removedInstallEntries(old.Installs, new.Installs)
	diff.RemovedOpMappings = removedEntries(old.OpMappings, new.OpMappings)
	return diff
}

func removedInstallEntries(old, new []InstallManifest) []InstallManifest {
	newSet := make(map[string]bool, len(new))
	for _, e := range new {
		newSet[e.URL] = true
	}
	var removed []InstallManifest
	for _, e := range old {
		if !newSet[e.URL] {
			removed = append(removed, e)
		}
	}
	return removed
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

func removedJobEntries(old, new []JobLockEntry) []string {
	newSet := make(map[string]bool, len(new))
	for _, e := range new {
		newSet[e.Name] = true
	}
	var removed []string
	for _, e := range old {
		if !newSet[e.Name] {
			removed = append(removed, e.Name)
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

// lockJobsToMap converts a []JobLockEntry to a name→hash map.
func lockJobsToMap(entries []JobLockEntry) map[string]string {
	m := make(map[string]string, len(entries))
	for _, e := range entries {
		m[e.Name] = e.Hash
	}
	return m
}

// mapToLockJobs converts a name→hash map to []JobLockEntry.
func mapToLockJobs(m map[string]string) []JobLockEntry {
	entries := make([]JobLockEntry, 0, len(m))
	for name, hash := range m {
		entries = append(entries, JobLockEntry{Name: name, Hash: hash})
	}
	return entries
}
