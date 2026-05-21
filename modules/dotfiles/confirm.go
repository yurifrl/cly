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

// backupPlan describes a single destination that will be moved into the
// per-run backup directory before being replaced with a new symlink or file.
type backupPlan struct {
	Destination string // path that will be replaced (the user-facing target)
	Kind        string // "file", "directory", or "symlink"
	BackupPath  string // where the existing content will be moved
}

// planBackups walks every mapping and returns the destinations that have
// REAL content (regular files or directories, or symlinks pointing somewhere
// other than the configured source) which would be moved into the backup
// directory if `cly dotfiles` proceeded.
//
// Already-correct symlinks are intentionally skipped: backing them up on
// every sync would be noise. Same for .jsonc -> .json copies whose existing
// destination already matches the bytes that would be written.
func planBackups(cfg *Config, oldLock *DotfilesLock) []backupPlan {
	jsoncHashByDest := map[string]string{}
	if oldLock != nil {
		for _, e := range oldLock.JsoncCopies {
			if e.SourceHash != "" {
				jsoncHashByDest[e.Destination] = e.SourceHash
			}
		}
	}

	var plans []backupPlan
	for _, m := range cfg.Mappings {
		info, err := os.Lstat(m.Destination)
		if err != nil {
			continue // missing destinations don't need backup
		}
		isSymlink := info.Mode()&os.ModeSymlink != 0
		if isSymlink {
			target, err := os.Readlink(m.Destination)
			if err == nil && target == m.Source {
				continue // already-correct symlink, no backup needed
			}
		}
		if IsJsoncToJson(m) {
			// Fast path: stored source hash matches current source
			// hash → we last regenerated this dst from the same input.
			if stored, ok := jsoncHashByDest[m.Destination]; ok && stored != "" {
				if current := hashFile(m.Source); current != "" && current == stored {
					continue
				}
			}
			// Slow path / first run after this feature lands: do a full
			// content check so we still skip already-up-to-date files.
			if jsoncContentMatches(m) {
				continue
			}
		}
		target, err := PlanBackupTarget(m.Destination)
		if err != nil || target == "" {
			continue
		}
		plans = append(plans, backupPlan{
			Destination: m.Destination,
			Kind:        kindOf(info, isSymlink),
			BackupPath:  target,
		})
	}
	return plans
}

// jsoncContentMatches returns true when the destination of a .jsonc -> .json
// mapping already contains the bytes we would otherwise rewrite. Used to
// avoid pointless backups of regenerated JSON files.
func jsoncContentMatches(m Mapping) bool {
	src, err := os.ReadFile(m.Source)
	if err != nil {
		return false
	}
	want, err := StripJSONC(src)
	if err != nil {
		return false
	}
	got, err := os.ReadFile(m.Destination)
	if err != nil {
		return false
	}
	return bytes.Equal(want, got)
}

func kindOf(info os.FileInfo, isSymlink bool) string {
	switch {
	case isSymlink:
		return "symlink"
	case info.IsDir():
		return "directory"
	default:
		return "file"
	}
}

// confirmBackups previews every destination that would be backed up and
// blocks for user confirmation. Returns an error when the user declines so
// the caller can abort the sync without mutating anything.
func confirmBackups(cfg *Config, oldLock *DotfilesLock) error {
	plans := planBackups(cfg, oldLock)
	if len(plans) == 0 {
		return nil
	}

	fmt.Printf("\n%s\n", style.YellowStyle.Render("📦 The following destinations will be MOVED to a backup directory:"))
	for _, p := range plans {
		fmt.Printf("  %s %s  (%s)\n",
			style.YellowStyle.Render("•"),
			shortenPath(p.Destination),
			p.Kind,
		)
		fmt.Printf("      %s %s\n",
			style.BlueStyle.Render("→"),
			shortenPath(p.BackupPath),
		)
	}

	// Non-interactive stdin (piped input, CI, tests) auto-proceeds — the user
	// has already opted into a non-interactive run. Pass --no-it explicitly to
	// skip both the prompt and this preview block.
	if !isatty.IsTerminal(os.Stdin.Fd()) {
		fmt.Println("(stdin is not a TTY — proceeding without prompt)")
		return nil
	}

	fmt.Printf("\nProceed with backup and symlink? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))
	if answer != "y" && answer != "yes" {
		return fmt.Errorf("aborted by user")
	}
	return nil
}
