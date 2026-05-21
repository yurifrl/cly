// Package ai is core LLM infrastructure. It owns no module-specific
// concerns: it just answers "given the global ai config, plus an optional
// per-call override block from the caller, what client should I build?"
//
// Architecture:
//
//   pkg/ai          ←  reads top-level `ai:` config block. Knows about
//                       providers, models, api_key resolution. Period.
//
//   modules/foo     →  owns its own config under `modules.foo.*`. If
//                       module foo wants to override AI settings, it puts
//                       them under `modules.foo.ai` and passes that map
//                       to ai.NewClientWith / ai.LoadConfigWith.
//
// AI is therefore a sibling of App and Theme in the config struct, NOT a
// pseudo-module. Modules depend on ai; ai never depends on a module.
//
// Config layout:
//
//   ai:
//     provider: anthropic
//     providers:
//       anthropic:
//         model: claude-sonnet-4-20250514
//         api_key: $ANTHROPIC_API_KEY    # literal or $ENV / ${ENV}
//       openai:
//         model: gpt-4o-mini
//         api_key: $OPENAI_API_KEY
//
//   modules:
//     commit:
//       ai:                              # commit-local override
//         provider: openai
//         model: gpt-4o-mini
//     agent_session:
//       search:
//         ai:
//           provider: anthropic
package ai

import (
	"context"
	"os"
	"strings"

	pkgconfig "github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/llm"
)

// Library defaults — used when nothing is configured.
const (
	defaultProvider  = "anthropic"
	defaultModel     = "claude-sonnet-4-20250514"
	defaultAPIKeyEnv = "ANTHROPIC_API_KEY"
)

// providerEnv maps each known provider to its conventional env var name so
// `api_key: $ANTHROPIC_API_KEY` style references just work without extra
// configuration.
var providerEnv = map[string]string{
	"anthropic": "ANTHROPIC_API_KEY",
	"openai":    "OPENAI_API_KEY",
}

// Resolved is the effective AI config after merging global defaults with
// an optional per-caller override block.
type Resolved struct {
	Provider  string
	Model     string
	APIKey    string // literal key (resolved from $ENV expansion)
	APIKeyEnv string // env var name (set when api_key is empty)
	Enabled   bool
}

// LoadConfig returns the effective AI config from the top-level `ai:`
// block alone — no module overrides. Call sites that don't need overrides
// (or are themselves the default) use this.
func LoadConfig() *Resolved {
	return resolve(pkgconfig.Get().AI, nil)
}

// LoadConfigWith merges an optional per-caller override map (e.g. the
// contents of `modules.<name>.ai`) on top of the global `ai:` config.
// Pass nil for `override` to get the same behavior as LoadConfig.
//
// Modules invoke this with the map they themselves read from their own
// config block. AI does NOT walk into module configs; the module owns
// that responsibility (separation of concerns).
func LoadConfigWith(override map[string]interface{}) *Resolved {
	return resolve(pkgconfig.Get().AI, override)
}

func resolve(global, override map[string]interface{}) *Resolved {
	r := &Resolved{
		Provider:  defaultProvider,
		Model:     defaultModel,
		APIKey:    "",
		APIKeyEnv: defaultAPIKeyEnv,
		Enabled:   true,
	}

	// Layer 1: global `ai:` block.
	if global != nil {
		if v, ok := global["provider"].(string); ok && v != "" {
			r.Provider = v
		}
		if pmap, ok := descendMap(global, "providers", r.Provider); ok {
			applyProviderBlock(pmap, r)
		}
	}
	if r.APIKey == "" && r.APIKeyEnv == defaultAPIKeyEnv {
		if env, ok := providerEnv[r.Provider]; ok {
			r.APIKeyEnv = env
		}
	}

	// Layer 2: caller-supplied override block. The override may switch
	// the active provider; if so, re-base from that provider's defaults
	// before applying the override's own model/api_key fields.
	if override != nil {
		if v, ok := override["enabled"].(bool); ok {
			r.Enabled = v
		}
		if v, ok := override["provider"].(string); ok && v != "" && v != r.Provider {
			r.Provider = v
			r.APIKey = ""
			r.APIKeyEnv = providerEnv[v]
			if r.APIKeyEnv == "" {
				r.APIKeyEnv = defaultAPIKeyEnv
			}
			if global != nil {
				if pmap, ok := descendMap(global, "providers", v); ok {
					applyProviderBlock(pmap, r)
				}
			}
		}
		applyOverrideBlock(override, r)
	}

	if !r.Enabled {
		return nil
	}
	return r
}

func applyProviderBlock(p map[string]interface{}, r *Resolved) {
	if v, ok := p["model"].(string); ok && v != "" {
		r.Model = v
	}
	if v, ok := p["api_key"].(string); ok && v != "" {
		setKeyOrEnv(v, r)
	}
	if v, ok := p["api_key_env"].(string); ok && v != "" {
		r.APIKey = ""
		r.APIKeyEnv = v
	}
}

func applyOverrideBlock(o map[string]interface{}, r *Resolved) {
	if v, ok := o["model"].(string); ok && v != "" {
		r.Model = v
	}
	if v, ok := o["api_key"].(string); ok && v != "" {
		setKeyOrEnv(v, r)
	}
	if v, ok := o["api_key_env"].(string); ok && v != "" {
		r.APIKey = ""
		r.APIKeyEnv = v
	}
}

// setKeyOrEnv accepts either a literal key or "$VAR" / "${VAR}" form and
// updates the resolved struct accordingly.
func setKeyOrEnv(s string, r *Resolved) {
	s = strings.TrimSpace(s)
	switch {
	case strings.HasPrefix(s, "${") && strings.HasSuffix(s, "}"):
		r.APIKey = ""
		r.APIKeyEnv = s[2 : len(s)-1]
	case strings.HasPrefix(s, "$") && len(s) > 1:
		r.APIKey = ""
		r.APIKeyEnv = s[1:]
	default:
		r.APIKey = s
		r.APIKeyEnv = ""
	}
}

func descendMap(m map[string]interface{}, path ...string) (map[string]interface{}, bool) {
	cur := m
	for _, p := range path {
		next, ok := cur[p].(map[string]interface{})
		if !ok {
			return nil, false
		}
		cur = next
	}
	return cur, true
}

// NewClient builds the LLM client from the global config alone.
func NewClient() (llm.Client, error) {
	return NewClientWith(nil)
}

// NewClientWith builds the LLM client using global config plus an
// optional per-caller override block. Returns (nil, nil) when AI is
// disabled in the override (allowing modules to opt out).
func NewClientWith(override map[string]interface{}) (llm.Client, error) {
	r := LoadConfigWith(override)
	if r == nil {
		return nil, nil
	}
	cfg := llm.Config{
		Provider:  llm.Provider(r.Provider),
		Model:     r.Model,
		APIKey:    r.APIKey,
		APIKeyEnv: r.APIKeyEnv,
	}
	return llm.NewClient(cfg)
}

// HasAPIKey reports whether the resolved config will yield a usable key
// without instantiating a client. Useful for `aiOn` UI gating.
func HasAPIKey(override map[string]interface{}) bool {
	r := LoadConfigWith(override)
	if r == nil {
		return false
	}
	if r.APIKey != "" {
		return true
	}
	return os.Getenv(r.APIKeyEnv) != ""
}

// Complete is the most common operation: build a client, send a single
// user message with a system prompt, return the full response.
func Complete(ctx context.Context, override map[string]interface{}, systemPrompt, userMessage string) (string, error) {
	client, err := NewClientWith(override)
	if err != nil {
		return "", err
	}
	if client == nil {
		return "", ErrDisabled
	}
	return client.Complete(ctx, systemPrompt, []llm.Message{{Role: llm.RoleUser, Content: userMessage}})
}

// ErrDisabled is returned when AI is disabled in the caller's override.
var ErrDisabled = errDisabled{}

type errDisabled struct{}

func (errDisabled) Error() string { return "ai is disabled in module config" }

// ModuleOverride is a small helper modules use to read their own `ai`
// sub-block from their own module config map. Saves every caller from
// re-implementing the type-assertion dance.
//
// Usage:
//
//   mod := pkgconfig.Get().Modules["commit"]
//   client, err := ai.NewClientWith(ai.ModuleOverride(mod))
func ModuleOverride(moduleConfig map[string]interface{}) map[string]interface{} {
	if moduleConfig == nil {
		return nil
	}
	if v, ok := moduleConfig["ai"].(map[string]interface{}); ok {
		return v
	}
	return nil
}

// LookupModuleOverride is the sugar for the most common case: "give me
// the AI override block at `modules.<modulePath>.ai`". Accepts a dotted
// path so nested module trees work too:
//
//   ai.LookupModuleOverride("commit")                   -> modules.commit.ai
//   ai.LookupModuleOverride("agent_session.search")     -> modules.agent_session.search.ai
//
// Returns nil when any segment is missing or `ai` is absent. Combined
// with NewClientWith / HasAPIKey this lets callers skip the manual map
// lookup entirely.
func LookupModuleOverride(modulePath string) map[string]interface{} {
	parts := strings.Split(modulePath, ".")
	if len(parts) == 0 || parts[0] == "" {
		return nil
	}
	cfg := pkgconfig.Get()
	cur, ok := cfg.Modules[parts[0]]
	if !ok {
		return nil
	}
	for _, p := range parts[1:] {
		next, ok := cur[p].(map[string]interface{})
		if !ok {
			return nil
		}
		cur = next
	}
	return ModuleOverride(cur)
}

// NewClientFor is sugar for the most common shape:
//   override := ai.LookupModuleOverride("commit")
//   client, err := ai.NewClientWith(override)
// collapsed into a single call.
func NewClientFor(modulePath string) (llm.Client, error) {
	return NewClientWith(LookupModuleOverride(modulePath))
}

// HasAPIKeyFor mirrors NewClientFor for the boolean check.
func HasAPIKeyFor(modulePath string) bool {
	return HasAPIKey(LookupModuleOverride(modulePath))
}

// CompleteFor is sugar for one-shot module-scoped LLM calls.
func CompleteFor(ctx context.Context, modulePath, systemPrompt, userMessage string) (string, error) {
	return Complete(ctx, LookupModuleOverride(modulePath), systemPrompt, userMessage)
}
