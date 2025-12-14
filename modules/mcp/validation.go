package mcp

import (
	"fmt"
	"os/exec"
	"time"
)

// Severity indicates how serious an issue is
type Severity int

const (
	SeverityError   Severity = iota // Will prevent MCP from working
	SeverityWarning                 // Might cause issues
)

// Issue represents a single validation problem
type Issue struct {
	Severity Severity
	Source   string // "mcp:name", "preset:name", "catalog", "config"
	Message  string
}

// ValidationResult contains all validation issues
type ValidationResult struct {
	Issues    []Issue
	Timestamp time.Time
}

// ErrorCount returns the number of errors
func (r *ValidationResult) ErrorCount() int {
	count := 0
	for _, issue := range r.Issues {
		if issue.Severity == SeverityError {
			count++
		}
	}
	return count
}

// WarningCount returns the number of warnings
func (r *ValidationResult) WarningCount() int {
	count := 0
	for _, issue := range r.Issues {
		if issue.Severity == SeverityWarning {
			count++
		}
	}
	return count
}

// HasIssues returns true if there are any issues
func (r *ValidationResult) HasIssues() bool {
	return len(r.Issues) > 0
}

// Validate runs all validation checks
func Validate(catalog *Catalog, cfg *GlobalConfig, adapter Adapter, scope string) *ValidationResult {
	result := &ValidationResult{
		Timestamp: time.Now(),
	}

	// Validate MCPs in catalog
	if catalog != nil {
		for _, mcp := range catalog.GetAll() {
			issues := validateMCP(mcp)
			result.Issues = append(result.Issues, issues...)
		}
	}

	// Validate presets reference existing MCPs
	if cfg != nil && catalog != nil {
		issues := validatePresets(cfg.Presets, catalog)
		result.Issues = append(result.Issues, issues...)
	}

	// Validate installed config
	if adapter != nil {
		issues := validateInstalledConfig(adapter, scope)
		result.Issues = append(result.Issues, issues...)
	}

	return result
}

// validateMCP checks a single MCP for issues
func validateMCP(mcp MCP) []Issue {
	var issues []Issue

	// Check command or URL exists
	if mcp.Command == "" && mcp.URL == "" {
		issues = append(issues, Issue{
			Severity: SeverityError,
			Source:   fmt.Sprintf("mcp:%s", mcp.Name),
			Message:  "Missing command or URL",
		})
	}

	// Check command exists in PATH (for stdio MCPs)
	if mcp.Command != "" {
		if _, err := exec.LookPath(mcp.Command); err != nil {
			issues = append(issues, Issue{
				Severity: SeverityWarning,
				Source:   fmt.Sprintf("mcp:%s", mcp.Name),
				Message:  fmt.Sprintf("Command '%s' not found in PATH", mcp.Command),
			})
		}
	}

	return issues
}

// validatePresets checks that presets reference existing MCPs
func validatePresets(presets map[string][]string, catalog *Catalog) []Issue {
	var issues []Issue

	for presetName, mcpNames := range presets {
		for _, mcpName := range mcpNames {
			if _, ok := catalog.Get(mcpName); !ok {
				issues = append(issues, Issue{
					Severity: SeverityWarning,
					Source:   fmt.Sprintf("preset:%s", presetName),
					Message:  fmt.Sprintf("References non-existent MCP '%s'", mcpName),
				})
			}
		}
	}

	return issues
}

// validateInstalledConfig checks the currently installed MCP config
func validateInstalledConfig(adapter Adapter, scope string) []Issue {
	var issues []Issue

	toolCfg, err := adapter.ReadConfig(scope)
	if err != nil {
		return issues
	}

	for name, mcp := range toolCfg.MCPServers {
		if mcp.Command == "" && mcp.URL == "" {
			issues = append(issues, Issue{
				Severity: SeverityError,
				Source:   fmt.Sprintf("installed:%s", name),
				Message:  "Missing command or URL",
			})
		}

		if mcp.Command != "" {
			if _, err := exec.LookPath(mcp.Command); err != nil {
				issues = append(issues, Issue{
					Severity: SeverityWarning,
					Source:   fmt.Sprintf("installed:%s", name),
					Message:  fmt.Sprintf("Command '%s' not found in PATH", mcp.Command),
				})
			}
		}
	}

	return issues
}
