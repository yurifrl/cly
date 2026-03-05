package mcp

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ExtraParamsModal is a self-contained overlay for toggling extra params on a single MCP.
// It is not a Bubbletea program — it is embedded in Model and rendered as a lipgloss overlay.
type ExtraParamsModal struct {
	mcpName string
	params  []ExtraParam    // available params from GlobalConfig
	active  map[string]bool // which param keys are currently toggled on
	cursor  int
}

// newExtraParamsModal creates a modal for the given MCP using available params from global config.
// initialActive is the current per-MCP param state (may be nil).
func newExtraParamsModal(mcpName string, params []ExtraParam, initialActive map[string]bool) ExtraParamsModal {
	active := make(map[string]bool)
	for k, v := range initialActive {
		active[k] = v
	}
	return ExtraParamsModal{
		mcpName: mcpName,
		params:  params,
		active:  active,
		cursor:  0,
	}
}

// Update processes a key and returns (updated modal, closed, applied).
// closed=true means user dismissed (esc/q). applied=true means user confirmed (enter).
// Both closed and applied are mutually exclusive; check applied first.
func (m ExtraParamsModal) Update(key string) (ExtraParamsModal, bool, bool) {
	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.params)-1 {
			m.cursor++
		}
	case " ":
		if len(m.params) > 0 {
			k := m.params[m.cursor].Key
			m.active[k] = !m.active[k]
		}
	case "enter", "ctrl+s":
		return m, false, true // applied
	case "esc", "q":
		return m, true, false // closed/cancelled
	}
	return m, false, false
}

// ActiveParams returns a map of key→value for all toggled-on params.
func (m ExtraParamsModal) ActiveParams() map[string]interface{} {
	result := make(map[string]interface{})
	for _, p := range m.params {
		if m.active[p.Key] {
			result[p.Key] = p.Value
		}
	}
	return result
}

// View renders the modal as a bordered lipgloss box ready to be overlaid.
func (m ExtraParamsModal) View() string {
	modalBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("135")).
		Padding(1, 2)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("135"))

	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true)

	checkOn := lipgloss.NewStyle().
		Foreground(lipgloss.Color("82")).
		Bold(true)

	checkOff := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	selectedRowStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("255")).
		Bold(true)

	var sb strings.Builder
	sb.WriteString(titleStyle.Render(fmt.Sprintf("Extra Params: %s", m.mcpName)))
	sb.WriteString("\n\n")

	if len(m.params) == 0 {
		sb.WriteString(hintStyle.Render("No extra params configured.\nAdd them to extraParams in config.yaml"))
	} else {
		for i, p := range m.params {
			label := p.Label
			if label == "" {
				label = fmt.Sprintf("%s = %v", p.Key, p.Value)
			}

			var check string
			if m.active[p.Key] {
				check = checkOn.Render("[✓]")
			} else {
				check = checkOff.Render("[ ]")
			}

			row := fmt.Sprintf("%s %s", check, label)
			if i == m.cursor {
				// Pad to consistent width for highlight
				row = selectedRowStyle.Render(fmt.Sprintf("  %s %s  ", check, label))
			}
			sb.WriteString(row)
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString(hintStyle.Render("j/k: move  space: toggle  enter: apply  esc: cancel"))

	return modalBorder.Render(sb.String())
}
