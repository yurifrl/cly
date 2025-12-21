package mcp

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type operationCompleteMsg struct {
	success bool
	message string
}

func (m Model) applyChanges() tea.Cmd {
	return func() tea.Msg {
		return m.performOperation()
	}
}

func (m Model) performOperation() tea.Msg {
	var mcpsToWrite []MCP

	for _, mcp := range m.availableMCPs {
		if m.checkedMCPs[mcp.Name] {
			mcpsToWrite = append(mcpsToWrite, mcp)
		}
	}

	if err := m.adapter.WriteConfig(m.context.Scope, mcpsToWrite); err != nil {
		return operationCompleteMsg{
			success: false,
			message: "Failed to update config: " + err.Error(),
		}
	}

	installed := make(map[string]bool)
	for _, mcp := range mcpsToWrite {
		installed[mcp.Name] = true
	}

	var installedNames []string
	var removedNames []string
	for name, checked := range m.checkedMCPs {
		wasInstalled := m.installedMCPs[name]
		if checked && !wasInstalled {
			installedNames = append(installedNames, name)
		} else if !checked && wasInstalled {
			removedNames = append(removedNames, name)
		}
	}

	message := "No changes"
	if len(installedNames) > 0 || len(removedNames) > 0 {
		message = fmt.Sprintf("%s:%s\n", m.context.AI, m.context.Scope)

		if len(installedNames) > 0 {
			message += fmt.Sprintf("+ Enabled: %s", strings.Join(installedNames, ", "))
		}
		if len(removedNames) > 0 {
			if len(installedNames) > 0 {
				message += "\n"
			}
			message += fmt.Sprintf("- Disabled: %s", strings.Join(removedNames, ", "))
		}
	}

	return operationCompleteMsg{
		success: true,
		message: message,
	}
}

type uiPreferencesSavedMsg struct {
	success bool
	message string
}

func (m Model) saveUIPreferences() tea.Cmd {
	return func() tea.Msg {
		var hiddenSections []string
		for section, hidden := range m.hiddenSections {
			if hidden {
				hiddenSections = append(hiddenSections, section)
			}
		}

		if m.globalConfig != nil {
			m.globalConfig.UI.HiddenSections = hiddenSections
			if err := SaveGlobalConfig(m.globalConfig); err != nil {
				return uiPreferencesSavedMsg{
					success: false,
					message: "Failed to save preferences: " + err.Error(),
				}
			}
		}

		return uiPreferencesSavedMsg{
			success: true,
			message: "",
		}
	}
}
