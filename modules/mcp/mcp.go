package mcp

// MCP represents a Model Context Protocol server definition
// It includes standard MCP server configuration plus custom metadata
type MCP struct {
	Name        string                 `json:"name,omitempty"`
	Type        string                 `json:"type,omitempty"`        // "stdio", "http", "sse"
	Command     string                 `json:"command,omitempty"`     // For stdio
	Args        []string               `json:"args,omitempty"`        // For stdio
	URL         string                 `json:"url,omitempty"`         // For http/sse
	Env         map[string]interface{} `json:"env,omitempty"`         // Environment variables
	Headers     map[string]string      `json:"headers,omitempty"`     // For http/sse WebSocket
	Tags        []string               `json:"tags,omitempty"`        // Custom metadata
	Description string                 `json:"description,omitempty"` // Custom metadata
	SourceFile  string                 `json:"-"`                     // Track which file MCP came from (not serialized)
	// ExtraFields holds additional key-value pairs injected at write-time from ExtraParams config.
	// These are NOT loaded from catalog YAML — they are set by the TUI before WriteConfig is called.
	ExtraFields map[string]interface{} `json:"-"`
}
