package agents

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// DefaultIDEs is the fallback when agents.yaml has empty ides list.
var DefaultIDEs = []string{"claude", "opencode"}

// ConfigFileName is the expected config file name.
const ConfigFileName = "agents.yaml"

// Config holds the parsed agents.yaml.
type Config struct {
	IDEs []string `yaml:"ides"`
}

// IDEDef describes how to sync into a specific IDE's config directory.
type IDEDef struct {
	Name              string
	LocalDir          string            // e.g. ".claude"
	GlobalDir         string            // e.g. "~/.claude"
	DirRenames        map[string]string // e.g. "commands"->"command"
	AgentsMDTarget    string            // target filename for AGENTS.md
	StripAllowedTools bool              // strip allowed-tools from SKILL.md
	SpecialFiles      map[string]string // source filename -> target filename
}

// SocketPath is where the daemon listens.
const SocketPath = "/tmp/cly-agents.sock"

// Subdirectories to sync.
var Subdirs = []string{"commands", "agents", "skills"}

var ideDefs = map[string]*IDEDef{
	"claude": {
		Name:              "claude",
		LocalDir:          ".claude",
		GlobalDir:         filepath.Join(homeDir(), ".claude"),
		DirRenames:        map[string]string{},
		AgentsMDTarget:    "CLAUDE.md",
		StripAllowedTools: false,
		SpecialFiles: map[string]string{
			"claude.json": "settings.json",
		},
	},
	"opencode": {
		Name:     "opencode",
		LocalDir: ".opencode",
		GlobalDir: filepath.Join(homeDir(), ".config", "opencode"),
		DirRenames: map[string]string{
			"commands": "command",
			"agents":   "agent",
			"skills":   "skill",
		},
		AgentsMDTarget:    "AGENTS.md",
		StripAllowedTools: true,
		SpecialFiles: map[string]string{
			"opencode.json": "opencode.json",
		},
	},
	"crush": {
		Name:              "crush",
		LocalDir:          ".crush",
		GlobalDir:         filepath.Join(homeDir(), ".crush"),
		DirRenames:        map[string]string{},
		AgentsMDTarget:    "AGENTS.md",
		StripAllowedTools: false,
		SpecialFiles:      map[string]string{},
	},
}

// ParseConfig reads and parses agents.yaml. Returns nil if file doesn't exist (no config = no sync).
func ParseConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if len(cfg.IDEs) == 0 {
		cfg.IDEs = DefaultIDEs
	}

	return &cfg, nil
}

// GetIDEDef returns the IDE definition for the given name, or nil.
func GetIDEDef(name string) *IDEDef {
	return ideDefs[name]
}

// ResolveSourceDirs returns the .agents source directories to check.
func ResolveSourceDirs(globalOnly bool) []string {
	if globalOnly {
		return globalSourceDirs()
	}
	// Local first, then global
	dirs := []string{".agents"}
	dirs = append(dirs, globalSourceDirs()...)
	return dirs
}

func globalSourceDirs() []string {
	home := homeDir()
	return []string{
		filepath.Join(home, ".agents"),
	}
}

// FindConfigFile looks for agents.yaml in source dirs.
func FindConfigFile(sourceDirs []string) string {
	for _, dir := range sourceDirs {
		p := filepath.Join(dir, ConfigFileName)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func homeDir() string {
	h, _ := os.UserHomeDir()
	return h
}
