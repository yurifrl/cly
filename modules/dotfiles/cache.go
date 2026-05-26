package dotfiles

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os/exec"
	"time"

	"github.com/yurifrl/cly/pkg/mut"
	"github.com/yurifrl/cly/pkg/style"
)

// CacheApplyOptions controls how ApplyCache executes pending entries.
type CacheApplyOptions struct {
	Force    bool
	FailFast bool
}

// cacheShell is the shell used to execute @cache commands. Overridable in
// tests so we don't depend on fish being installed in CI.
var cacheShell = "fish"

// cacheGracePeriod is how long a stale cache lock entry is kept (flagged)
// before it gets dropped during normal sync.
const cacheGracePeriod = 7 * 24 * time.Hour

// hashCacheEntry returns the sha256 hex digest of the entry's command.
// The command text is the only identity for a @cache entry; editing it
// invalidates the cache.
func hashCacheEntry(e CacheEntry) string {
	sum := sha256.Sum256([]byte(e.Command))
	return hex.EncodeToString(sum[:])
}

// shortHash returns the first 8 hex chars of a hash for display.
func shortHash(h string) string {
	if len(h) >= 8 {
		return h[:8]
	}
	return h
}

// truncateCmd renders a command line for log output, truncating to 60
// runes with an ellipsis suffix when longer.
func truncateCmd(cmd string) string {
	const max = 60
	r := []rune(cmd)
	if len(r) <= max {
		return cmd
	}
	return string(r[:max]) + "…"
}

// ApplyCache runs every @cache entry whose hash differs from the lock. With
// opts.Force the hash check is bypassed and every entry runs. Failures are
// streamed live; without --fail-fast they are summarised at the end.
//
// Hash-keyed semantics:
//   - If lock has an entry with the same hash, skip (preserve metadata).
//   - Otherwise run the command, capture exit code, and rewrite the lock
//     entry with hash, command, last-run timestamp and exit code.
//   - Two identical @cache lines hash to the same value: the first run
//     populates the lock entry and the second is a cache hit, so the
//     command runs once.
func ApplyCache(cfg *Config, lock *DotfilesLock, opts CacheApplyOptions) error {
	if len(cfg.CacheEntries) == 0 {
		return nil
	}

	// Index existing lock entries by hash so we can mutate in place.
	idx := make(map[string]int, len(lock.Cache))
	for i, e := range lock.Cache {
		idx[e.Hash] = i
	}

	upsert := func(entry CacheLockEntry) {
		if i, ok := idx[entry.Hash]; ok {
			lock.Cache[i] = entry
			return
		}
		idx[entry.Hash] = len(lock.Cache)
		lock.Cache = append(lock.Cache, entry)
	}

	var errs []error
	for _, entry := range cfg.CacheEntries {
		newHash := hashCacheEntry(entry)
		short := shortHash(newHash)
		display := truncateCmd(entry.Command)

		var prev CacheLockEntry
		if i, ok := idx[newHash]; ok {
			prev = lock.Cache[i]
		}

		if !opts.Force && prev.Hash == newHash {
			fmt.Printf("  %s cache %s: %s\n", style.SubtleStyle.Render("✓"), short, display)
			// Make sure FlaggedForDelete is cleared (entry is in cfg) and
			// Command is up-to-date (legacy entries may lack it).
			prev.FlaggedForDelete = ""
			if prev.Command == "" {
				prev.Command = entry.Command
			}
			upsert(prev)
			continue
		}

		fmt.Printf("  %s cache %s: %s\n", style.BlueStyle.Render("⚙️"), short, display)
		runErr := mut.ExecDir(cfg.BaseDir, cacheShell, "-c", entry.Command)
		exitCode := 0
		if runErr != nil {
			var ee *exec.ExitError
			if errors.As(runErr, &ee) {
				exitCode = ee.ExitCode()
			} else {
				exitCode = 1
			}
		}

		updated := CacheLockEntry{
			Hash:     newHash,
			Command:  entry.Command,
			LastRun:  time.Now().UTC().Format(time.RFC3339),
			ExitCode: exitCode,
		}
		upsert(updated)

		if runErr != nil {
			wrapped := fmt.Errorf("run @cache %s: %w", short, runErr)
			fmt.Printf("  %s %s\n", style.RedStyle.Render("❌"), wrapped)
			if opts.FailFast {
				return wrapped
			}
			errs = append(errs, wrapped)
		}
	}

	if len(errs) > 0 {
		fmt.Printf("%s %d @cache entry/entries failed (continuing without --fail-fast)\n",
			style.YellowStyle.Render("⚠️ "), len(errs))
	}
	return nil
}

// pruneStaleCacheEntries walks lock.Cache, dropping entries whose hash is
// no longer present in cfg according to the flag-then-grace-window
// policy. With hardPrune=true, any entry not in cfg is dropped immediately
// regardless of flag age (used by `cly dotfiles prune --apply`).
//
// Returns the entries that were kept (in cfg or still flagged within
// grace), newly-or-still flagged, and pruned this call.
func pruneStaleCacheEntries(lock *DotfilesLock, cfg *Config, now time.Time, hardPrune bool) (kept, flagged, pruned []CacheLockEntry) {
	cfgHashes := make(map[string]bool, len(cfg.CacheEntries))
	for _, e := range cfg.CacheEntries {
		cfgHashes[hashCacheEntry(e)] = true
	}

	var newCache []CacheLockEntry
	for _, e := range lock.Cache {
		display := fmt.Sprintf("%s: %s", shortHash(e.Hash), truncateCmd(e.Command))
		if cfgHashes[e.Hash] {
			e.FlaggedForDelete = ""
			kept = append(kept, e)
			newCache = append(newCache, e)
			continue
		}

		// Stale candidate.
		if hardPrune {
			fmt.Printf("%s pruned stale cache entry %s\n", style.YellowStyle.Render("🗑️ "), display)
			pruned = append(pruned, e)
			continue
		}

		if e.FlaggedForDelete == "" {
			e.FlaggedForDelete = now.Format(time.RFC3339)
			fmt.Printf("%s cache entry %s no longer in config; will prune in %s\n",
				style.YellowStyle.Render("🚩"), display, cacheGracePeriod)
			flagged = append(flagged, e)
			newCache = append(newCache, e)
			continue
		}

		flaggedAt, perr := time.Parse(time.RFC3339, e.FlaggedForDelete)
		if perr != nil {
			// Corrupt timestamp - reset and keep.
			e.FlaggedForDelete = now.Format(time.RFC3339)
			flagged = append(flagged, e)
			newCache = append(newCache, e)
			continue
		}
		age := now.Sub(flaggedAt)
		if age >= cacheGracePeriod {
			fmt.Printf("%s pruned stale cache entry %s (flagged %s ago)\n",
				style.YellowStyle.Render("🗑️ "), display, age.Truncate(time.Second))
			pruned = append(pruned, e)
			continue
		}
		flagged = append(flagged, e)
		newCache = append(newCache, e)
	}

	lock.Cache = newCache
	return
}
