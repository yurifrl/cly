package ai

import (
	"fmt"
	"strings"
)

// Entry is one named provider in the ai.providers list.
type Entry struct {
	Name      string
	Provider  string
	Model     string
	APIKey    string
	APIKeyEnv string
	BaseURL   string
	Weight    int
	Condition string
	Default   bool
	cond      condExpr
}

// parseProviders validates the ai.providers list from raw config.
// providerEnv comes from ai.go (provider -> conventional env var name).
func parseProviders(global map[string]interface{}) ([]Entry, error) {
	raw, ok := global["providers"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("ai.providers must be a list of named entries (map form was removed; see docs/superpowers/specs/2026-08-12-ai-provider-conditions-design.md)")
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("ai.providers is empty: define at least one entry")
	}
	seen := map[string]bool{}
	entries := make([]Entry, 0, len(raw))
	for i, item := range raw {
		m, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("ai.providers[%d]: must be a map", i)
		}
		e := Entry{Provider: defaultProvider}
		e.Name, _ = m["name"].(string)
		if e.Name == "" {
			return nil, fmt.Errorf("ai.providers[%d]: name is required", i)
		}
		if seen[e.Name] {
			return nil, fmt.Errorf("ai.providers[%d]: duplicate name %q", i, e.Name)
		}
		seen[e.Name] = true
		if v, ok := m["provider"].(string); ok && v != "" {
			e.Provider = v
		}
		e.Model, _ = m["model"].(string)
		e.BaseURL, _ = m["base_url"].(string)
		e.Weight = toInt(m["weight"])
		e.Default, _ = m["default"].(bool)
		e.Condition, _ = m["condition"].(string)
		if e.Condition != "" {
			cond, err := parseCondition(e.Condition)
			if err != nil {
				return nil, fmt.Errorf("ai.providers[%d] (%s): invalid condition: %w", i, e.Name, err)
			}
			e.cond = cond
		}
		if v, ok := m["api_key"].(string); ok && v != "" {
			setKeyOrEnvStr(v, &e)
		}
		if v, ok := m["api_key_env"].(string); ok && v != "" {
			e.APIKey = ""
			e.APIKeyEnv = v
		}
		if e.APIKey == "" && e.APIKeyEnv == "" {
			e.APIKeyEnv = providerEnv[e.Provider]
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// toInt accepts the numeric shapes YAML decoders produce.
func toInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	}
	return 0
}

// setKeyOrEnvStr mirrors setKeyOrEnv in ai.go but targets Entry fields.
func setKeyOrEnvStr(s string, e *Entry) {
	s = strings.TrimSpace(s)
	switch {
	case strings.HasPrefix(s, "${") && strings.HasSuffix(s, "}"):
		e.APIKey = ""
		e.APIKeyEnv = s[2 : len(s)-1]
	case strings.HasPrefix(s, "$") && len(s) > 1:
		e.APIKey = ""
		e.APIKeyEnv = s[1:]
	default:
		e.APIKey = s
		e.APIKeyEnv = ""
	}
}
