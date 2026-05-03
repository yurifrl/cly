package beads

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// persistedState is written to ~/.config/cly/beads/state.json between runs.
// Keep fields small and backward-compatible: missing keys just fall back to
// their zero value.
type persistedState struct {
	LastType     string `json:"last_type,omitempty"`
	LastPriority string `json:"last_priority,omitempty"`
}

func stateFile() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "cly", "beads", "state.json"), nil
}

// loadState reads the persisted state. Returns zero value on any error.
func loadState() persistedState {
	var s persistedState
	path, err := stateFile()
	if err != nil {
		return s
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	_ = json.Unmarshal(b, &s)
	return s
}

// saveState writes the state file best-effort (swallows errors).
func saveState(s persistedState) {
	path, err := stateFile()
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, b, 0o644)
}
