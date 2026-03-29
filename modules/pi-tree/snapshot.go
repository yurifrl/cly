package pitree

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// PiSession represents one open π session.
type PiSession struct {
	SessionID string `json:"session_id"`
	StartedAt string `json:"started_at"`
	SizeBytes int64  `json:"size_bytes"`
}

// WorkspaceNode represents one cmux workspace with its π sessions.
type WorkspaceNode struct {
	Name     string      `json:"name"`
	Sessions []PiSession `json:"sessions"`
}

// Snapshot is one versioned capture of the full tree.
type Snapshot struct {
	Version   int             `json:"version"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	Tree      []WorkspaceNode `json:"tree"`
}

type snapshotFile struct {
	Snapshots []Snapshot `json:"snapshots"`
}

func snapshotsPath() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".local", "share", "cly", "pi-tree")
	_ = os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, "snapshots.json")
}

// fingerprint returns a stable hash of the workspace+session-id pairs.
func fingerprint(tree []WorkspaceNode) string {
	var pairs []string
	for _, ws := range tree {
		for _, s := range ws.Sessions {
			pairs = append(pairs, ws.Name+":"+s.SessionID)
		}
	}
	sort.Strings(pairs)
	h := sha256.Sum256([]byte(fmt.Sprint(pairs)))
	return fmt.Sprintf("%x", h[:8])
}

// LoadSnapshots reads all snapshots from disk.
func LoadSnapshots() ([]Snapshot, error) {
	path := snapshotsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var sf snapshotFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, err
	}
	return sf.Snapshots, nil
}

func saveSnapshots(snapshots []Snapshot) error {
	sf := snapshotFile{Snapshots: snapshots}
	data, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(snapshotsPath(), data, 0o644)
}

// Upsert either updates the latest snapshot in-place (same IDs, update sizes/timestamps)
// or appends a new version (IDs changed). Returns (snapshot, isNewVersion).
// If force is true, always creates a new version.
func Upsert(tree []WorkspaceNode, force bool) (Snapshot, bool, error) {
	snapshots, err := LoadSnapshots()
	if err != nil {
		return Snapshot{}, false, err
	}

	now := time.Now()
	fp := fingerprint(tree)

	// Check if latest snapshot has same fingerprint
	if !force && len(snapshots) > 0 {
		latest := snapshots[len(snapshots)-1]
		if fingerprint(latest.Tree) == fp {
			// Same IDs — update sizes and timestamps in-place
			latest.UpdatedAt = now
			latest.Tree = tree
			snapshots[len(snapshots)-1] = latest
			if err := saveSnapshots(snapshots); err != nil {
				return latest, false, err
			}
			return latest, false, nil
		}
	}

	// New version
	version := 1
	if len(snapshots) > 0 {
		version = snapshots[len(snapshots)-1].Version + 1
	}
	snap := Snapshot{
		Version:   version,
		CreatedAt: now,
		UpdatedAt: now,
		Tree:      tree,
	}
	snapshots = append(snapshots, snap)
	if err := saveSnapshots(snapshots); err != nil {
		return snap, true, err
	}
	return snap, true, nil
}
