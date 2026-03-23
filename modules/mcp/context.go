package mcp

import (
	"fmt"
	"os"
)

// Detector handles context detection using config hierarchy
type Detector struct {
	globalConfig  *GlobalConfig
	projectConfig *ProjectConfig
	cliAI         string
	cliScope      string
}

// NewDetector creates a new context detector
func NewDetector(globalCfg *GlobalConfig, projectCfg *ProjectConfig, cliAI, cliScope string) *Detector {
	return &Detector{
		globalConfig:  globalCfg,
		projectConfig: projectCfg,
		cliAI:         cliAI,
		cliScope:      cliScope,
	}
}

// DetectContext determines which AI tool and scope to use based on priority order:
// 1. CLI flags (highest priority)
// 2. Project config (.mcpcli.json)
// 3. Global project mapping (current path in global config)
// 4. Global defaults
// 5. Hardcoded fallback
func (d *Detector) DetectContext() (Context, error) {
	// Priority 1: CLI flags
	if d.cliAI != "" && d.cliScope != "" {
		return Context{AI: d.cliAI, Scope: d.cliScope}, nil
	}

	// Priority 2: Project config
	if d.projectConfig != nil {
		ctx := Context{
			AI:    d.projectConfig.AI,
			Scope: d.projectConfig.Scope,
		}
		if d.cliAI != "" {
			ctx.AI = d.cliAI
		}
		if d.cliScope != "" {
			ctx.Scope = d.cliScope
		}
		return ctx, nil
	}

	// Priority 3: Global project mapping
	if d.globalConfig != nil {
		cwd, err := os.Getwd()
		if err == nil {
			if projectCfg, ok := d.globalConfig.Projects[cwd]; ok {
				ctx := Context{
					AI:    projectCfg.AI,
					Scope: projectCfg.Scope,
				}
				if d.cliAI != "" {
					ctx.AI = d.cliAI
				}
				if d.cliScope != "" {
					ctx.Scope = d.cliScope
				}
				return ctx, nil
			}
		}
	}

	// Priority 4: Global defaults
	if d.globalConfig != nil {
		ctx := Context{
			AI:    d.globalConfig.Defaults.AI,
			Scope: d.globalConfig.Defaults.Scope,
		}
		if d.cliAI != "" {
			ctx.AI = d.cliAI
		}
		if d.cliScope != "" {
			ctx.Scope = d.cliScope
		}
		return ctx, nil
	}

	// Priority 5: Hardcoded fallback
	return Context{AI: "agents", Scope: "project"}, nil
}

// GetContextSource returns a human-readable description of where the context came from
func (d *Detector) GetContextSource() string {
	if d.cliAI != "" || d.cliScope != "" {
		return "via CLI flags"
	}
	if d.projectConfig != nil {
		return "via .mcpcli.json"
	}

	if d.globalConfig != nil {
		cwd, _ := os.Getwd()
		if _, ok := d.globalConfig.Projects[cwd]; ok {
			return "via global project mapping"
		}
		if d.globalConfig.Defaults.AI != "" {
			return "via global defaults"
		}
	}

	return "hardcoded fallback"
}

// FormatContext returns a formatted string showing the context
func FormatContext(ctx Context, source string) string {
	return fmt.Sprintf("📍 %s (%s) • %s", ctx.AI, ctx.Scope, source)
}
