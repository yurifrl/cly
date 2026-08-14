package dotfiles

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// CacheLockEntry stores the metadata for one @cache entry. Hash is the
// primary key (sha256 of the command); Command is kept for audit so the
// lock is human-readable. LastRun is RFC3339 UTC. FlaggedForDelete
// (RFC3339 UTC) marks an entry whose hash no longer appears in the
// config; after the grace window it gets pruned.
type CacheLockEntry struct {
	Hash             string `json:"hash"`
	Command          string `json:"command,omitempty"`
	LastRun          string `json:"last_run,omitempty"`
	ExitCode         int    `json:"exit_code"`
	FlaggedForDelete string `json:"flagged_for_delete,omitempty"`
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
	Cache           []CacheLockEntry  `json:"cache"`
	InstallCommands []string          `json:"install_commands"`
	Installs        []InstallManifest `json:"installs,omitempty"`
	OpMappings      []LockEntry       `json:"op_mappings"`
}

// UnmarshalJSON handles legacy formats:
//   - `"jobs": ["name", ...]`         (very old, names only) — dropped
//   - `"jobs": [{name,hash}]`          (older format) — hash kept, name dropped
//   - `"cache": [{name,hash,...}]`     (previous format) — hash kept, name dropped
//   - `"cache": [{hash,command,...}]`  (current format)
//
// Entries without a Hash (the bare-string and orphan name-only forms) are
// treated as orphaned legacy and dropped — they would have referenced a
// pre-hash-keyed identity that no longer exists.
func (d *DotfilesLock) UnmarshalJSON(data []byte) error {
	type Alias DotfilesLock
	aux := &struct {
		Jobs  json.RawMessage `json:"jobs"`
		Cache json.RawMessage `json:"cache"`
		*Alias
	}{Alias: (*Alias)(d)}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	raw := aux.Cache
	if len(raw) == 0 || string(raw) == "null" {
		raw = aux.Jobs
	}
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var entries []CacheLockEntry
	if err := json.Unmarshal(raw, &entries); err == nil {
		kept := entries[:0]
		for _, e := range entries {
			if e.Hash == "" {
				continue // orphaned legacy (name-only)
			}
			kept = append(kept, e)
		}
		d.Cache = kept
		return nil
	}
	// Legacy: ["name", ...] — nothing usable now, drop.
	var names []string
	if err := json.Unmarshal(raw, &names); err != nil {
		return err
	}
	d.Cache = nil
	return nil
}

// LockDiff holds items present in the old lock but absent from the new one.
type LockDiff struct {
	RemovedSymlinks        []LockEntry
	RemovedJsoncCopies     []LockEntry
	RemovedCache           []string
	RemovedInstallCommands []string
	RemovedInstalls        []InstallManifest
	RemovedOpMappings      []LockEntry
}

// lockFilePath returns the single lock for the merged config set. Because the
// base config and the per-user overlay are now applied together, their applied
// artifacts must be tracked in one place — otherwise the removal diff would
// never see overlay-managed symlinks and would leave them orphaned. The lock
// lives next to the base config.
func lockFilePath() (string, error) {
	_, applied, err := loadConfig()
	if err != nil {
		return "", err
	}
	if len(applied) == 0 {
		return "", fmt.Errorf("no dotfiles config applied; cannot resolve lock path")
	}
	return lockPathFor(baseConfigPath(applied)), nil
}

// baseConfigPath picks dotfiles.conf out of the applied set, falling back to
// the first applied config when only an overlay matched.
func baseConfigPath(applied []string) string {
	for _, p := range applied {
		if filepath.Base(p) == "dotfiles.conf" {
			return p
		}
	}
	return applied[0]
}

// lockPathFor maps a config path to its sibling lock file.
func lockPathFor(configPath string) string {
	base := filepath.Base(configPath)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	return filepath.Join(filepath.Dir(configPath), name+".lock")
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
	if _, applied, err := loadConfig(); err == nil && len(applied) > 0 {
		paths = append(paths, filepath.Join(filepath.Dir(baseConfigPath(applied)), "dotfiles.lock.json"))
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
		adoptPerUserLock(lock, path)
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
	adoptPerUserLock(&lock, path)
	return &lock, nil
}

// adoptPerUserLock folds a pre-existing dotfiles.<user>.lock into the single
// merged lock and removes it. Before per-user configs became an overlay they
// each kept their own lock; those files still list real symlinks, jsonc copies
// and op outputs on disk. Dropping them would strand those artifacts — the
// removal diff would never see them again — so they are adopted once.
func adoptPerUserLock(lock *DotfilesLock, basePath string) {
	user := effectiveUsername()
	if user == "" {
		return
	}
	userPath := filepath.Join(filepath.Dir(basePath), fmt.Sprintf("dotfiles.%s.lock", user))
	if userPath == basePath {
		return
	}
	data, err := os.ReadFile(userPath)
	if err != nil {
		return
	}
	var prior DotfilesLock
	if err := json.Unmarshal(data, &prior); err != nil {
		return
	}

	lock.Symlinks = appendUniqueEntries(lock.Symlinks, prior.Symlinks)
	lock.JsoncCopies = appendUniqueEntries(lock.JsoncCopies, prior.JsoncCopies)
	lock.OpMappings = appendUniqueEntries(lock.OpMappings, prior.OpMappings)
	lock.InstallCommands = appendUniqueStrings(lock.InstallCommands, prior.InstallCommands)

	seenCache := make(map[string]bool, len(lock.Cache))
	for _, e := range lock.Cache {
		seenCache[e.Hash] = true
	}
	for _, e := range prior.Cache {
		if e.Hash != "" && !seenCache[e.Hash] {
			lock.Cache = append(lock.Cache, e)
			seenCache[e.Hash] = true
		}
	}

	seenInstalls := make(map[string]bool, len(lock.Installs))
	for _, e := range lock.Installs {
		seenInstalls[e.URL] = true
	}
	for _, e := range prior.Installs {
		if !seenInstalls[e.URL] {
			lock.Installs = append(lock.Installs, e)
			seenInstalls[e.URL] = true
		}
	}

	// Persist before dropping the source so a crash cannot lose the entries.
	if err := saveLock(basePath, lock); err != nil {
		return
	}
	_ = mut.Remove(userPath)
}

// appendUniqueEntries appends entries whose Destination is not already tracked.
func appendUniqueEntries(dst, src []LockEntry) []LockEntry {
	seen := make(map[string]bool, len(dst))
	for _, e := range dst {
		seen[e.Destination] = true
	}
	for _, e := range src {
		if !seen[e.Destination] {
			dst = append(dst, e)
			seen[e.Destination] = true
		}
	}
	return dst
}

func appendUniqueStrings(dst, src []string) []string {
	seen := make(map[string]bool, len(dst))
	for _, s := range dst {
		seen[s] = true
	}
	for _, s := range src {
		if !seen[s] {
			dst = append(dst, s)
			seen[s] = true
		}
	}
	return dst
}

// migrateJobsState merges hashes from the legacy jobs-state.json into the
// lock's Cache field and deletes the legacy file. Preserved so users
// upgrading from the @once era do not lose state. Note: the legacy hashes
// were keyed by name, not by command, so the carried-over entries will
// fail the new hash skip check on the next sync and be re-run — which is
// the correct behaviour now that command identity replaces names.
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
	existing := make(map[string]bool, len(lock.Cache))
	for _, e := range lock.Cache {
		existing[e.Hash] = true
	}
	for _, hash := range legacy.Jobs {
		if hash == "" || existing[hash] {
			continue
		}
		lock.Cache = append(lock.Cache, CacheLockEntry{Hash: hash})
		existing[hash] = true
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

	prevByHash := make(map[string]CacheLockEntry)
	if prev != nil {
		for _, e := range prev.Cache {
			if e.Hash == "" {
				continue
			}
			prevByHash[e.Hash] = e
		}
	}
	cfgHashes := make(map[string]bool, len(cfg.CacheEntries))
	seen := make(map[string]bool, len(cfg.CacheEntries))
	for _, e := range cfg.CacheEntries {
		h := hashCacheEntry(e)
		cfgHashes[h] = true
		if seen[h] {
			// Duplicate identical @cache lines collapse into one lock entry.
			continue
		}
		seen[h] = true
		// Only carry forward entries that already exist in prev (with their
		// metadata). New entries are left for ApplyCache to insert after
		// running so we don't accidentally pre-seed a hash match that would
		// short-circuit the first execution.
		if pe, ok := prevByHash[h]; ok {
			lock.Cache = append(lock.Cache, pe)
		}
	}
	// Carry forward stale prev entries so pruneStaleCacheEntries can
	// flag/age them out. They are dropped from the lock once the grace
	// window elapses (or immediately on `cly dotfiles prune --apply`).
	if prev != nil {
		for _, pe := range prev.Cache {
			if pe.Hash == "" {
				continue
			}
			if !cfgHashes[pe.Hash] {
				lock.Cache = append(lock.Cache, pe)
			}
		}
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
	diff.RemovedCache = removedCacheEntries(old.Cache, new.Cache)
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

func removedCacheEntries(old, new []CacheLockEntry) []string {
	newSet := make(map[string]bool, len(new))
	for _, e := range new {
		newSet[e.Hash] = true
	}
	var removed []string
	for _, e := range old {
		if !newSet[e.Hash] {
			removed = append(removed, e.Hash)
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

// lockCacheToMap converts a []CacheLockEntry to a hash→command map.
func lockCacheToMap(entries []CacheLockEntry) map[string]string {
	m := make(map[string]string, len(entries))
	for _, e := range entries {
		m[e.Hash] = e.Command
	}
	return m
}
