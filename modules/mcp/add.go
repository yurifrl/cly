package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// AddMCPOptions contains options for adding an MCP
type AddMCPOptions struct {
	Name        string
	Type        string // "stdio", "http", "sse"
	Command     string // For stdio
	Args        []string
	URL         string // For http/sse
	Env         map[string]string
	Headers     map[string]string
	Tags        []string
	Description string
	TargetFile  string
}

// AddMCP adds a new MCP to the catalog with full options support
func (c *Catalog) AddMCP(opts AddMCPOptions) error {
	// Validate required fields
	if opts.Name == "" {
		return fmt.Errorf("missing required field: name")
	}

	// Default type to stdio
	if opts.Type == "" {
		opts.Type = "stdio"
	}

	// Validate based on type
	switch opts.Type {
	case "stdio":
		if opts.Command == "" {
			return fmt.Errorf("stdio type requires command")
		}
	case "http", "sse":
		if opts.URL == "" {
			return fmt.Errorf("%s type requires URL", opts.Type)
		}
	default:
		return fmt.Errorf("invalid type: %s (use stdio, http, or sse)", opts.Type)
	}

	// Check if MCP already exists
	if _, exists := c.mcps[opts.Name]; exists {
		// Prompt for overwrite
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("MCP '%s' already exists. Overwrite? (y/n): ", opts.Name)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			return fmt.Errorf("cancelled - MCP not overwritten")
		}
	}

	// Convert env to interface{} map for serialization
	env := make(map[string]interface{})
	for k, v := range opts.Env {
		env[k] = v
	}

	// Create MCP
	mcp := MCP{
		Name:        opts.Name,
		Type:        opts.Type,
		Command:     opts.Command,
		Args:        opts.Args,
		URL:         opts.URL,
		Env:         env,
		Headers:     opts.Headers,
		Tags:        opts.Tags,
		Description: opts.Description,
	}

	// Determine target file path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	libDir := filepath.Join(homeDir, ".config", "mcpcli", "mcps")

	// If no file specified, use custom.yaml
	if opts.TargetFile == "" {
		opts.TargetFile = "custom.yaml"
	}

	// If relative path, join with source directory
	var filePath string
	if filepath.IsAbs(opts.TargetFile) {
		filePath = opts.TargetFile
	} else {
		filePath = filepath.Join(libDir, opts.TargetFile)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Read existing file if it exists
	existingMCPs := make(map[string]MCP)
	if data, err := os.ReadFile(filePath); err == nil {
		// Parse existing file
		parsedMCPs, parseErr := ParseMCPFile(filePath, data)
		if parseErr == nil {
			existingMCPs = parsedMCPs
		}
	}

	// Add/update MCP
	existingMCPs[opts.Name] = mcp

	// Determine format from file extension
	format := detectFormat(filePath)

	// Write file
	var data []byte
	if format == "yaml" {
		data, err = yaml.Marshal(existingMCPs)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML: %w", err)
		}
		// Add comment
		yamlWithComments := []byte("# MCP Sources - Custom MCPs\n\n")
		data = append(yamlWithComments, data...)
	} else {
		data, err = json.MarshalIndent(existingMCPs, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Add to catalog in memory
	c.mcps[opts.Name] = mcp

	fmt.Printf("✓ Added MCP '%s' to %s\n", opts.Name, filepath.Base(filePath))
	return nil
}
