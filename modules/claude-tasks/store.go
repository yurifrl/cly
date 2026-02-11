package claudetasks

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type TaskList struct {
	Name string `json:"name"`
}

type Store map[string]TaskList

func FilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "cly", "task-lists.json")
}

func Load(filePath string) (Store, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return Store{}, nil
		}
		return nil, err
	}

	var s Store
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return s, nil
}

func Save(filePath string, s Store) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0o644)
}
