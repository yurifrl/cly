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
//     providers:                          # list of named entries; first condition
//       - name: aihub                     # match (highest weight, ties by list
//         provider: openai                # order) wins; else `default: true`;
//         base_url: https://gw.example/v1 # else first entry
//         api_key: $AIHUB_API_KEY         # literal or $ENV / ${ENV}
//         model: aihub/claude-sonnet-5
//         weight: 10                      # optional, default 0
//         condition: 'user == "yuri" && dir =~ "~/Workdir/Yuri/*"'
//         default: true                   # fallback when no condition matches
//       - name: bedrock                   # Anthropic models via AWS Bedrock
//         provider: bedrock
//         model: us.anthropic.claude-sonnet-4-5-20250929-v1:0
//         # no api_key: auth uses AWS_BEARER_TOKEN_BEDROCK or AWS creds/profile
//
// Condition fields: user, host, arch, os, dir, env.NAME. Operators:
// ==, !=, =~ (glob), !~, &&, ||, !, parens. Bare field = truthy when set.
// Selection context and per-entry results are recorded in a Decision
// (LastDecision); with app.debug on, the pick and full table are logged
// to stderr once per process.
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
	"fmt"
	"os"
	"strings"
	"sync"

	pkgconfig "github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/llm"
)

// Library defaults — used when nothing is configured.
const (
	defaultProvider  = "anthropic"
	defaultModel     = "claude-sonnet-4-5-20250929"
	defaultAPIKeyEnv = "ANTHROPIC_API_KEY"
)

// providerEnv maps each known provider to its conventional env var name so
// `api_key: $ANTHROPIC_API_KEY` style references just work without extra
// configuration.
var providerEnv = map[string]string{
	"anthropic":  "ANTHROPIC_API_KEY",
	"openai":     "OPENAI_API_KEY",
	"openrouter": "OPENROUTER_API_KEY",
}

// Resolved is the effective AI config after merging global defaults with
// an optional per-caller override block.
type Resolved struct {
	Provider  string
	Model     string
	APIKey    string // literal key (resolved from $ENV expansion)
	APIKeyEnv string // env var name (set when api_key is empty)
	BaseURL   string // optional override for OpenAI-compatible endpoints
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

var (
	lastDecision *Decision
	lastSelErr   error
	loggedOnce   sync.Once
)

// LastDecision returns the most recent provider selection record, or nil
// if no selection has run yet this process.
func LastDecision() *Decision { return lastDecision }

// LastSelectionError returns the most recent resolution error, or nil.
func LastSelectionError() error { return lastSelErr }

// resolveE is resolve with an error return for paths that surface config
// problems (NewClientWith, cly ai status).
func resolveE(global, override map[string]interface{}) (*Resolved, error) {
	if global == nil {
		return &Resolved{
			Provider:  defaultProvider,
			Model:     defaultModel,
			APIKeyEnv: defaultAPIKeyEnv,
			Enabled:   true,
		}, nil
	}
	entries, err := parseProviders(global)
	if err != nil {
		lastSelErr = err
		return nil, err
	}
	if len(entries) == 0 {
		// ai: block present but no providers defined: library defaults.
		return &Resolved{
			Provider:  defaultProvider,
			Model:     defaultModel,
			APIKeyEnv: defaultAPIKeyEnv,
			Enabled:   true,
		}, nil
	}
	entry, decision := selectProvider(entries, buildContext())
	lastDecision = decision
	lastSelErr = nil
	logSelection(decision, entry)

	r := &Resolved{
		Provider:  entry.Provider,
		Model:     entry.Model,
		APIKey:    entry.APIKey,
		APIKeyEnv: entry.APIKeyEnv,
		BaseURL:   entry.BaseURL,
		Enabled:   true,
	}

	// Override layer: may switch the active provider type; if so, re-base
	// from the first entry of that type before applying the override's own
	// model/api_key fields.
	if override != nil {
		if v, ok := override["enabled"].(bool); ok {
			r.Enabled = v
		}
		if v, ok := override["provider"].(string); ok && v != "" && v != r.Provider {
			r.Provider = v
			r.APIKey = ""
			r.APIKeyEnv = providerEnv[v]
			r.BaseURL = ""
			for _, e := range entries {
				if e.Provider == v {
					r.Model = e.Model
					r.APIKey = e.APIKey
					r.APIKeyEnv = e.APIKeyEnv
					r.BaseURL = e.BaseURL
					break
				}
			}
		}
		applyOverrideBlock(override, r)
	}
	if !r.Enabled {
		return nil, nil
	}
	return r, nil
}

// resolve keeps the historical signature: nil on error or disabled.
// The error is retrievable via LastSelectionError.
func resolve(global, override map[string]interface{}) *Resolved {
	r, _ := resolveE(global, override)
	return r
}

// logSelection prints the selection one-liner plus the full decision table
// to stderr when app.debug is on. Logged once per process: the context and
// entries can't change between selections in a single run.
func logSelection(d *Decision, picked Entry) {
	if !pkgconfig.Get().App.Debug {
		return
	}
	loggedOnce.Do(func() {
		w := os.Stderr
		fmt.Fprintf(w, "ai: picked provider %q (%s, weight %d)\n", picked.Name, d.Reason, picked.Weight)
		fmt.Fprintf(w, "ai: context: user=%s host=%s arch=%s os=%s dir=%s\n",
			d.Context.User, d.Context.Host, d.Context.Arch, d.Context.OS, d.Context.Dir)
		for name, set := range d.EnvRefs {
			fmt.Fprintf(w, "ai: context: env.%s=%s\n", name, setUnset(set))
		}
		for _, e := range d.Entries {
			fmt.Fprintf(w, "ai:   %-20s matched=%-5v weight=%d %s\n", e.Name, e.Matched, e.Weight, e.Note)
		}
	})
}

func setUnset(b bool) string {
	if b {
		return "(set)"
	}
	return "(unset)"
}

func applyOverrideBlock(o map[string]interface{}, r *Resolved) {
	if v, ok := o["model"].(string); ok && v != "" {
		r.Model = v
	}
	if v, ok := o["base_url"].(string); ok && v != "" {
		r.BaseURL = v
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

// NewClient builds the LLM client from the global config alone.
func NewClient() (llm.Client, error) {
	return NewClientWith(nil)
}

// NewClientWith builds the LLM client using global config plus an
// optional per-caller override block. Returns (nil, nil) when AI is
// disabled in the override (allowing modules to opt out).
func NewClientWith(override map[string]interface{}) (llm.Client, error) {
	r, err := resolveE(pkgconfig.Get().AI, override)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, nil
	}
	cfg := llm.Config{
		Provider:  llm.Provider(r.Provider),
		Model:     r.Model,
		APIKey:    r.APIKey,
		APIKeyEnv: r.APIKeyEnv,
		BaseURL:   r.BaseURL,
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
	// Bedrock uses the AWS auth chain, not an API key.
	if r.Provider == "bedrock" {
		return os.Getenv("AWS_BEARER_TOKEN_BEDROCK") != "" ||
			os.Getenv("AWS_ACCESS_KEY_ID") != "" ||
			os.Getenv("AWS_PROFILE") != ""
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
