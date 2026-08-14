package dotfiles

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/yurifrl/cly/pkg/style"
)

// ApplyJsoncMapping is the single-mapping equivalent of `cly dotfiles` for
// jsonc -> json copies. It is meant for callers like `cly pi y settings` that
// regenerate exactly one destination after the user edits the source: it
// preserves the same backup, lock-tracking, and confirmation guarantees as a
// full sync without forcing the user through the entire dotfiles loop.
//
// Behavior:
//   - If the existing destination already matches the would-be stripped
//     output, returns immediately with no FS mutation.
//   - Otherwise, when stdin is a TTY, prints the planned backup path and
//     prompts [y/N]. Aborts with an error on anything other than y/yes.
//   - Performs the copy via CopyJsoncToJson (which moves the existing dst
//     into the per-run backup dir before writing).
//   - Updates dotfiles.lock with the new SHA-256 of the source so a later
//     `cly dotfiles` does not re-prompt for this destination.
func ApplyJsoncMapping(m Mapping) (LinkResult, error) {
	if !IsJsoncToJson(m) {
		return LinkResult{}, fmt.Errorf("not a .jsonc -> .json mapping: %s -> %s", m.Source, m.Destination)
	}

	src, err := os.ReadFile(m.Source)
	if err != nil {
		return LinkResult{}, fmt.Errorf("read source: %w", err)
	}
	stripped, serr := StripJSONC(src)
	if serr != nil {
		return LinkResult{}, fmt.Errorf("strip jsonc: %w", serr)
	}
	// Env expansion happens after stripping so the comparison below reflects
	// what will actually be written. expandEnvIfAllowed no-ops when the
	// source carries @no-interpolation.
	expanded, eerr := expandEnvIfAllowed(src, stripped)
	if eerr != nil {
		return LinkResult{}, fmt.Errorf("expand env vars: %w", eerr)
	}

	// No-op short-circuit: existing dst already matches.
	if existing, err := os.ReadFile(m.Destination); err == nil && bytes.Equal(existing, expanded) {
		fmt.Printf("%s %s (no changes)\n",
			style.GreenStyle.Render("✅ Up to date:"),
			shortenPath(m.Destination))
		return LinkResult{Mapping: m, State: StateLinked}, nil
	}

	// Confirmation: only when an existing destination would be moved aside.
	if _, err := os.Lstat(m.Destination); err == nil {
		target, perr := PlanBackupTarget(m.Destination)
		if perr == nil && target != "" {
			fmt.Printf("\n%s\n", style.YellowStyle.Render("📦 About to update existing file:"))
			fmt.Printf("  %s %s\n",
				style.YellowStyle.Render("•"),
				shortenPath(m.Destination))
			fmt.Printf("      %s %s\n",
				style.BlueStyle.Render("→"),
				shortenPath(target))

			if isatty.IsTerminal(os.Stdin.Fd()) {
				fmt.Printf("\nProceed with backup and rewrite? [y/N]: ")
				reader := bufio.NewReader(os.Stdin)
				answer, _ := reader.ReadString('\n')
				answer = strings.ToLower(strings.TrimSpace(answer))
				if answer != "y" && answer != "yes" {
					return LinkResult{Mapping: m, State: StateError, Error: "aborted by user"},
						fmt.Errorf("aborted by user")
				}
			} else {
				fmt.Println("(stdin is not a TTY — proceeding without prompt)")
			}
		}
	}

	res := CopyJsoncToJson(m)
	if res.State == StateError {
		return res, fmt.Errorf("%s", res.Error)
	}

	if res.BackupPath != "" {
		fmt.Printf("%s %s -> %s\n",
			style.YellowStyle.Render("📦 Backed up:"),
			shortenPath(m.Destination),
			shortenPath(res.BackupPath))
	}
	fmt.Printf("%s %s -> %s\n",
		style.GreenStyle.Render("✅ Wrote:"),
		shortenPath(m.Source),
		shortenPath(m.Destination))

	// Refresh this mapping's entry in the lock so a future `cly dotfiles`
	// run sees the source as up-to-date and does not re-prompt for it.
	if err := updateLockJsoncEntry(m); err != nil {
		// Non-fatal: the copy already happened; surface a warning.
		fmt.Printf("%s lock update failed: %v\n",
			style.YellowStyle.Render("⚠️"), err)
	}
	return res, nil
}

// updateLockJsoncEntry merges (or appends) the given jsonc mapping's current
// source hash into dotfiles.lock without disturbing other entries.
func updateLockJsoncEntry(m Mapping) error {
	lockPath, err := lockFilePath()
	if err != nil {
		return err
	}
	lock, err := loadLock(lockPath)
	if err != nil {
		return err
	}
	hash := hashFile(m.Source)
	updated := false
	for i := range lock.JsoncCopies {
		if lock.JsoncCopies[i].Destination == m.Destination {
			lock.JsoncCopies[i].Source = m.Source
			lock.JsoncCopies[i].SourceHash = hash
			updated = true
			break
		}
	}
	if !updated {
		lock.JsoncCopies = append(lock.JsoncCopies, LockEntry{
			Source:      m.Source,
			Destination: m.Destination,
			SourceHash:  hash,
		})
	}
	return saveLock(lockPath, lock)
}
