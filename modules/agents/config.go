package agents

import (
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// DefaultIDEs is the fallback when agents.yaml has empty ides list.
var DefaultIDEs = []string{"claude", "opencode"}

// ConfigFileName is the expected config file name.
const ConfigFileName = "agents.yaml"

// Config holds the parsed agents.yaml.
type Config struct {
	IDEs  []string `yaml:"ides"`
	Repos []string `yaml:"repos"`
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

const (
	pidFileName    = "agents.pid"
	logFileName    = "agents.log"
	statusFileName = "agents.status.yaml"
)

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
		Name:      "opencode",
		LocalDir:  ".opencode",
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
	if cfg.Repos == nil {
		cfg.Repos = []string{}
	}

	return &cfg, nil
}

// LoadGlobalConfig reads ~/.config/cly/agents.yaml and falls back to defaults when missing.
func LoadGlobalConfig() (*Config, error) {
	cfgPath := GlobalConfigPath()
	cfg, err := ParseConfig(cfgPath)
	if err != nil {
		return nil, err
	}
	if cfg != nil {
		return cfg, nil
	}

	// Compatibility fallback: migrate IDE settings from old ~/.agents/agents.yaml if present.
	legacyPath := filepath.Join(homeDir(), ".agents", ConfigFileName)
	legacy, err := ParseConfig(legacyPath)
	if err != nil {
		return nil, err
	}
	if legacy != nil {
		legacy.Repos = []string{}
		return legacy, nil
	}

	return &Config{
		IDEs:  append([]string{}, DefaultIDEs...),
		Repos: []string{},
	}, nil
}

// SaveGlobalConfig writes ~/.config/cly/agents.yaml.
func SaveGlobalConfig(cfg *Config) error {
	if cfg == nil {
		cfg = &Config{}
	}
	if len(cfg.IDEs) == 0 {
		cfg.IDEs = append([]string{}, DefaultIDEs...)
	}
	cfg.Repos = normalizeRepoList(cfg.Repos)

	if err := os.MkdirAll(GlobalConfigDir(), 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(GlobalConfigPath(), data, 0644)
}

// AddRepo appends a repo path if missing. Returns true when added.
func AddRepo(cfg *Config, repo string) bool {
	for _, existing := range cfg.Repos {
		if existing == repo {
			return false
		}
	}
	cfg.Repos = append(cfg.Repos, repo)
	cfg.Repos = normalizeRepoList(cfg.Repos)
	return true
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

// GlobalConfigDir returns ~/.config/cly.
func GlobalConfigDir() string {
	return filepath.Join(homeDir(), ".config", "cly")
}

// GlobalConfigPath returns ~/.config/cly/agents.yaml.
func GlobalConfigPath() string {
	return filepath.Join(GlobalConfigDir(), ConfigFileName)
}

// PidFilePath returns ~/.config/cly/agents.pid.
func PidFilePath() string {
	return filepath.Join(GlobalConfigDir(), pidFileName)
}

// LogFilePath returns ~/.config/cly/agents.log.
func LogFilePath() string {
	return filepath.Join(GlobalConfigDir(), logFileName)
}

// StatusFilePath returns ~/.config/cly/agents.status.yaml.
func StatusFilePath() string {
	return filepath.Join(GlobalConfigDir(), statusFileName)
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

func normalizeRepoList(repos []string) []string {
	seen := make(map[string]struct{}, len(repos))
	out := make([]string, 0, len(repos))
	for _, r := range repos {
		if r == "" {
			continue
		}
		if _, ok := seen[r]; ok {
			continue
		}
		seen[r] = struct{}{}
		out = append(out, r)
	}
	sort.Strings(out)
	return out
}
