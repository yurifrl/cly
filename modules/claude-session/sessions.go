package claudesession

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Entry struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Description string    `json:"description,omitempty"`
	SavedAt     time.Time `json:"saved_at,omitempty"`
}

type Sessions map[string]Entry

func FilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "cly", "sessions.json")
}

// filePathFn is the active file path resolver. Overridable in tests.
var filePathFn = FilePath

func filterByPath(s Sessions, path string) Sessions {
	out := Sessions{}
	for k, e := range s {
		if e.Path == path {
			out[k] = e
		}
	}
	return out
}

func findEntry(s Sessions, query string) *Entry {
	if e := FindByName(s, query); e != nil {
		return e
	}
	return FindByID(s, query)
}

func Load(filePath string) (Sessions, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return Sessions{}, nil
		}
		return nil, err
	}

	var s Sessions
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return s, nil
}

func Save(filePath string, s Sessions) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0o644)
}

func FindByName(s Sessions, name string) *Entry {
	e, ok := s[name]
	if !ok {
		return nil
	}
	return &e
}

func FindByID(s Sessions, id string) *Entry {
	for _, e := range s {
		if e.ID == id {
			return &e
		}
	}
	return nil
}

func Remove(s Sessions, name string) Sessions {
	delete(s, name)
	return s
}
