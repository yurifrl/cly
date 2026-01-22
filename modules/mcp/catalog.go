package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Catalog manages the MCP source catalog
type Catalog struct {
	mcps map[string]MCP
}

// DefaultSourcePaths returns the default MCP source paths
func DefaultSourcePaths() []string {
	homeDir, _ := os.UserHomeDir()
	if homeDir == "" {
		return nil
	}
	return []string{
		filepath.Join(homeDir, ".mcps.jsonc"),
		filepath.Join(homeDir, ".mcps.json"),
	}
}

// LoadCatalog loads all MCP definitions from configDir/mcps/ subdirectory
func LoadCatalog(configDir string) (*Catalog, error) {
	return LoadCatalogWithSources(configDir, nil)
}

// LoadCatalogWithSources loads MCP definitions from source paths + configDir/mcps/
// Source paths are loaded first, then configDir/mcps/ overrides
func LoadCatalogWithSources(configDir string, sourcePaths []string) (*Catalog, error) {
	cat := &Catalog{
		mcps: make(map[string]MCP),
	}

	// Merge default paths with configured paths
	allPaths := append(DefaultSourcePaths(), sourcePaths...)

	// Load from source paths first
	for _, path := range allPaths {
		if path == "" {
			continue
		}
		// Expand ~ to home dir
		if len(path) > 0 && path[0] == '~' {
			if homeDir, err := os.UserHomeDir(); err == nil {
				path = filepath.Join(homeDir, path[1:])
			}
		}
		if _, err := os.Stat(path); err == nil {
			if err := cat.loadFile(path); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", path, err)
			}
		}
	}

	// Resolve symlinks in config dir path
	resolvedDir, err := filepath.EvalSymlinks(configDir)
	if err != nil {
		resolvedDir = configDir
	}

	// List files in mcps/ subdirectory
	var files []string
	mcpsDir := filepath.Join(resolvedDir, "mcps")
	entries, err := os.ReadDir(mcpsDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			ext := filepath.Ext(name)
			if ext == ".json" || ext == ".jsonc" || ext == ".yaml" || ext == ".yml" {
				files = append(files, filepath.Join(mcpsDir, name))
			}
		}
	}

	// If no config dir files and no sources loaded, create defaults
	if len(files) == 0 && len(cat.mcps) == 0 {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create config directory: %w", err)
		}

		if err := cat.createDefaultSources(configDir); err != nil {
			return nil, fmt.Errorf("failed to create default sources: %w", err)
		}
		return cat, nil
	}

	// Sort files alphabetically for deterministic merging
	sort.Strings(files)

	// Load and merge all source files (these override source paths)
	for _, file := range files {
		if err := cat.loadFile(file); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", filepath.Base(file), err)
			continue
		}
	}

	return cat, nil
}

// loadFile reads a single source file (JSON, JSONC, or YAML) and merges MCPs into the catalog
func (c *Catalog) loadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	mcps, err := ParseMCPFile(path, data)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	// Merge into catalog (last loaded wins for duplicates)
	for name, mcp := range mcps {
		mcp.Name = name // Ensure name field matches key
		mcp.SourceFile = filepath.Base(path) // Track source file
		c.mcps[name] = mcp
	}

	return nil
}

// createDefaultSources creates mcps.json with example MCPs in the config directory
func (c *Catalog) createDefaultSources(configDir string) error {
	defaultMCPs := map[string]MCP{
		"filesystem": {
			Name:        "filesystem",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
			Tags:        []string{"filesystem", "core"},
			Description: "Access local filesystem within specified directory",
		},
		"brave-search": {
			Name:        "brave-search",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-brave-search"},
			Tags:        []string{"search", "web"},
			Description: "Search the web using Brave Search API",
		},
		"git": {
			Name:        "git",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-git"},
			Tags:        []string{"git", "vcs"},
			Description: "Git repository operations and history",
		},
	}

	// Write as mcps.json in config directory (not in subdirectory)
	mcpsPath := filepath.Join(configDir, "mcps.json")
	data, err := json.MarshalIndent(defaultMCPs, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(mcpsPath, data, 0644); err != nil {
		return err
	}

	// Load into catalog
	for name, mcp := range defaultMCPs {
		c.mcps[name] = mcp
	}

	return nil
}

// GetAll returns all MCPs in the catalog
func (c *Catalog) GetAll() []MCP {
	mcps := make([]MCP, 0, len(c.mcps))
	for _, mcp := range c.mcps {
		mcps = append(mcps, mcp)
	}

	// Sort by name for consistent ordering
	sort.Slice(mcps, func(i, j int) bool {
		return mcps[i].Name < mcps[j].Name
	})

	return mcps
}

// Get returns a specific MCP by name
func (c *Catalog) Get(name string) (MCP, bool) {
	mcp, ok := c.mcps[name]
	return mcp, ok
}

// Filter returns MCPs matching the search query and/or tags
// query: searches in name, description, and tags (case-insensitive)
// tags: all specified tags must exist in MCP (AND logic)
func (c *Catalog) Filter(query string, tags []string) []MCP {
	var results []MCP

	queryLower := ""
	if query != "" {
		queryLower = toLower(query)
	}

	for _, mcp := range c.mcps {
		// Check query match (if query provided)
		matchesQuery := query == ""
		if !matchesQuery {
			nameLower := toLower(mcp.Name)
			descLower := toLower(mcp.Description)

			if contains(nameLower, queryLower) || contains(descLower, queryLower) {
				matchesQuery = true
			}

			// Also search in tags
			if !matchesQuery {
				for _, tag := range mcp.Tags {
					if contains(toLower(tag), queryLower) {
						matchesQuery = true
						break
					}
				}
			}
		}

		if !matchesQuery {
			continue
		}

		// Check tag match (if tags provided)
		matchesTags := true
		for _, requiredTag := range tags {
			found := false
			for _, mcpTag := range mcp.Tags {
				if toLower(mcpTag) == toLower(requiredTag) {
					found = true
					break
				}
			}
			if !found {
				matchesTags = false
				break
			}
		}

		if matchesTags {
			results = append(results, mcp)
		}
	}

	// Sort by name
	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	return results
}

// Helper functions for case-insensitive string operations
func toLower(s string) string {
	// Simple ASCII lowercase (good enough for tags/names)
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}

func contains(s, substr string) bool {
	return indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
