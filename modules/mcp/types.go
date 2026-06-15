package mcp

// Context represents the detected AI tool and scope to target
type Context struct {
	AI    string
	Scope string
}

// ExtraParam defines a configurable extra field that can be injected into MCP entries at write-time.
// Users define available params in the cly config (~/.config/cly/config.yaml) under mcp.extraParams.
// Example: key=lifecycle, value=lazy → adds "lifecycle": "lazy" to the MCP JSON.
type ExtraParam struct {
	Key   string      `json:"key"             yaml:"key"`
	Value interface{} `json:"value"           yaml:"value"`
	Label string      `json:"label,omitempty" yaml:"label,omitempty"`
}

// UIConfig represents UI preferences that can be persisted
type UIConfig struct {
	HiddenSections []string `json:"hiddenSections,omitempty" yaml:"hiddenSections,omitempty"`
}

// GlobalConfig represents the MCP configuration stored under the `modules.mcp` key in ~/.config/cly/config.yaml
type GlobalConfig struct {
	Defaults struct {
		AI      string   `json:"ai"`
		Scope   string   `json:"scope"`
		Presets []string `json:"presets,omitempty"`
		Tags    []string `json:"tags,omitempty"`
	} `json:"defaults"`
	SourcePaths []string                  `json:"sourcePaths,omitempty" yaml:"sourcePaths,omitempty"`
	Presets     map[string][]string       `json:"presets,omitempty"`
	Projects    map[string]ProjectConfig  `json:"projects,omitempty"`
	UI          UIConfig                  `json:"ui,omitempty" yaml:"ui,omitempty"`
	ExtraParams []ExtraParam              `json:"extraParams,omitempty" yaml:"extraParams,omitempty"`
}

// ProjectConfig represents project-specific configuration
// Stored as .mcpcli.json in repository root
type ProjectConfig struct {
	AI      string    `json:"ai"`
	Scope   string    `json:"scope"`
	Presets []string  `json:"presets,omitempty"`
	Tags    []string  `json:"tags,omitempty"`
	UI      *UIConfig `json:"ui,omitempty" yaml:"ui,omitempty"`
}

// ToolConfig represents an AI tool's configuration file structure
// The MCPServers map contains all installed MCP server definitions
type ToolConfig struct {
	MCPServers map[string]MCP `json:"mcpServers"`
	// Additional fields from original config are preserved during read/write
	// but not exposed through this struct
}

// ListItemType represents the type of item in the list
type ListItemType int

const (
	ListItemMCP ListItemType = iota
	ListItemPreset
	ListItemTag
	ListItemSeparator
	ListItemValidation
)

// ListItem represents an item in the display list (MCP, preset, or tag)
type ListItem struct {
	Type            ListItemType
	Name            string
	MCP             *MCP    // For MCP items
	MCPNames        []string // For preset/tag items (which MCPs they contain)
	Expanded        bool     // For expandable items (presets)
	ValidationIssue *Issue   // For validation items
}
