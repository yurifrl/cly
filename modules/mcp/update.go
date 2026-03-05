package mcp

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Init() tea.Cmd {
	return nil
}

func (m *Model) ensureCursorVisible() {
	headerLines := 4
	effectiveHeight := m.viewportHeight - headerLines - 6
	if effectiveHeight < 5 {
		effectiveHeight = 5
	}

	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}
	if m.cursor >= m.scrollOffset+effectiveHeight {
		m.scrollOffset = m.cursor - effectiveHeight + 1
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

func listItemRenderHeight(item ListItem) int {
	if item.Type == ListItemSeparator {
		return 2 // separator renders a blank line + separator line
	}
	return 1
}

func (m *Model) indexAtMouseY(y int) int {
	if len(m.filteredItems) == 0 {
		return -1
	}

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

	listStartY := headerLines
	if m.scrollOffset > 0 {
		listStartY++ // top "more above" indicator
	}

	lineY := listStartY
	for i := m.scrollOffset; i < endIdx; i++ {
		h := listItemRenderHeight(m.filteredItems[i])
		if y >= lineY && y < lineY+h {
			return i
		}
		lineY += h
	}

	return -1
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Delegate to extra params modal when it's open
	if m.showExtraModal {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			updated, closed, applied := m.extraModal.Update(keyMsg.String())
			m.extraModal = updated
			if applied {
				// Save per-MCP active params
				active := make(map[string]bool)
				for _, p := range m.extraModal.params {
					active[p.Key] = m.extraModal.active[p.Key]
				}
				m.mcpExtraParams[m.extraModal.mcpName] = active
				m.showExtraModal = false
				m.statusMessage = fmt.Sprintf("Extra params updated for %s", m.extraModal.mcpName)
			} else if closed {
				m.showExtraModal = false
			}
			return m, nil
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewportHeight = msg.Height
		m.ensureCursorVisible()
		return m, nil

	case tea.MouseMsg:
		if m.showHelp {
			if msg.Action != tea.MouseActionPress {
				return m, nil
			}
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				if m.helpScrollOffset > 0 {
					m.helpScrollOffset--
				}
			case tea.MouseButtonWheelDown:
				m.helpScrollOffset++
			}
			return m, nil
		}

		if m.showContextSwitcher {
			if msg.Button == tea.MouseButtonLeft {
				// Box has border + padding; options start a few rows below the top.
				row := msg.Y - 4
				if row >= 0 {
					idx := row / 2
					if idx >= 0 && idx < len(contextOptions) {
						m.contextMenuCursor = idx
						selected := contextOptions[m.contextMenuCursor]
						return m, m.switchContext(selected.ai, selected.scope)
					}
				}
			}
			return m, nil
		}

		switch msg.Button {
		case tea.MouseButtonWheelUp:
			if msg.Action != tea.MouseActionPress {
				return m, nil
			}
			if m.cursor > 0 {
				m.cursor--
				m.ensureCursorVisible()
			}
		case tea.MouseButtonWheelDown:
			if msg.Action != tea.MouseActionPress {
				return m, nil
			}
			if m.cursor < len(m.filteredItems)-1 {
				m.cursor++
				m.ensureCursorVisible()
			}
		case tea.MouseButtonLeft:
			idx := m.indexAtMouseY(msg.Y)
			if idx >= 0 && idx < len(m.filteredItems) {
				m.cursor = idx
				m.ensureCursorVisible()
			}
		}
		return m, nil

	case contextSwitchedMsg:
		if msg.adapter != nil {
			m.context = msg.newContext
			m.adapter = msg.adapter
			m.contextSource = "via TUI context switcher"
			m.showContextSwitcher = false
			m.contextMenuCursor = 0

			m.installedMCPs = make(map[string]bool)
			m.checkedMCPs = make(map[string]bool)
			if toolCfg, err := m.adapter.ReadConfig(m.context.Scope); err == nil {
				for name := range toolCfg.MCPServers {
					m.installedMCPs[name] = true
					m.checkedMCPs[name] = true
				}
			}

			m.runValidation()
			m.statusMessage = "Switched to " + msg.newContext.AI + " (" + msg.newContext.Scope + ")"
		}
		return m, nil

	case operationCompleteMsg:
		m.statusMessage = msg.message

		if msg.success {
			if toolCfg, err := m.adapter.ReadConfig(m.context.Scope); err == nil {
				m.installedMCPs = make(map[string]bool)
				for name := range toolCfg.MCPServers {
					m.installedMCPs[name] = true
				}
			}
			m.runValidation()
		}

		if m.exitAfterApply {
			m.quitting = true
			return m, tea.Quit
		}

		return m, nil

	case uiPreferencesSavedMsg:
		if !msg.success {
			m.statusMessage = msg.message
		}
		return m, nil

	case tea.KeyMsg:
		if m.showContextSwitcher {
			switch msg.String() {
			case "esc":
				m.showContextSwitcher = false
				m.contextMenuCursor = 0
				return m, nil
			case "up", "k":
				if m.contextMenuCursor > 0 {
					m.contextMenuCursor--
				}
				return m, nil
			case "down", "j":
				if m.contextMenuCursor < len(contextOptions)-1 {
					m.contextMenuCursor++
				}
				return m, nil
			case "enter":
				selected := contextOptions[m.contextMenuCursor]
				return m, m.switchContext(selected.ai, selected.scope)
			}
			return m, nil
		}

		if m.showHelp {
			switch msg.Type {
			case tea.KeyUp:
				if m.helpScrollOffset > 0 {
					m.helpScrollOffset--
				}
				return m, nil
			case tea.KeyDown:
				m.helpScrollOffset++
				return m, nil
			case tea.KeyHome:
				m.helpScrollOffset = 0
				return m, nil
			case tea.KeyEnd:
				m.helpScrollOffset = 999
				return m, nil
			case tea.KeyEsc:
				m.showHelp = false
				m.helpScrollOffset = 0
				return m, nil
			}
			switch msg.String() {
			case "?", "q":
				m.showHelp = false
				m.helpScrollOffset = 0
				return m, nil
			case "k":
				if m.helpScrollOffset > 0 {
					m.helpScrollOffset--
				}
				return m, nil
			case "j":
				m.helpScrollOffset++
				return m, nil
			}
			return m, nil
		}

		if m.searchFocused {
			switch msg.String() {
			case "esc":
				m.searchFocused = false
				m.searchInput.Blur()
				m.searchQuery = ""
				m.filteredItems = m.displayItems
				m.cursor = 0
				m.scrollOffset = 0
				return m, nil
			case "enter":
				m.searchFocused = false
				m.searchInput.Blur()
				return m, nil
			case "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			case "up", "ctrl+k":
				if m.cursor > 0 {
					m.cursor--
					for m.cursor >= 0 && m.cursor < len(m.filteredItems) && m.filteredItems[m.cursor].Type == ListItemSeparator {
						m.cursor--
					}
					if m.cursor < 0 {
						m.cursor = 0
					}
				}
				m.ensureCursorVisible()
				return m, nil
			case "down", "ctrl+j":
				if m.cursor < len(m.filteredItems)-1 {
					m.cursor++
					for m.cursor < len(m.filteredItems) && m.filteredItems[m.cursor].Type == ListItemSeparator {
						m.cursor++
					}
					if m.cursor >= len(m.filteredItems) {
						m.cursor = len(m.filteredItems) - 1
					}
				}
				m.ensureCursorVisible()
				return m, nil
			case "ctrl+h", "left":
				if m.cursor < len(m.filteredItems) {
					item := m.filteredItems[m.cursor]
					if item.Type == ListItemPreset {
						m.expandedPresets[item.Name] = false
						m.filteredItems = m.rebuildFilteredItems()
						m.ensureCursorVisible()
					} else if item.Type == ListItemTag {
						m.expandedTags[item.Name] = false
						m.filteredItems = m.rebuildFilteredItems()
						m.ensureCursorVisible()
					}
				}
				return m, nil
			case "ctrl+l", "right":
				if m.cursor < len(m.filteredItems) {
					item := m.filteredItems[m.cursor]
					if item.Type == ListItemPreset {
						m.expandedPresets[item.Name] = true
						m.filteredItems = m.rebuildFilteredItems()
						m.ensureCursorVisible()
					} else if item.Type == ListItemTag {
						m.expandedTags[item.Name] = true
						m.filteredItems = m.rebuildFilteredItems()
						m.ensureCursorVisible()
					}
				}
				return m, nil
			case " ":
				if m.cursor < len(m.filteredItems) {
					item := m.filteredItems[m.cursor]
					switch item.Type {
					case ListItemMCP:
						mcpName := strings.TrimSpace(item.Name)
						m.checkedMCPs[mcpName] = !m.checkedMCPs[mcpName]

					case ListItemPreset, ListItemTag:
						allChecked := true
						for _, mcpName := range item.MCPNames {
							if !m.checkedMCPs[mcpName] {
								allChecked = false
								break
							}
						}
						for _, mcpName := range item.MCPNames {
							m.checkedMCPs[mcpName] = !allChecked
						}
					}
				}
				return m, nil
			}
			m.searchInput, cmd = m.searchInput.Update(msg)
			m.searchQuery = m.searchInput.Value()
			filteredMCPs := m.catalog.Filter(m.searchQuery, m.selectedTags)
			m.filteredItems = m.buildFilteredDisplayItems(filteredMCPs, m.searchQuery)
			m.cursor = 0
			m.scrollOffset = 0
			return m, cmd
		}

		switch msg.String() {
		case "?":
			m.showHelp = !m.showHelp
			return m, nil

		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case " ":
			if m.cursor < len(m.filteredItems) {
				item := m.filteredItems[m.cursor]
				switch item.Type {
				case ListItemMCP:
					mcpName := strings.TrimSpace(item.Name)
					m.checkedMCPs[mcpName] = !m.checkedMCPs[mcpName]

				case ListItemPreset, ListItemTag:
					allChecked := true
					for _, mcpName := range item.MCPNames {
						if !m.checkedMCPs[mcpName] {
							allChecked = false
							break
						}
					}
					for _, mcpName := range item.MCPNames {
						m.checkedMCPs[mcpName] = !allChecked
					}
				}
			}
			return m, nil

		case "right", "l":
			if m.cursor < len(m.filteredItems) {
				item := m.filteredItems[m.cursor]
				if item.Type == ListItemPreset {
					m.expandedPresets[item.Name] = true
					m.filteredItems = m.rebuildFilteredItems()
					m.ensureCursorVisible()
				} else if item.Type == ListItemTag {
					m.expandedTags[item.Name] = true
					m.filteredItems = m.rebuildFilteredItems()
					m.ensureCursorVisible()
				}
			}
			return m, nil

		case "left", "h":
			if m.cursor < len(m.filteredItems) {
				item := m.filteredItems[m.cursor]
				if item.Type == ListItemPreset {
					m.expandedPresets[item.Name] = false
					m.filteredItems = m.rebuildFilteredItems()
					m.ensureCursorVisible()
				} else if item.Type == ListItemTag {
					m.expandedTags[item.Name] = false
					m.filteredItems = m.rebuildFilteredItems()
					m.ensureCursorVisible()
				}
			}
			return m, nil

		case "up", "k", "ctrl+k":
			if m.cursor > 0 {
				m.cursor--
				for m.cursor >= 0 && m.cursor < len(m.filteredItems) && m.filteredItems[m.cursor].Type == ListItemSeparator {
					m.cursor--
				}
				if m.cursor < 0 {
					m.cursor = 0
				}
			}
			m.ensureCursorVisible()
			return m, nil

		case "down", "j", "ctrl+j":
			if m.cursor < len(m.filteredItems)-1 {
				m.cursor++
				for m.cursor < len(m.filteredItems) && m.filteredItems[m.cursor].Type == ListItemSeparator {
					m.cursor++
				}
				if m.cursor >= len(m.filteredItems) {
					m.cursor = len(m.filteredItems) - 1
				}
			}
			m.ensureCursorVisible()
			return m, nil

		case "home":
			m.cursor = 0
			m.scrollOffset = 0
			return m, nil

		case "end":
			if len(m.filteredItems) > 0 {
				m.cursor = len(m.filteredItems) - 1
			}
			m.ensureCursorVisible()
			return m, nil

		case "n":
			sectionStarts := m.findSectionStarts()
			for _, idx := range sectionStarts {
				if idx > m.cursor {
					m.cursor = idx
					m.ensureCursorVisible()
					return m, nil
				}
			}
			if len(sectionStarts) > 0 {
				m.cursor = sectionStarts[0]
				m.ensureCursorVisible()
			}
			return m, nil

		case "N":
			sectionStarts := m.findSectionStarts()
			for i := len(sectionStarts) - 1; i >= 0; i-- {
				if sectionStarts[i] < m.cursor {
					m.cursor = sectionStarts[i]
					m.ensureCursorVisible()
					return m, nil
				}
			}
			if len(sectionStarts) > 0 {
				m.cursor = sectionStarts[len(sectionStarts)-1]
				m.ensureCursorVisible()
			}
			return m, nil
		}

		switch msg.String() {
		case "/":
			m.searchFocused = true
			m.searchInput.Focus()
			return m, textinput.Blink

		case "c":
			m.showContextSwitcher = true
			m.contextMenuCursor = 0
			return m, nil

		case "v":
			m.runValidation()
			if m.validationResult != nil && m.validationResult.HasIssues() {
				m.statusMessage = "Validation refreshed"
			} else {
				m.statusMessage = "No validation issues"
			}
			return m, nil

		case "f":
			if m.selectedFile != "" {
				m.selectedFile = ""
				m.filteredItems = m.displayItems
				m.cursor = 0
				m.scrollOffset = 0
				m.statusMessage = "Filter cleared"
			} else {
				m.statusMessage = "File filter: Feature tracks source files (f key cycles)"
			}
			return m, nil

		case "0":
			m.hiddenSections["validation"] = !m.hiddenSections["validation"]
			m.rebuildDisplayItems()
			if m.hiddenSections["validation"] {
				m.statusMessage = "Validation collapsed"
			} else {
				m.statusMessage = "Validation expanded"
			}
			return m, m.saveUIPreferences()

		case "1":
			m.hiddenSections["presets"] = !m.hiddenSections["presets"]
			m.rebuildDisplayItems()
			if m.hiddenSections["presets"] {
				m.statusMessage = "Presets collapsed"
			} else {
				m.statusMessage = "Presets expanded"
			}
			return m, m.saveUIPreferences()

		case "2":
			m.hiddenSections["tags"] = !m.hiddenSections["tags"]
			m.rebuildDisplayItems()
			if m.hiddenSections["tags"] {
				m.statusMessage = "Tags collapsed"
			} else {
				m.statusMessage = "Tags expanded"
			}
			return m, m.saveUIPreferences()

		case "3":
			m.hiddenSections["mcps"] = !m.hiddenSections["mcps"]
			m.rebuildDisplayItems()
			if m.hiddenSections["mcps"] {
				m.statusMessage = "MCPs collapsed"
			} else {
				m.statusMessage = "MCPs expanded"
			}
			return m, m.saveUIPreferences()

		case "enter":
			m.statusMessage = "Applying changes..."
			m.exitAfterApply = true
			return m, m.applyChanges()

		case "ctrl+s":
			m.statusMessage = "Applying changes..."
			m.exitAfterApply = false
			return m, m.applyChanges()

		case "e":
			// Open extra params modal for the selected MCP
			if m.cursor < len(m.filteredItems) {
				item := m.filteredItems[m.cursor]
				if item.Type == ListItemMCP && item.MCP != nil {
					params := m.availableExtraParams()
					if len(params) == 0 {
						m.statusMessage = "No extra params defined. Add 'extraParams' to ~/.config/mcpcli/config.yaml"
					} else {
						mcpName := strings.TrimSpace(item.Name)
						m.extraModal = newExtraParamsModal(mcpName, params, m.mcpExtraParams[mcpName])
						m.showExtraModal = true
					}
				}
			}
			return m, nil

		case "g":
			// Cycle global extra params: each press toggles the next param on/off
			params := m.availableExtraParams()
			if len(params) == 0 {
				m.statusMessage = "No extra params defined. Add 'extraParams' to ~/.config/mcpcli/config.yaml"
			} else {
				idx := m.globalParamCycle % len(params)
				p := params[idx]
				m.globalExtraParams[p.Key] = !m.globalExtraParams[p.Key]
				m.globalParamCycle++
				if m.globalExtraParams[p.Key] {
					m.statusMessage = fmt.Sprintf("Global: %s = %v ON (all MCPs)", p.Key, p.Value)
				} else {
					m.statusMessage = fmt.Sprintf("Global: %s OFF", p.Key)
				}
			}
			return m, nil
		}
	}

	return m, nil
}

// availableExtraParams returns the extra params defined in global config, or empty slice.
func (m *Model) availableExtraParams() []ExtraParam {
	if m.globalConfig == nil {
		return nil
	}
	return m.globalConfig.ExtraParams
}
