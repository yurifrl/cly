package mcp

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

type contextOption struct {
	ai          string
	scope       string
	label       string
	description string
}

var contextOptions = []contextOption{
	{"agents", "project", "Agents (project)", "edits .agents/mcp.json, shared/added to repo"},
	{"agents", "user", "Agents (user)", "global ~/.agents/mcp.json"},
	{"claude", "project", "Claude Code (project)", "edits .mcp.json, shared/added to repo"},
	{"claude", "user", "Claude Code (user)", "global, available in all your projects"},
	{"claude", "local", "Claude Code (local)", "edits ~/.claude.json under projects[cwd] key"},
	{"pi", "project", "Pi (project)", "project .pi/mcp.json"},
	{"pi", "user", "Pi (user)", "global ~/.pi/agent/mcp.json"},
	{"cursor", "user", "Cursor IDE (user)", "global cursor settings"},
	{"cursor", "project", "Cursor IDE (project)", "project .cursor/mcp.json"},
	{"desktop", "user", "Claude Desktop", "desktop app config"},
}

// formatMCPCount returns a display string for the MCP count
func formatMCPCount(count int) string {
	if count == 0 {
		return "(empty)"
	}
	if count == 1 {
		return "[1 MCP]"
	}
	return fmt.Sprintf("[%d MCPs]", count)
}

func (m Model) renderContextSwitcher() string {
	content := "Select Context:\n\n"

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	for i, opt := range contextOptions {
		// Check if tool is installed
		var adapter Adapter
		switch opt.ai {
		case "agents":
			adapter = &AgentsAdapter{}
		case "claude":
			adapter = &ClaudeAdapter{}
		case "pi":
			adapter = &PiAdapter{}
		case "cursor":
			adapter = &CursorAdapter{}
		case "desktop":
			adapter = &DesktopAdapter{}
		}

		installed := ""
		if adapter != nil && !adapter.IsInstalled() {
			installed = " (not installed)"
		}

		// Get config path and MCP count
		configPath := ""
		mcpCount := 0
		if adapter != nil {
			if path, err := adapter.GetConfigPath(opt.scope); err == nil {
				configPath = path
			}
			if cfg, err := adapter.ReadConfig(opt.scope); err == nil && cfg != nil {
				mcpCount = len(cfg.MCPServers)
			}
		}

		// Current selection
		current := ""
		if opt.ai == m.context.AI && opt.scope == m.context.Scope {
			current = " ← current"
		}

		// Format: Label (path) [N MCPs] current
		//         description
		label := fmt.Sprintf("%s (%s) %s%s%s", opt.label, configPath, formatMCPCount(mcpCount), installed, current)
		desc := dimStyle.Render("  " + opt.description)

		if i == m.contextMenuCursor {
			content += selectedStyle.Render("> "+label) + "\n"
			content += desc + "\n"
		} else {
			content += "  " + label + "\n"
			content += desc + "\n"
		}
	}

	content += "\nEnter: Select • ESC: Cancel • ↑↓: Navigate"

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBlue).
		Padding(1, 2).
		Width(70)

	return boxStyle.Render(content)
}
