package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// RemoveMCP removes an MCP from the catalog
func (c *Catalog) RemoveMCP(name string) error {
	// Check if MCP exists
	mcp, exists := c.mcps[name]
	if !exists {
		// Suggest similar names
		suggestions := c.findSimilar(name)
		if len(suggestions) > 0 {
			return fmt.Errorf("MCP '%s' not found\nDid you mean: %s?", name, strings.Join(suggestions, ", "))
		}
		return fmt.Errorf("MCP '%s' not found in catalog", name)
	}

	// Determine which file contains this MCP
	sourceFile := mcp.SourceFile
	if sourceFile == "" {
		// If no source file tracked, search all files
		sourceFile = "custom.yaml" // Default fallback
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	libDir := filepath.Join(homeDir, ".config", "cly", "mcps")
	filePath := filepath.Join(libDir, sourceFile)

	// Read existing file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse file
	mcps, err := ParseMCPFile(filePath, data)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	// Remove MCP
	delete(mcps, name)

	// If file is now empty, delete it
	if len(mcps) == 0 {
		if err := os.Remove(filePath); err != nil {
			return fmt.Errorf("failed to delete empty file: %w", err)
		}
		delete(c.mcps, name)
		fmt.Printf("✓ Removed MCP '%s' and deleted empty file %s\n", name, sourceFile)
		return nil
	}

	// Write updated file atomically
	format := detectFormat(filePath)
	var newData []byte

	if format == "yaml" {
		newData, err = yaml.Marshal(mcps)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML: %w", err)
		}
		yamlWithComments := []byte("# MCP Sources\n\n")
		newData = append(yamlWithComments, newData...)
	} else {
		newData, err = json.MarshalIndent(mcps, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
	}

	// Atomic write
	tempPath := filePath + ".tmp"
	if err := os.WriteFile(tempPath, newData, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tempPath, filePath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to update file: %w", err)
	}

	// Remove from in-memory catalog
	delete(c.mcps, name)

	fmt.Printf("✓ Removed MCP '%s' from %s\n", name, sourceFile)
	return nil
}

// findSimilar finds MCPs with similar names (simple prefix matching)
func (c *Catalog) findSimilar(query string) []string {
	var suggestions []string
	queryLower := toLower(query)

	for name := range c.mcps {
		nameLower := toLower(name)
		// Check if query is prefix of name, or name is prefix of query
		if strings.HasPrefix(nameLower, queryLower) || strings.HasPrefix(queryLower, nameLower) {
			suggestions = append(suggestions, name)
			if len(suggestions) >= 3 {
				break
			}
		}
	}

	return suggestions
}
