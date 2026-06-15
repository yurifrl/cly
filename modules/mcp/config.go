package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// clyConfigPath returns the path to the shared cly config file (~/.config/cly/config.yaml).
func clyConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".config", "cly", "config.yaml"), nil
}

// LoadGlobalConfig loads the MCP configuration from the shared cly config file
// (~/.config/cly/config.yaml) under the `modules.mcp` key.
func LoadGlobalConfig() (*GlobalConfig, error) {
	path, err := clyConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		// No cly config yet: use defaults.
		return createDefaultGlobalConfig(), nil
	}

	var root map[string]interface{}
	if err := yaml.Unmarshal(data, &root); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: cly config is malformed; using default MCP config.\n")
		return createDefaultGlobalConfig(), nil
	}

	modules, _ := root["modules"].(map[string]interface{})
	mcpRaw, ok := modules["mcp"]
	if !ok || mcpRaw == nil {
		return createDefaultGlobalConfig(), nil
	}

	// Re-marshal the modules.mcp subtree and decode into GlobalConfig.
	sub, err := yaml.Marshal(mcpRaw)
	if err != nil {
		return createDefaultGlobalConfig(), nil
	}

	cfg := createDefaultGlobalConfig()
	if err := yaml.Unmarshal(sub, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: modules.mcp config is malformed; using defaults.\n")
		return createDefaultGlobalConfig(), nil
	}

	if cfg.Presets == nil {
		cfg.Presets = make(map[string][]string)
	}
	if cfg.Projects == nil {
		cfg.Projects = make(map[string]ProjectConfig)
	}

	return cfg, nil
}

// LoadProjectConfig loads project-specific config from .mcpcli.{yaml,jsonc,json}
func LoadProjectConfig() (*ProjectConfig, error) {
	var configPath string
	var configData []byte
	for _, filename := range []string{".mcpcli.yaml", ".mcpcli.jsonc", ".mcpcli.json"} {
		if data, err := os.ReadFile(filename); err == nil {
			configPath = filename
			configData = data
			break
		}
	}

	if configPath == "" {
		return nil, nil
	}

	var cfg ProjectConfig
	if err := parseConfigFile(configPath, configData, &cfg); err != nil {
		return nil, fmt.Errorf("malformed project config (%s): %w", configPath, err)
	}

	if cfg.AI == "" || cfg.Scope == "" {
		return nil, fmt.Errorf("project config must specify 'ai' and 'scope' fields")
	}

	return &cfg, nil
}

// createDefaultGlobalConfig returns a default global configuration
func createDefaultGlobalConfig() *GlobalConfig {
	cfg := &GlobalConfig{
		Presets:  make(map[string][]string),
		Projects: make(map[string]ProjectConfig),
	}
	cfg.Defaults.AI = "agents"
	cfg.Defaults.Scope = "project"
	return cfg
}

// SaveGlobalConfig merges the MCP config into the shared cly config file
// (~/.config/cly/config.yaml) under the `modules.mcp` key, preserving all
// other sections. Note: YAML comments in the file are not preserved.
func SaveGlobalConfig(cfg *GlobalConfig) error {
	path, err := clyConfigPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	root := map[string]interface{}{}
	if data, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(data, &root); err != nil {
			return fmt.Errorf("refusing to overwrite malformed cly config (%s): %w", path, err)
		}
		if root == nil {
			root = map[string]interface{}{}
		}
	}

	modules, _ := root["modules"].(map[string]interface{})
	if modules == nil {
		modules = map[string]interface{}{}
		root["modules"] = modules
	}

	// Convert the typed config into a generic map so it nests cleanly.
	sub, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	var mcpMap map[string]interface{}
	if err := yaml.Unmarshal(sub, &mcpMap); err != nil {
		return err
	}
	modules["mcp"] = mcpMap

	out, err := yaml.Marshal(root)
	if err != nil {
		return err
	}

	return os.WriteFile(path, out, 0644)
}

// parseConfigFile parses config file in any supported format
func parseConfigFile(path string, data []byte, target interface{}) error {
	format := detectFormat(path)

	switch format {
	case "yaml":
		return yaml.Unmarshal(data, target)
	case "jsonc":
		cleaned := stripJSONComments(data)
		return json.Unmarshal(cleaned, target)
	default:
		return json.Unmarshal(data, target)
	}
}

// GetGlobalConfigPath returns the path to the shared cly config file that
// holds MCP settings under `modules.mcp`.
func GetGlobalConfigPath() (string, error) {
	return clyConfigPath()
}
