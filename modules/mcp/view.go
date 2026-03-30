package mcp

import (
	"fmt"
	"regexp"
	"strings"

	tea "charm.land/bubbletea/v2"

	"charm.land/lipgloss/v2"
)

// stripAnsi removes ANSI escape codes from a string
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// Color palette
var (
	colorGreen  = lipgloss.Color("82")  // Checked/installed
	colorRed    = lipgloss.Color("196") // Will remove
	colorYellow = lipgloss.Color("226") // Pending install
	colorBlue   = lipgloss.Color("39")  // Selection
	colorGray   = lipgloss.Color("240") // Unchecked/dimmed
	colorCyan   = lipgloss.Color("51")  // Presets (future)
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86"))

	mcpStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("255")).
			Bold(true).
			PaddingLeft(2).
			PaddingRight(2).
			MarginLeft(1)

	checkedStyle = lipgloss.NewStyle().
			Foreground(colorGreen).
			Bold(true)

	uncheckedStyle = lipgloss.NewStyle().
			Foreground(colorGray)

	pendingInstallStyle = lipgloss.NewStyle().
				Foreground(colorYellow)

	pendingRemoveStyle = lipgloss.NewStyle().
				Foreground(colorRed)

	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("240")).
			Foreground(lipgloss.Color("255")).
			Padding(0, 1)
)

// View renders the TUI (Bubble Tea MVU pattern)
func (m Model) View() tea.View {
	v := tea.View{}
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion

	if m.quitting {
		if m.statusMessage != "" {
			v.SetContent(m.statusMessage + "\n")
			return v
		}
		v.SetContent("Goodbye!\n")
		return v
	}

	// Show help overlay if active
	if m.showHelp {
		v.SetContent(m.renderHelp())
		return v
	}

	// Show context switcher if active
	if m.showContextSwitcher {
		v.SetContent(m.renderContextSwitcher())
		return v
	}

	// Render base UI, then overlay extra params modal if open
	base := m.renderBaseView()
	if m.showExtraModal {
		v.SetContent(m.renderWithModalOverlay(base))
		return v
	}
	v.SetContent(base)
	return v
}

// renderBaseView renders the main list UI (without modal).
func (m Model) renderBaseView() string {
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("MCP v0.1.0"))
	b.WriteString("\n\n")

	// Search input
	if m.searchFocused {
		b.WriteString("Search: " + m.searchInput.View())
	} else {
		searchHint := "Press / to search"
		if m.searchQuery != "" {
			searchHint = fmt.Sprintf("Search: %s (press / to edit, ESC to clear)", m.searchQuery)
		}
		b.WriteString(searchHint)
	}
	b.WriteString("\n\n")

	// Display List (MCPs, Presets, Tags)
	if len(m.filteredItems) == 0 {
		b.WriteString("No items match your search\n")
	} else {
		// Calculate visible range
		headerLines := 4
		footerLines := 6
		effectiveHeight := m.viewportHeight - headerLines - footerLines
		if effectiveHeight < 5 {
			effectiveHeight = 5
		}
		endIdx := m.scrollOffset + effectiveHeight
		if endIdx > len(m.filteredItems) {
			endIdx = len(m.filteredItems)
		}

		// Show scroll indicator at top if scrolled down
		if m.scrollOffset > 0 {
			b.WriteString(lipgloss.NewStyle().Foreground(colorGray).PaddingLeft(2).Render(fmt.Sprintf("↑ %d more above", m.scrollOffset)))
			b.WriteString("\n")
		}

		for i := m.scrollOffset; i < endIdx; i++ {
			item := m.filteredItems[i]
			var line string

			switch item.Type {
			case ListItemSeparator:
				// Section separator (not selectable)
				separatorStyle := lipgloss.NewStyle().
					Foreground(colorGray).
					Bold(true)
				line = separatorStyle.Render("\n─── " + item.Name + " ───")

			case ListItemPreset:
				// Preset item with checkbox (checked if all MCPs in preset are checked)
				allChecked := true
				for _, mcpName := range item.MCPNames {
					if !m.checkedMCPs[mcpName] {
						allChecked = false
						break
					}
				}

				var checkbox string
				if allChecked {
					checkbox = checkedStyle.Render("☑")
				} else {
					checkbox = uncheckedStyle.Render("☐")
				}

				icon := "📦"
				expandIcon := ""
				if m.expandedPresets[item.Name] {
					icon = "📂" // Open folder when expanded
					expandIcon = " ▼"
				} else {
					expandIcon = " ▶"
				}
				line = checkbox + " " + lipgloss.NewStyle().Foreground(colorCyan).Render(icon+" preset:"+item.Name+expandIcon)

			case ListItemTag:
				// Tag item with checkbox (checked if all MCPs with tag are checked)
				allChecked := true
				for _, mcpName := range item.MCPNames {
					if !m.checkedMCPs[mcpName] {
						allChecked = false
						break
					}
				}

				var checkbox string
				if allChecked {
					checkbox = checkedStyle.Render("☑")
				} else {
					checkbox = uncheckedStyle.Render("☐")
				}

				tagIcon := "▶"
				if m.expandedTags[item.Name] {
					tagIcon = "▼"
				}
				line = checkbox + " " + lipgloss.NewStyle().Foreground(colorYellow).Render("🏷  tag:"+item.Name+" "+tagIcon)

			case ListItemMCP:
				// Regular MCP item
				isIndented := strings.HasPrefix(item.Name, "  ")
				mcpName := strings.TrimSpace(item.Name) // Remove indent if child of preset
				isChecked := m.checkedMCPs[mcpName]
				wasInstalled := m.installedMCPs[mcpName]
				isPending := isChecked != wasInstalled

				// Build checkbox with color
				var checkbox string
				var checkboxStyle lipgloss.Style
				if isChecked {
					checkbox = "☑"
					checkboxStyle = checkedStyle
				} else {
					checkbox = "☐"
					checkboxStyle = uncheckedStyle
				}

				// Build line with proper indentation for children
				indent := ""
				if isIndented {
					indent = "    " // 4 spaces for child items
				}
				line = indent + checkboxStyle.Render(checkbox) + " " + mcpName

				// Add tags
				if item.MCP != nil && len(item.MCP.Tags) > 0 {
					tags := strings.Join(item.MCP.Tags, ", ")
					line += lipgloss.NewStyle().Foreground(colorGray).Render(fmt.Sprintf(" [%s]", tags))
				}

				// Add extra param badges
				line += m.renderExtraParamBadges(mcpName)

				// Add pending indicator
				if isPending {
					if isChecked && !wasInstalled {
						line += pendingInstallStyle.Render(" (pending)")
					} else if !isChecked && wasInstalled {
						line += pendingRemoveStyle.Render(" (will remove)")
					}
				}

			case ListItemValidation:
				// Validation issue item
				if item.ValidationIssue != nil {
					issue := item.ValidationIssue
					var icon string
					var style lipgloss.Style
					if issue.Severity == SeverityError {
						icon = "✗"
						style = lipgloss.NewStyle().Foreground(colorRed)
					} else {
						icon = "!"
						style = lipgloss.NewStyle().Foreground(colorYellow)
					}
					line = style.Render(icon + " " + issue.Source + ": " + issue.Message)
				}
			}

			// Apply selection highlight
			if i == m.cursor {
				// Strip ANSI codes so selectedStyle fully applies
				plainLine := ansiRegex.ReplaceAllString(line, "")
				b.WriteString(selectedStyle.Render("> " + plainLine))
			} else {
				b.WriteString(mcpStyle.Render("  " + line))
			}
			b.WriteString("\n")
		}

		// Show scroll indicator at bottom if more items below
		remaining := len(m.filteredItems) - endIdx
		if remaining > 0 {
			b.WriteString(lipgloss.NewStyle().Foreground(colorGray).Render(fmt.Sprintf("  ↓ %d more below\n", remaining)))
		}
	}

	b.WriteString("\n")

	// Status message (if any)
	if m.statusMessage != "" {
		statusStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Bold(true)
		b.WriteString(statusStyle.Render(m.statusMessage))
		b.WriteString("\n\n")
	}

	// Status bar with pending changes
	statusText := FormatContext(m.context, m.contextSource)

	// Calculate pending changes
	toInstall := 0
	toRemove := 0
	for name, checked := range m.checkedMCPs {
		wasInstalled := m.installedMCPs[name]
		if checked && !wasInstalled {
			toInstall++
		} else if !checked && wasInstalled {
			toRemove++
		}
	}

	if toInstall > 0 || toRemove > 0 {
		statusText += " • "
		if toInstall > 0 {
			statusText += fmt.Sprintf("+%d", toInstall)
		}
		if toRemove > 0 {
			if toInstall > 0 {
				statusText += " "
			}
			statusText += fmt.Sprintf("-%d", toRemove)
		}
		statusText += " pending"
	}

	b.WriteString(statusBarStyle.Render(statusText))
	b.WriteString("\n\n")

	// Help
	helpText := "↑/k ↓/j nav • n/N section • Space toggle • e extra params • g global • Enter apply • / search • q quit"
	if m.searchFocused {
		helpText += " • Shift+Space for space"
	}
	b.WriteString(helpText + "\n")

	return b.String()
}

// renderWithModalOverlay places the extra params modal centered over the base view.
func (m Model) renderWithModalOverlay(base string) string {
	lines := strings.Split(base, "\n")
	height := len(lines)
	width := 0
	for _, l := range lines {
		w := lipgloss.Width(l)
		if w > width {
			width = w
		}
	}
	if width < 50 {
		width = 50
	}
	if height < 20 {
		height = 20
	}

	modal := m.extraModal.View()
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, modal.Content,
		lipgloss.WithWhitespaceChars("·"),
		lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("236"))),
	)
}

// renderExtraParamBadges returns badge strings for extra params active on an MCP.
// Per-MCP params show as [key], global params show as [G:key].
func (m Model) renderExtraParamBadges(mcpName string) string {
	if m.globalConfig == nil || len(m.globalConfig.ExtraParams) == 0 {
		return ""
	}

	badgeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("135")).
		Bold(true)
	globalBadgeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("226")).
		Bold(true)

	var badges []string
	seen := make(map[string]bool)

	// Per-MCP badges (take precedence)
	if perMCP, ok := m.mcpExtraParams[mcpName]; ok {
		for _, p := range m.globalConfig.ExtraParams {
			if perMCP[p.Key] {
				badges = append(badges, badgeStyle.Render("["+p.Key+"]"))
				seen[p.Key] = true
			}
		}
	}

	// Global badges (only if not already shown as per-MCP)
	for _, p := range m.globalConfig.ExtraParams {
		if m.globalExtraParams[p.Key] && !seen[p.Key] {
			badges = append(badges, globalBadgeStyle.Render("[G:"+p.Key+"]"))
		}
	}

	if len(badges) == 0 {
		return ""
	}
	return " " + strings.Join(badges, " ")
}
