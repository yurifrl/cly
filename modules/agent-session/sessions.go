package agentsession

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Entry struct {
	ID          string            `json:"id"`
	Name        string            `json:"name,omitempty"`
	Provider    string            `json:"provider,omitempty"`
	Path        string            `json:"path"`
	Description string            `json:"description,omitempty"`
	SavedAt     time.Time         `json:"saved_at,omitempty"`
	Meta        map[string]string `json:"meta,omitempty"`
}

type Sessions map[string]Entry

func FilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "cly", "sessions.json")
}

// filePathFn is the active file path resolver. Overridable in tests.
var filePathFn = FilePath

func sessionKey(provider, name string) string {
	return normalizeProvider(provider) + ":" + name
}

func effectiveProvider(entry Entry) string {
	return normalizeProvider(entry.Provider)
}

func filterByPath(s Sessions, path string) Sessions {
	out := Sessions{}
	for k, e := range s {
		if e.Path == path {
			out[k] = e
		}
	}
	return out
}

func filterByProvider(s Sessions, provider string) Sessions {
	provider = normalizeProvider(provider)
	out := Sessions{}
	for k, e := range s {
		if effectiveProvider(e) == provider {
			if e.Provider == "" {
				e.Provider = provider
			}
			out[k] = e
		}
	}
	return out
}

func findEntry(s Sessions, provider, query string) *Entry {
	if e := FindByNameForProvider(s, provider, query); e != nil {
		return e
	}
	return FindByIDForProvider(s, provider, query)
}

func findByNameWithKey(s Sessions, provider, name string) (string, *Entry) {
	provider = normalizeProvider(provider)
	for key, e := range s {
		if e.Name != name {
			continue
		}
		if effectiveProvider(e) != provider {
			continue
		}
		entry := e
		if entry.Provider == "" {
			entry.Provider = provider
		}
		return key, &entry
	}
	return "", nil
}

func upsertEntry(s Sessions, entry Entry) {
	provider := normalizeProvider(entry.Provider)
	entry.Provider = provider
	if oldKey, _ := findByNameWithKey(s, provider, entry.Name); oldKey != "" && oldKey != sessionKey(provider, entry.Name) {
		delete(s, oldKey)
	}
	s[sessionKey(provider, entry.Name)] = entry
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
	return FindByNameForProvider(s, defaultProvider, name)
}

func FindByNameForProvider(s Sessions, provider, name string) *Entry {
	_, entry := findByNameWithKey(s, provider, name)
	return entry
}

func FindByID(s Sessions, id string) *Entry {
	return FindByIDForProvider(s, defaultProvider, id)
}

func FindByIDForProvider(s Sessions, provider, id string) *Entry {
	provider = normalizeProvider(provider)
	for _, e := range s {
		if e.ID != id {
			continue
		}
		if effectiveProvider(e) != provider {
			continue
		}
		entry := e
		if entry.Provider == "" {
			entry.Provider = provider
		}
		return &entry
	}
	return nil
}

func Remove(s Sessions, name string) Sessions {
	return RemoveForProvider(s, defaultProvider, name)
}

func RemoveForProvider(s Sessions, provider, name string) Sessions {
	if key, _ := findByNameWithKey(s, provider, name); key != "" {
		delete(s, key)
	}
	return s
}

func filterByName(s Sessions, substring string) Sessions {
	substring = strings.ToLower(substring)
	out := Sessions{}
	for k, e := range s {
		if strings.Contains(strings.ToLower(e.Name), substring) {
			out[k] = e
		}
	}
	return out
}

// FindByIDAny finds an entry by ID across all providers.
func FindByIDAny(s Sessions, id string) *Entry {
	for _, e := range s {
		if e.ID == id {
			entry := e
			if entry.Provider == "" {
				entry.Provider = defaultProvider
			}
			return &entry
		}
	}
	return nil
}
