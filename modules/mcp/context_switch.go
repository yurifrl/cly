package mcp

import (
	tea "charm.land/bubbletea/v2"
)

type contextSwitchedMsg struct {
	newContext Context
	adapter    Adapter
}

func (m Model) switchContext(ai, scope string) tea.Cmd {
	return func() tea.Msg {
		var adapter Adapter
		switch ai {
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

		if adapter == nil || !adapter.IsInstalled() {
			return contextSwitchedMsg{}
		}

		return contextSwitchedMsg{
			newContext: Context{AI: ai, Scope: scope},
			adapter:    adapter,
		}
	}
}
