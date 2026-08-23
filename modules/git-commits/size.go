package gitcommits

import (
	"fmt"
	"os"
	"path/filepath"
)

// fileSize returns the on-disk size of a file relative to the repo root.
// Missing files (e.g. deletions) report 0.
func fileSize(path string) int64 {
	info, err := os.Stat(filepath.Join(repoRoot(), path))
	if err != nil {
		return 0
	}
	return info.Size()
}

// changesetSize sums the on-disk sizes of all non-deleted files in the changeset.
func changesetSize(cs *Changeset) int64 {
	var total int64
	for _, f := range cs.Files {
		if f.Status == StatusDeleted {
			continue
		}
		total += fileSize(f.Path)
	}
	return total
}

// groupSize sums the on-disk sizes of all non-deleted files in a commit group.
func groupSize(g CommitGroup) int64 {
	var total int64
	for _, f := range g.Files {
		if f.Status == StatusDeleted {
			continue
		}
		total += fileSize(f.Path)
	}
	return total
}

// humanSize formats a byte count as a short human-readable string.
func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}
