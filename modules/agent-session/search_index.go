package agentsession

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type indexedSession struct {
	ID             string    `json:"id"`
	Provider       string    `json:"provider"`
	Path           string    `json:"path"`
	Name           string    `json:"name,omitempty"`
	Description    string    `json:"description,omitempty"`
	JsonlPath      string    `json:"jsonl_path"`
	JsonlMtime     time.Time `json:"jsonl_mtime"`
	SavedAt        time.Time `json:"saved_at,omitempty"`
	FirstUserMsg   string    `json:"first_user_msg,omitempty"`
	SearchableText string    `json:"searchable_text,omitempty"`
}

type searchIndex struct {
	Version  int                        `json:"version"`
	Updated  time.Time                  `json:"updated"`
	Sessions map[string]*indexedSession `json:"sessions"`
}

const (
	searchIndexVersion     = 1
	searchExcerptMaxBytes  = 5 * 1024
	searchFirstMsgMaxBytes = 400
)

func searchIndexPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "cly", "agent-session-index.json")
}

func loadSearchIndex() *searchIndex {
	idx := &searchIndex{Version: searchIndexVersion, Sessions: map[string]*indexedSession{}}
	data, err := os.ReadFile(searchIndexPath())
	if err != nil {
		return idx
	}
	if err := json.Unmarshal(data, idx); err != nil || idx.Version != searchIndexVersion {
		return &searchIndex{Version: searchIndexVersion, Sessions: map[string]*indexedSession{}}
	}
	if idx.Sessions == nil {
		idx.Sessions = map[string]*indexedSession{}
	}
	return idx
}

func saveSearchIndex(idx *searchIndex) error {
	idx.Updated = time.Now()
	if err := os.MkdirAll(filepath.Dir(searchIndexPath()), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(searchIndexPath(), data, 0o644)
}

type jsonlFile struct {
	Path     string
	Provider string
	Mtime    time.Time
}

func discoverJsonl() ([]jsonlFile, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	var out []jsonlFile

	piRoot := filepath.Join(home, ".pi", "agent", "sessions")
	if matches, _ := filepath.Glob(filepath.Join(piRoot, "*", "*.jsonl")); len(matches) > 0 {
		for _, p := range matches {
			st, err := os.Stat(p)
			if err != nil {
				continue
			}
			out = append(out, jsonlFile{Path: p, Provider: "pi", Mtime: st.ModTime()})
		}
	}

	claudeRoot := filepath.Join(home, ".claude", "projects")
	if matches, _ := filepath.Glob(filepath.Join(claudeRoot, "*", "*.jsonl")); len(matches) > 0 {
		for _, p := range matches {
			st, err := os.Stat(p)
			if err != nil {
				continue
			}
			out = append(out, jsonlFile{Path: p, Provider: "claude", Mtime: st.ModTime()})
		}
	}
	return out, nil
}

func rebuildSearchIndex(idx *searchIndex, catalog Sessions) error {
	files, err := discoverJsonl()
	if err != nil {
		return err
	}
	catalogByID := make(map[string]Entry, len(catalog))
	for _, e := range catalog {
		catalogByID[e.ID] = e
	}
	seen := make(map[string]bool, len(files))

	for _, f := range files {
		id := extractIDFromJsonlPath(f.Path)
		if id == "" {
			continue
		}
		seen[id] = true

		existing, ok := idx.Sessions[id]
		if ok && !existing.JsonlMtime.Before(f.Mtime) {
			if cat, ok := catalogByID[id]; ok {
				existing.Name = cat.Name
				existing.Description = cat.Description
				existing.Path = cat.Path
				if !cat.SavedAt.IsZero() {
					existing.SavedAt = cat.SavedAt
				}
			}
			continue
		}
		entry := &indexedSession{
			ID:         id,
			Provider:   f.Provider,
			JsonlPath:  f.Path,
			JsonlMtime: f.Mtime,
		}
		if cat, ok := catalogByID[id]; ok {
			entry.Name = cat.Name
			entry.Description = cat.Description
			entry.Path = cat.Path
			entry.SavedAt = cat.SavedAt
		}
		entry.FirstUserMsg, entry.SearchableText = extractExcerpt(f.Path)
		idx.Sessions[id] = entry
	}

	for id := range idx.Sessions {
		if !seen[id] {
			delete(idx.Sessions, id)
		}
	}
	return nil
}

func extractIDFromJsonlPath(p string) string {
	base := strings.TrimSuffix(filepath.Base(p), ".jsonl")
	if i := strings.LastIndex(base, "_"); i >= 0 {
		return base[i+1:]
	}
	return base
}

func extractExcerpt(path string) (firstUser, searchable string) {
	f, err := os.Open(path)
	if err != nil {
		return "", ""
	}
	defer f.Close()

	var blob strings.Builder
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 4*1024*1024)

	for scanner.Scan() {
		raw := scanner.Bytes()
		if len(raw) == 0 {
			continue
		}
		role, text := parseJsonlMessage(raw)
		if role == "" || text == "" {
			continue
		}
		if firstUser == "" && role == "user" {
			firstUser = truncateText(text, searchFirstMsgMaxBytes)
		}
		if blob.Len() < searchExcerptMaxBytes {
			blob.WriteString(text)
			blob.WriteString("\n")
		}
		if blob.Len() >= searchExcerptMaxBytes {
			break
		}
	}
	return firstUser, truncateText(blob.String(), searchExcerptMaxBytes)
}

func parseJsonlMessage(line []byte) (role, text string) {
	var v map[string]any
	if err := json.Unmarshal(line, &v); err != nil {
		return "", ""
	}
	r, _ := v["role"].(string)
	if r == "" {
		if msg, ok := v["message"].(map[string]any); ok {
			r, _ = msg["role"].(string)
			v = msg
		}
	}
	if r != "user" && r != "assistant" {
		return "", ""
	}
	switch c := v["content"].(type) {
	case string:
		return r, c
	case []any:
		var parts []string
		for _, it := range c {
			if m, ok := it.(map[string]any); ok {
				if t, _ := m["type"].(string); t == "text" {
					if s, _ := m["text"].(string); s != "" {
						parts = append(parts, s)
					}
				}
			}
		}
		return r, strings.Join(parts, " ")
	}
	if t, ok := v["text"].(string); ok {
		return r, t
	}
	return "", ""
}

func truncateText(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

func indexStats(idx *searchIndex) string {
	return fmt.Sprintf("%d sessions indexed (cache: %s)", len(idx.Sessions), idx.Updated.Format("2006-01-02 15:04"))
}
