package mcp

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// helpLines contains all help content as lines
var helpLines = []string{
	"MCP v0.1.0 - Keyboard Shortcuts",
	"",
	"NAVIGATION",
	"  ↑ / k / Ctrl+K    Move up",
	"  ↓ / j / Ctrl+J    Move down",
	"  → / l             Expand preset",
	"  ← / h             Collapse preset",
	"  n / N             Next/prev section (cycles tags, presets, MCPs)",
	"  Home              Jump to first",
	"  End               Jump to last",
	"",
	"ACTIONS",
	"  Space             Toggle MCP/Preset/Tag",
	"  Enter             Apply changes and exit",
	"  Ctrl+S            Apply changes and stay",
	"",
	"ITEM TYPES",
	"  ☑ MCP Name        Regular MCP server",
	"  📦 preset:name    Preset group (→ to expand)",
	"  🏷  tag:name      Tag group (Space toggles all)",
	"",
	"MODES",
	"  /                 Open search (filter MCPs)",
	"  ?                 Toggle this help screen",
	"  c                 Switch AI tool/scope",
	"  f                 Toggle file filter (source tracking)",
	"",
	"SECTIONS",
	"  0                 Collapse/expand Validation",
	"  1                 Collapse/expand Presets",
	"  2                 Collapse/expand Tags",
	"  3                 Collapse/expand MCPs",
	"",
	"VALIDATION",
	"  v                 Refresh validation",
	"",
	"SEARCH MODE (when / pressed)",
	"  Type              Filter MCPs by text",
	"  Shift+Space       Type a space in search",
	"  ESC               Clear search and exit",
	"  Enter             Keep search and exit",
	"",
	"QUIT",
	"  q / Ctrl+C        Quit application",
}

// renderHelp renders the help overlay with scrolling support
func (m Model) renderHelp() string {
	// Calculate visible lines based on viewport height
	// Reserve space for border (2), padding (2), scroll indicators (2), and margins (4)
	visibleLines := m.viewportHeight - 10
	if visibleLines < 5 {
		visibleLines = 5
	}

	totalLines := len(helpLines)

	// Clamp scroll offset
	maxOffset := totalLines - visibleLines
	if maxOffset < 0 {
		maxOffset = 0
	}
	offset := m.helpScrollOffset
	if offset > maxOffset {
		offset = maxOffset
	}

	// Calculate visible range
	endIdx := offset + visibleLines
	if endIdx > totalLines {
		endIdx = totalLines
	}

	// Build visible content
	var content strings.Builder

	// Scroll up indicator
	if offset > 0 {
		content.WriteString(lipgloss.NewStyle().Foreground(colorGray).Render(fmt.Sprintf("  ↑ %d more above", offset)))
		content.WriteString("\n")
	}

	// Visible help lines
	for i := offset; i < endIdx; i++ {
		content.WriteString(helpLines[i])
		content.WriteString("\n")
	}

	// Scroll down indicator
	remaining := totalLines - endIdx
	if remaining > 0 {
		content.WriteString(lipgloss.NewStyle().Foreground(colorGray).Render(fmt.Sprintf("  ↓ %d more below", remaining)))
		content.WriteString("\n")
	}

	content.WriteString("\n")
	content.WriteString(lipgloss.NewStyle().Foreground(colorGray).Render("j/k scroll • ? or ESC to close"))

	helpStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBlue).
		Padding(1, 2).
		Width(60).
		Foreground(lipgloss.Color("255"))

	overlay := lipgloss.NewStyle().
		Background(lipgloss.Color("0")).
		Foreground(lipgloss.Color("255"))

	help := helpStyle.Render(content.String())

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(help)
	b.WriteString("\n")

	return overlay.Render(b.String())
}
