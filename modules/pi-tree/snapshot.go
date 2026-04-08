package pitree

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// PiSession represents one open π session.
type PiSession struct {
	SessionID   string `json:"session_id"`
	SessionName string `json:"session_name,omitempty"` // from session_info in JSONL
	StartedAt   string `json:"started_at"`
	SizeBytes   int64  `json:"size_bytes"`
	SurfaceRef  string `json:"surface_ref,omitempty"` // e.g. "surface:65"
	FilePath    string `json:"file_path,omitempty"`   // full path to the .jsonl file
	IsOpen      bool   `json:"is_open,omitempty"`     // true if pi process is running for this session's workspace
}

// WorkspaceNode represents one cmux workspace with its π sessions.
type WorkspaceNode struct {
	Name         string      `json:"name"`
	WorkspaceRef string      `json:"workspace_ref,omitempty"` // e.g. "workspace:18"
	Sessions     []PiSession `json:"sessions"`
}

// Snapshot is one versioned capture of the full tree.
type Snapshot struct {
	Version   int             `json:"version"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	Deleted   bool            `json:"deleted,omitempty"`
	AutoSave  bool            `json:"auto_save,omitempty"`
	Tree      []WorkspaceNode `json:"tree"`
}

type snapshotFile struct {
	Snapshots []Snapshot `json:"snapshots"`
}

var snapshotsPath = func() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".local", "share", "cly", "pi-tree")
	_ = os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, "snapshots.json")
}

var lastHistIdxPath = func() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "cly", "pi-tree", "last-hist-idx")
}

// SaveLastHistIdx persists the last selected history version. -1 means Latest (live).
func SaveLastHistIdx(version int) {
	_ = os.WriteFile(lastHistIdxPath(), []byte(fmt.Sprintf("%d", version)), 0o644)
}

// LoadLastHistIdx reads the last selected history version.
func LoadLastHistIdx() int {
	data, err := os.ReadFile(lastHistIdxPath())
	if err != nil {
		return -1
	}
	var v int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &v); err != nil {
		return -1
	}
	return v
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

	// New version — use Unix timestamp as version ID
	// AutoSave is true when not forced (i.e. automatic upsert on launch)
	version := int(now.Unix())
	snap := Snapshot{
		Version:   version,
		CreatedAt: now,
		UpdatedAt: now,
		AutoSave:  !force,
		Tree:      tree,
	}
	snapshots = append(snapshots, snap)
	if err := saveSnapshots(snapshots); err != nil {
		return snap, true, err
	}
	return snap, true, nil
}

// DeleteSnapshot removes a snapshot by version number.
// DeleteSnapshot soft-deletes a snapshot by marking it as deleted.
// The data is preserved and can be recovered.
func DeleteSnapshot(version int) error {
	snapshots, err := LoadSnapshots()
	if err != nil {
		return err
	}
	found := false
	for i, s := range snapshots {
		if s.Version == version {
			snapshots[i].Deleted = true
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("version %d not found", version)
	}
	return saveSnapshots(snapshots)
}

// ActiveSnapshots returns only non-deleted snapshots.
func ActiveSnapshots(snapshots []Snapshot) []Snapshot {
	var active []Snapshot
	for _, s := range snapshots {
		if !s.Deleted {
			active = append(active, s)
		}
	}
	return active
}
