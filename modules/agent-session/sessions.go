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
	DeletedAt   *time.Time        `json:"deleted_at,omitempty"`
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
	newKey := sessionKey(provider, entry.Name)
	// Dedupe by ID across all providers: if another entry has the same ID, remove it.
	if entry.ID != "" {
		for k, e := range s {
			if e.ID == entry.ID && k != newKey {
				delete(s, k)
			}
		}
	}
	if oldKey, _ := findByNameWithKey(s, provider, entry.Name); oldKey != "" && oldKey != newKey {
		delete(s, oldKey)
	}
	s[newKey] = entry
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
	return FindByNameForProvider(s, defaultProviderFallback, name)
}

func FindByNameForProvider(s Sessions, provider, name string) *Entry {
	_, entry := findByNameWithKey(s, provider, name)
	return entry
}

func FindByID(s Sessions, id string) *Entry {
	return FindByIDForProvider(s, defaultProviderFallback, id)
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
	return RemoveForProvider(s, defaultProviderFallback, name)
}

func RemoveForProvider(s Sessions, provider, name string) Sessions {
	if key, _ := findByNameWithKey(s, provider, name); key != "" {
		delete(s, key)
	}
	return s
}

// SoftDeleteForProvider marks a session as deleted without removing it.
func SoftDeleteForProvider(s Sessions, provider, name string) Sessions {
	if key, entry := findByNameWithKey(s, provider, name); key != "" {
		now := time.Now()
		entry.DeletedAt = &now
		s[key] = *entry
	}
	return s
}

// filterDeleted returns only non-deleted sessions.
func filterDeleted(s Sessions, includeDeleted bool) Sessions {
	if includeDeleted {
		return s
	}
	out := Sessions{}
	for k, e := range s {
		if e.DeletedAt == nil {
			out[k] = e
		}
	}
	return out
}

// filterOnlyDeleted returns only soft-deleted sessions.
func filterOnlyDeleted(s Sessions) Sessions {
	out := Sessions{}
	for k, e := range s {
		if e.DeletedAt != nil {
			out[k] = e
		}
	}
	return out
}

// CleanupDeleted permanently removes all soft-deleted sessions.
func CleanupDeleted(s Sessions) Sessions {
	for k, e := range s {
		if e.DeletedAt != nil {
			delete(s, k)
		}
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
				entry.Provider = defaultProviderFallback
			}
			return &entry
		}
	}
	return nil
}
