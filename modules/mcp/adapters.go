package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Adapter interface for AI tool config management
type Adapter interface {
	GetConfigPath(scope string) (string, error)
	ReadConfig(scope string) (*ToolConfig, error)
	WriteConfig(scope string, mcps []MCP) error
	IsInstalled() bool
}

// GetAdapter returns the appropriate adapter for the AI tool
func GetAdapter(ai string) (Adapter, error) {
	switch ai {
	case "claude":
		return &ClaudeAdapter{}, nil
	case "cursor":
		return &CursorAdapter{}, nil
	case "desktop":
		return &DesktopAdapter{}, nil
	case "pi":
		return &PiAdapter{}, nil
	default:
		return nil, fmt.Errorf("unsupported AI tool: %s (valid: claude, cursor, desktop, pi)", ai)
	}
}

// ClaudeAdapter implements the Adapter interface for Claude Code
type ClaudeAdapter struct{}

// applyExtraFields merges ExtraFields from an MCP into a raw map (used by all adapters).
// ExtraFields keys are written at the root level of the MCP JSON entry.
func applyExtraFields(m map[string]interface{}, mcp MCP) {
	for k, v := range mcp.ExtraFields {
		m[k] = v
	}
}

// claudeMCP wraps MCP with custom JSON marshaling for Claude Code
type claudeMCP struct {
	MCP
}

func (c claudeMCP) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	if c.Type != "" {
		m["type"] = c.Type
	}
	if c.Command != "" {
		m["command"] = c.Command
	}
	if len(c.Args) > 0 {
		m["args"] = c.Args
	}
	if c.URL != "" {
		m["url"] = c.URL
	}
	if len(c.Env) > 0 {
		m["env"] = c.Env
	}
	if len(c.Headers) > 0 {
		m["headers"] = c.Headers
	}
	applyExtraFields(m, c.MCP)
	return json.Marshal(m)
}

func (c *claudeMCP) UnmarshalJSON(data []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if typeVal, ok := raw["type"].(string); ok {
		c.Type = typeVal
	} else if transportVal, ok := raw["transport"].(string); ok {
		c.Type = transportVal
	}

	if c.Type == "" {
		if _, hasCommand := raw["command"]; hasCommand {
			c.Type = "stdio"
		} else if _, hasURL := raw["url"]; hasURL {
			c.Type = "http"
		}
	}

	if cmd, ok := raw["command"].(string); ok {
		c.Command = cmd
	}
	if args, ok := raw["args"].([]interface{}); ok {
		c.Args = make([]string, len(args))
		for i, arg := range args {
			if s, ok := arg.(string); ok {
				c.Args[i] = s
			}
		}
	}
	if url, ok := raw["url"].(string); ok {
		c.URL = url
	}
	if env, ok := raw["env"].(map[string]interface{}); ok {
		c.Env = env
	}
	if headers, ok := raw["headers"].(map[string]interface{}); ok {
		c.Headers = make(map[string]string)
		for k, v := range headers {
			if s, ok := v.(string); ok {
				c.Headers[k] = s
			}
		}
	}
	return nil
}

func (a *ClaudeAdapter) GetConfigPath(scope string) (string, error) {
	switch scope {
	case "user", "local":
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, ".claude.json"), nil
	case "project":
		return ".mcp.json", nil
	default:
		return "", fmt.Errorf("unsupported scope: %s (valid: user, project, local)", scope)
	}
}

func (a *ClaudeAdapter) ReadConfig(scope string) (*ToolConfig, error) {
	path, err := a.GetConfigPath(scope)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &ToolConfig{MCPServers: make(map[string]MCP)}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("malformed JSON: %w", err)
	}

	toolCfg := &ToolConfig{MCPServers: make(map[string]MCP)}
	var mcpServers map[string]interface{}

	if scope == "local" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
		if projects, ok := raw["projects"].(map[string]interface{}); ok {
			if project, ok := projects[cwd].(map[string]interface{}); ok {
				mcpServers, _ = project["mcpServers"].(map[string]interface{})
			}
		}
	} else {
		mcpServers, _ = raw["mcpServers"].(map[string]interface{})
	}

	if mcpServers != nil {
		for name, serverData := range mcpServers {
			serverJSON, _ := json.Marshal(serverData)
			var wrapped claudeMCP
			if err := json.Unmarshal(serverJSON, &wrapped); err == nil {
				wrapped.Name = name
				toolCfg.MCPServers[name] = wrapped.MCP
			}
		}
	}

	return toolCfg, nil
}

func (a *ClaudeAdapter) WriteConfig(scope string, mcps []MCP) error {
	path, err := a.GetConfigPath(scope)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	var config map[string]interface{}
	if data, err := os.ReadFile(path); err == nil {
		json.Unmarshal(data, &config)
	}
	if config == nil {
		config = make(map[string]interface{})
	}

	mcpServers := make(map[string]claudeMCP)
	for _, mcp := range mcps {
		mcpServers[mcp.Name] = claudeMCP{MCP: mcp}
	}

	if scope == "local" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		projects, ok := config["projects"].(map[string]interface{})
		if !ok {
			projects = make(map[string]interface{})
			config["projects"] = projects
		}
		project, ok := projects[cwd].(map[string]interface{})
		if !ok {
			project = make(map[string]interface{})
			projects[cwd] = project
		}
		project["mcpServers"] = mcpServers
	} else {
		config["mcpServers"] = mcpServers
	}

	tempPath := path + ".tmp"
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return err
	}

	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath)
		return err
	}

	return nil
}

func (a *ClaudeAdapter) IsInstalled() bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(homeDir, ".claude"))
	return err == nil
}

// PiAdapter implements the Adapter interface for Pi
type PiAdapter struct{}

type piMCP struct {
	MCP
}

func (p piMCP) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	if p.Type != "" {
		m["type"] = p.Type
	}
	if p.Command != "" {
		m["command"] = p.Command
	}
	if len(p.Args) > 0 {
		m["args"] = p.Args
	}
	if p.URL != "" {
		m["url"] = p.URL
	}
	if len(p.Env) > 0 {
		m["env"] = p.Env
	}
	if len(p.Headers) > 0 {
		m["headers"] = p.Headers
	}
	applyExtraFields(m, p.MCP)
	return json.Marshal(m)
}

func (p *piMCP) UnmarshalJSON(data []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if typeVal, ok := raw["type"].(string); ok {
		p.Type = typeVal
	} else if transportVal, ok := raw["transport"].(string); ok {
		p.Type = transportVal
	}

	if p.Type == "" {
		if _, hasCommand := raw["command"]; hasCommand {
			p.Type = "stdio"
		} else if _, hasURL := raw["url"]; hasURL {
			p.Type = "http"
		}
	}

	if cmd, ok := raw["command"].(string); ok {
		p.Command = cmd
	}
	if args, ok := raw["args"].([]interface{}); ok {
		p.Args = make([]string, len(args))
		for i, arg := range args {
			if s, ok := arg.(string); ok {
				p.Args[i] = s
			}
		}
	}
	if url, ok := raw["url"].(string); ok {
		p.URL = url
	}
	if env, ok := raw["env"].(map[string]interface{}); ok {
		p.Env = env
	}
	if headers, ok := raw["headers"].(map[string]interface{}); ok {
		p.Headers = make(map[string]string)
		for k, v := range headers {
			if s, ok := v.(string); ok {
				p.Headers[k] = s
			}
		}
	}
	return nil
}

func (a *PiAdapter) GetConfigPath(scope string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch scope {
	case "user":
		return filepath.Join(homeDir, ".pi", "agent", "mcp.json"), nil
	case "project":
		return ".pi/mcp.json", nil
	default:
		return "", fmt.Errorf("unsupported scope for Pi: %s (only user and project)", scope)
	}
}

func (a *PiAdapter) ReadConfig(scope string) (*ToolConfig, error) {
	path, err := a.GetConfigPath(scope)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &ToolConfig{MCPServers: make(map[string]MCP)}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("malformed JSON: %w", err)
	}

	toolCfg := &ToolConfig{MCPServers: make(map[string]MCP)}

	if mcpServers, ok := raw["mcpServers"].(map[string]interface{}); ok {
		for name, serverData := range mcpServers {
			serverJSON, _ := json.Marshal(serverData)
			var wrapped piMCP
			if err := json.Unmarshal(serverJSON, &wrapped); err == nil {
				wrapped.Name = name
				toolCfg.MCPServers[name] = wrapped.MCP
			}
		}
	}

	return toolCfg, nil
}

func (a *PiAdapter) WriteConfig(scope string, mcps []MCP) error {
	path, err := a.GetConfigPath(scope)
	if err != nil {
		return err
	}

	var config map[string]interface{}
	if data, err := os.ReadFile(path); err == nil {
		json.Unmarshal(data, &config)
	}
	if config == nil {
		config = make(map[string]interface{})
	}

	mcpServers := make(map[string]piMCP)
	for _, mcp := range mcps {
		mcpServers[mcp.Name] = piMCP{MCP: mcp}
	}

	config["mcpServers"] = mcpServers

	tempPath := path + ".tmp"
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return err
	}

	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath)
		return err
	}

	return nil
}

func (a *PiAdapter) IsInstalled() bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(homeDir, ".pi"))
	return err == nil
}

// CursorAdapter implements the Adapter interface for Cursor IDE
type CursorAdapter struct{}

type cursorMCP struct {
	MCP
}

func (c cursorMCP) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	if c.Command != "" {
		m["command"] = c.Command
	}
	if len(c.Args) > 0 {
		m["args"] = c.Args
	}
	if c.URL != "" {
		m["url"] = c.URL
	}
	if len(c.Env) > 0 {
		m["env"] = c.Env
	}
	if len(c.Headers) > 0 {
		m["headers"] = c.Headers
	}
	applyExtraFields(m, c.MCP)
	return json.Marshal(m)
}

func (c *cursorMCP) UnmarshalJSON(data []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if typeVal, ok := raw["type"].(string); ok {
		c.Type = typeVal
	} else if transportVal, ok := raw["transport"].(string); ok {
		c.Type = transportVal
	}

	if c.Type == "" {
		if _, hasCommand := raw["command"]; hasCommand {
			c.Type = "stdio"
		} else if _, hasURL := raw["url"]; hasURL {
			c.Type = "http"
		}
	}

	if cmd, ok := raw["command"].(string); ok {
		c.Command = cmd
	}
	if args, ok := raw["args"].([]interface{}); ok {
		c.Args = make([]string, len(args))
		for i, arg := range args {
			if s, ok := arg.(string); ok {
				c.Args[i] = s
			}
		}
	}
	if url, ok := raw["url"].(string); ok {
		c.URL = url
	}
	if env, ok := raw["env"].(map[string]interface{}); ok {
		c.Env = env
	}
	if headers, ok := raw["headers"].(map[string]interface{}); ok {
		c.Headers = make(map[string]string)
		for k, v := range headers {
			if s, ok := v.(string); ok {
				c.Headers[k] = s
			}
		}
	}
	return nil
}

func (a *CursorAdapter) GetConfigPath(scope string) (string, error) {
	switch scope {
	case "user":
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, ".cursor", "mcp.json"), nil
	case "project":
		return ".cursor/mcp.json", nil
	default:
		return "", fmt.Errorf("unsupported scope for Cursor: %s (only user and project)", scope)
	}
}

func (a *CursorAdapter) ReadConfig(scope string) (*ToolConfig, error) {
	path, err := a.GetConfigPath(scope)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &ToolConfig{MCPServers: make(map[string]MCP)}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("malformed JSON: %w", err)
	}

	toolCfg := &ToolConfig{MCPServers: make(map[string]MCP)}

	if mcpServers, ok := raw["mcpServers"].(map[string]interface{}); ok {
		for name, serverData := range mcpServers {
			serverJSON, _ := json.Marshal(serverData)
			var wrapped cursorMCP
			if err := json.Unmarshal(serverJSON, &wrapped); err == nil {
				wrapped.Name = name
				toolCfg.MCPServers[name] = wrapped.MCP
			}
		}
	}

	return toolCfg, nil
}

func (a *CursorAdapter) WriteConfig(scope string, mcps []MCP) error {
	path, err := a.GetConfigPath(scope)
	if err != nil {
		return err
	}

	var config map[string]interface{}
	if data, err := os.ReadFile(path); err == nil {
		json.Unmarshal(data, &config)
	}
	if config == nil {
		config = make(map[string]interface{})
	}

	mcpServers := make(map[string]cursorMCP)
	for _, mcp := range mcps {
		mcpServers[mcp.Name] = cursorMCP{MCP: mcp}
	}

	config["mcpServers"] = mcpServers

	tempPath := path + ".tmp"
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return err
	}

	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath)
		return err
	}

	return nil
}

func (a *CursorAdapter) IsInstalled() bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(homeDir, ".cursor"))
	return err == nil
}

// DesktopAdapter implements the Adapter interface for Claude Desktop
type DesktopAdapter struct{}

type desktopMCP struct {
	MCP
}

func (d desktopMCP) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	if d.Command != "" {
		m["command"] = d.Command
	}
	if len(d.Args) > 0 {
		m["args"] = d.Args
	}
	if d.URL != "" {
		m["url"] = d.URL
	}
	if len(d.Env) > 0 {
		m["env"] = d.Env
	}
	if len(d.Headers) > 0 {
		m["headers"] = d.Headers
	}
	applyExtraFields(m, d.MCP)
	return json.Marshal(m)
}

func (d *desktopMCP) UnmarshalJSON(data []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if typeVal, ok := raw["type"].(string); ok {
		d.Type = typeVal
	} else if transportVal, ok := raw["transport"].(string); ok {
		d.Type = transportVal
	}

	if d.Type == "" {
		if _, hasCommand := raw["command"]; hasCommand {
			d.Type = "stdio"
		} else if _, hasURL := raw["url"]; hasURL {
			d.Type = "http"
		}
	}

	if cmd, ok := raw["command"].(string); ok {
		d.Command = cmd
	}
	if args, ok := raw["args"].([]interface{}); ok {
		d.Args = make([]string, len(args))
		for i, arg := range args {
			if s, ok := arg.(string); ok {
				d.Args[i] = s
			}
		}
	}
	if url, ok := raw["url"].(string); ok {
		d.URL = url
	}
	if env, ok := raw["env"].(map[string]interface{}); ok {
		d.Env = env
	}
	if headers, ok := raw["headers"].(map[string]interface{}); ok {
		d.Headers = make(map[string]string)
		for k, v := range headers {
			if s, ok := v.(string); ok {
				d.Headers[k] = s
			}
		}
	}
	return nil
}

func (a *DesktopAdapter) GetConfigPath(scope string) (string, error) {
	if scope != "user" {
		return "", fmt.Errorf("Claude Desktop only supports 'user' scope")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", "Claude", "claude_desktop_config.json"), nil
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
		return filepath.Join(appData, "Claude", "claude_desktop_config.json"), nil
	case "linux":
		return filepath.Join(homeDir, ".config", "Claude", "claude_desktop_config.json"), nil
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func (a *DesktopAdapter) ReadConfig(scope string) (*ToolConfig, error) {
	path, err := a.GetConfigPath(scope)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &ToolConfig{MCPServers: make(map[string]MCP)}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("malformed JSON: %w", err)
	}

	toolCfg := &ToolConfig{MCPServers: make(map[string]MCP)}

	if mcpServers, ok := raw["mcpServers"].(map[string]interface{}); ok {
		for name, serverData := range mcpServers {
			serverJSON, _ := json.Marshal(serverData)
			var wrapped desktopMCP
			if err := json.Unmarshal(serverJSON, &wrapped); err == nil {
				wrapped.Name = name
				toolCfg.MCPServers[name] = wrapped.MCP
			}
		}
	}

	return toolCfg, nil
}

func (a *DesktopAdapter) WriteConfig(scope string, mcps []MCP) error {
	path, err := a.GetConfigPath(scope)
	if err != nil {
		return err
	}

	var config map[string]interface{}
	if data, err := os.ReadFile(path); err == nil {
		json.Unmarshal(data, &config)
	}
	if config == nil {
		config = make(map[string]interface{})
	}

	mcpServers := make(map[string]desktopMCP)
	for _, mcp := range mcps {
		mcpServers[mcp.Name] = desktopMCP{MCP: mcp}
	}

	config["mcpServers"] = mcpServers

	tempPath := path + ".tmp"
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return err
	}

	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath)
		return err
	}

	return nil
}

func (a *DesktopAdapter) IsInstalled() bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	var checkPath string
	switch runtime.GOOS {
	case "darwin":
		checkPath = filepath.Join(homeDir, "Library", "Application Support", "Claude")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return false
		}
		checkPath = filepath.Join(appData, "Claude")
	case "linux":
		checkPath = filepath.Join(homeDir, ".config", "Claude")
	default:
		return false
	}

	_, err = os.Stat(checkPath)
	return err == nil
}
