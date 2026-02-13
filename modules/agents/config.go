package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// DefaultIDEs is the fallback when no ai.json exists.
var DefaultIDEs = []string{"claude", "opencode"}

// Config holds the parsed ai.json.
type Config struct {
	IDEs []string `json:"ides"`
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

// ParseConfig reads and parses ai.json. Returns defaults if file doesn't exist.
func ParseConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{IDEs: DefaultIDEs}, nil
		}
		return nil, err
	}

	// ai.json is JSONC (bare words, comments) — strip to valid JSON
	clean := stripJSONCComments(string(data))

	// Handle bare-word arrays like [claude, opencode] by quoting them
	clean = quoteBareWords(clean)

	var cfg Config
	if err := json.Unmarshal([]byte(clean), &cfg); err != nil {
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
		filepath.Join(home, ".config", "ai"),
		filepath.Join(home, ".agents"),
	}
}

// FindConfigFile looks for ai.json in source dirs.
func FindConfigFile(sourceDirs []string) string {
	for _, dir := range sourceDirs {
		p := filepath.Join(dir, "ai.json")
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

// quoteBareWords handles JSONC quirks where keys and array values are unquoted.
// Converts {ides: [claude, opencode]} to {"ides": ["claude", "opencode"]}.
func quoteBareWords(s string) string {
	var out []byte
	i := 0
	for i < len(s) {
		ch := s[i]
		if ch == '"' {
			// Pass string through
			out = append(out, ch)
			i++
			for i < len(s) {
				out = append(out, s[i])
				if s[i] == '\\' {
					i++
					if i < len(s) {
						out = append(out, s[i])
					}
				} else if s[i] == '"' {
					i++
					break
				}
				i++
			}
			continue
		}
		if isIdentStart(ch) {
			start := i
			for i < len(s) && isIdentChar(s[i]) {
				i++
			}
			word := s[start:i]
			if word == "true" || word == "false" || word == "null" {
				out = append(out, []byte(word)...)
			} else {
				out = append(out, '"')
				out = append(out, []byte(word)...)
				out = append(out, '"')
			}
			continue
		}
		out = append(out, ch)
		i++
	}
	return string(out)
}

func isIdentStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || c == '-'
}

func isIdentChar(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}
