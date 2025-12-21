package mcp

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

type contextOption struct {
	ai          string
	scope       string
	label       string
	description string
}

var contextOptions = []contextOption{
	{"claude", "user", "Claude Code (user)", "global, available in all your projects"},
	{"claude", "local", "Claude Code (local)", "edits ~/.claude.json under projects[cwd] key"},
	{"claude", "project", "Claude Code (project)", "edits .mcp.json, shared/added to repo"},
	{"cursor", "user", "Cursor IDE (user)", "global cursor settings"},
	{"cursor", "project", "Cursor IDE (project)", "project .cursor/mcp.json"},
	{"desktop", "user", "Claude Desktop", "desktop app config"},
}

func (m Model) renderContextSwitcher() string {
	content := "Select Context:\n\n"

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	for i, opt := range contextOptions {
		// Check if tool is installed
		var adapter Adapter
		switch opt.ai {
		case "claude":
			adapter = &ClaudeAdapter{}
		case "cursor":
			adapter = &CursorAdapter{}
		case "desktop":
			adapter = &DesktopAdapter{}
		}

		installed := ""
		if adapter != nil && !adapter.IsInstalled() {
			installed = " (not installed)"
		}

		// Get config path
		configPath := ""
		if adapter != nil {
			if path, err := adapter.GetConfigPath(opt.scope); err == nil {
				configPath = path
			}
		}

		// Current selection
		current := ""
		if opt.ai == m.context.AI && opt.scope == m.context.Scope {
			current = " ← current"
		}

		// Format: Label (path) current
		//         description
		label := fmt.Sprintf("%s (%s)%s%s", opt.label, configPath, installed, current)
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
