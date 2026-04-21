package agentsession

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	pkgconfig "github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/session"
)

const defaultProviderFallback = "claude"

// defaultProviderFn is overridable in tests.
var defaultProviderFn = func() string {
	cfg := pkgconfig.Get()
	if cfg != nil {
		if module, ok := cfg.Modules["agent_session"]; ok {
			if v, ok := module["default_provider"].(string); ok {
				if v = strings.TrimSpace(strings.ToLower(v)); v != "" && v != "all" {
					return v
				}
			}
		}
	}
	return defaultProviderFallback
}

func defaultProvider() string { return defaultProviderFn() }

// detectProviderByID scans known session storage locations to find which
// provider owns the given session ID. Returns "" when no match is found.
func detectProviderByID(id string) string {
	if id == "" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	// pi: ~/.pi/agent/sessions/<project>/*<id>.jsonl
	if matches, _ := filepath.Glob(filepath.Join(home, ".pi", "agent", "sessions", "*", "*"+id+".jsonl")); len(matches) > 0 {
		return "pi"
	}
	// claude: ~/.claude/projects/<project>/<id>.jsonl
	if matches, _ := filepath.Glob(filepath.Join(home, ".claude", "projects", "*", id+".jsonl")); len(matches) > 0 {
		return "claude"
	}
	return ""
}

// Provider defines how to resume a session for a specific agent CLI.
type Provider struct {
	Name       string
	Command    string
	ResumeArgs []string // supports {id} placeholder
	YoloArgs   []string
}

var loadProvidersFn = loadProviders

func normalizeProvider(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return defaultProviderFallback
	}
	return name
}

func defaultProviders() map[string]Provider {
	return map[string]Provider{
		"claude": {
			Name:       "claude",
			Command:    "claude",
			ResumeArgs: []string{"-r", "{id}"},
			YoloArgs:   session.YoloArgs(),
		},
		"pi": {
			Name:       "pi",
			Command:    "pi",
			ResumeArgs: []string{"--session", "{id}"},
		},
	}
}

func loadProviders() map[string]Provider {
	providers := defaultProviders()

	cfg := pkgconfig.Get()
	module, ok := cfg.Modules["agent_session"]
	if !ok {
		return providers
	}

	rawProviders, ok := module["providers"].(map[string]interface{})
	if !ok {
		return providers
	}

	for key, raw := range rawProviders {
		providerName := normalizeProvider(key)
		rawMap, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}

		entry := providers[providerName]
		if entry.Name == "" {
			entry.Name = providerName
		}

		if command, ok := rawMap["command"].(string); ok && strings.TrimSpace(command) != "" {
			entry.Command = strings.TrimSpace(command)
		}
		if resumeArgs := toStringSlice(rawMap["resume_args"]); len(resumeArgs) > 0 {
			entry.ResumeArgs = resumeArgs
		}
		if yoloArgs := toStringSlice(rawMap["yolo_args"]); len(yoloArgs) > 0 {
			entry.YoloArgs = yoloArgs
		}

		providers[providerName] = entry
	}

	return providers
}

func providerByName(name string) (Provider, error) {
	providerName := normalizeProvider(name)
	providers := loadProvidersFn()
	provider, ok := providers[providerName]
	if !ok {
		return Provider{}, fmt.Errorf("unknown provider %q", providerName)
	}
	if provider.Command == "" {
		return Provider{}, fmt.Errorf("provider %q has no command configured", providerName)
	}
	if len(provider.ResumeArgs) == 0 {
		return Provider{}, fmt.Errorf("provider %q has no resume_args configured", providerName)
	}
	provider.Name = providerName
	return provider, nil
}

func availableProviders() []string {
	providers := loadProvidersFn()
	out := make([]string, 0, len(providers))
	for name := range providers {
		out = append(out, name)
	}
	return out
}

func providerSupportsYolo(provider Provider) bool {
	return len(provider.YoloArgs) > 0
}

func buildResumeArgs(provider Provider, id string, yolo bool) []string {
	args := make([]string, 0, len(provider.ResumeArgs)+len(provider.YoloArgs))
	if yolo && providerSupportsYolo(provider) {
		args = append(args, provider.YoloArgs...)
	}
	for _, arg := range provider.ResumeArgs {
		args = append(args, strings.ReplaceAll(arg, "{id}", id))
	}
	return args
}

func execProvider(entry *Entry, provider Provider, yolo bool) error {
	sess := &session.Session{Name: entry.Name}
	_ = sess.RenameZellijTab()

	binaryPath, err := exec.LookPath(provider.Command)
	if err != nil {
		return fmt.Errorf("%s not found in PATH", provider.Command)
	}

	args := buildResumeArgs(provider, entry.ID, yolo)
	execArgs := append([]string{provider.Command}, args...)
	return syscall.Exec(binaryPath, execArgs, syscall.Environ())
}

func toStringSlice(v any) []string {
	if v == nil {
		return nil
	}
	switch raw := v.(type) {
	case []string:
		return append([]string(nil), raw...)
	case []interface{}:
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
