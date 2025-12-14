package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadGlobalConfig loads the global configuration from ~/.config/mcpcli/config.{yaml,jsonc,json}
func LoadGlobalConfig() (*GlobalConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "mcpcli")

	// Try formats in precedence order: yaml > jsonc > json
	var configPath string
	var configData []byte
	for _, filename := range []string{"config.yaml", "config.jsonc", "config.json"} {
		path := filepath.Join(configDir, filename)
		if data, err := os.ReadFile(path); err == nil {
			configPath = path
			configData = data
			break
		}
	}

	// If no config found, create default
	if configPath == "" {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create config directory: %w", err)
		}

		defaultConfig := createDefaultGlobalConfig()
		configPath = filepath.Join(configDir, "config.yaml")
		if err := saveGlobalConfig(configPath, defaultConfig); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}

		return defaultConfig, nil
	}

	// Parse existing config
	var cfg GlobalConfig
	if err := parseConfigFile(configPath, configData, &cfg); err != nil {
		// Create fresh default on corrupted config
		defaultConfig := createDefaultGlobalConfig()
		defaultPath := filepath.Join(configDir, "config.yaml")
		if err := saveGlobalConfig(defaultPath, defaultConfig); err != nil {
			return nil, fmt.Errorf("failed to recover from corrupted config: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Warning: Config was corrupted. Created fresh config.\n")
		return defaultConfig, nil
	}

	return &cfg, nil
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
	cfg.Defaults.AI = "claude"
	cfg.Defaults.Scope = "user"
	return cfg
}

// SaveGlobalConfig writes global config to the default config path
func SaveGlobalConfig(cfg *GlobalConfig) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "mcpcli")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	return saveGlobalConfig(configPath, cfg)
}

// saveGlobalConfig writes global config to file with formatting
func saveGlobalConfig(path string, cfg *GlobalConfig) error {
	format := detectFormat(path)

	var data []byte
	var err error

	if format == "yaml" {
		data, err = yaml.Marshal(cfg)
		if err != nil {
			return err
		}
		yamlWithComments := []byte("# MCP Manager Global Configuration\n# Default AI tool and scope settings\n\n")
		data = append(yamlWithComments, data...)
	} else {
		data, err = json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return err
		}
	}

	return os.WriteFile(path, data, 0644)
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

// GetGlobalConfigPath returns the path to global config
func GetGlobalConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(homeDir, ".config", "mcpcli")

	for _, filename := range []string{"config.yaml", "config.jsonc", "config.json"} {
		path := filepath.Join(configDir, filename)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return filepath.Join(configDir, "config.yaml"), nil
}
