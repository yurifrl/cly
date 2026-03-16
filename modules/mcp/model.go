package mcp

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
)

// Model represents the TUI application state (Bubble Tea MVU pattern)
type Model struct {
	catalog       *Catalog
	globalConfig  *GlobalConfig
	availableMCPs []MCP
	installedMCPs map[string]bool

	displayItems  []ListItem
	filteredItems []ListItem

	context       Context
	contextSource string
	adapter       Adapter

	searchInput textinput.Model

	searchQuery         string
	selectedTags        []string
	selectedFile        string
	cursor              int
	scrollOffset        int
	viewportHeight      int
	searchFocused       bool
	showHelp            bool
	helpScrollOffset    int
	showContextSwitcher bool
	contextMenuCursor   int
	checkedMCPs         map[string]bool
	expandedPresets     map[string]bool
	expandedTags        map[string]bool
	hiddenSections      map[string]bool

	validationResult *ValidationResult
	statusMessage    string
	exitAfterApply   bool
	err              error
	quitting         bool

	// Extra params state
	mcpExtraParams    map[string]map[string]bool // [mcpName][paramKey] → active (per-MCP)
	globalExtraParams map[string]bool            // [paramKey] → active (applied to all MCPs)

	// Extra params modal (shown over the list when showExtraModal is true)
	showExtraModal    bool
	extraModalIsGlobal bool // true = modal is editing global params, false = per-MCP
	extraModal        ExtraParamsModal
}

// NewModel creates a new TUI model with initial state
func NewModel(catalog *Catalog, globalCfg *GlobalConfig, ctx Context, contextSource string, adapter Adapter) Model {
	mcps := catalog.GetAll()

	ti := textinput.New()
	ti.Placeholder = "Search MCPs..."
	ti.CharLimit = 100

	installedMCPs := make(map[string]bool)
	checkedMCPs := make(map[string]bool)
	if toolCfg, err := adapter.ReadConfig(ctx.Scope); err == nil {
		for name := range toolCfg.MCPServers {
			installedMCPs[name] = true
			checkedMCPs[name] = true
		}
	}

	hiddenSections := make(map[string]bool)
	if globalCfg != nil {
		for _, section := range globalCfg.UI.HiddenSections {
			hiddenSections[section] = true
		}
	}

	validationResult := Validate(catalog, globalCfg, adapter, ctx.Scope)
	displayItems := buildDisplayItemsWithValidation(mcps, globalCfg, hiddenSections, validationResult)

	return Model{
		catalog:           catalog,
		globalConfig:      globalCfg,
		availableMCPs:     mcps,
		installedMCPs:     installedMCPs,
		checkedMCPs:       checkedMCPs,
		displayItems:      displayItems,
		filteredItems:     displayItems,
		expandedPresets:   make(map[string]bool),
		expandedTags:      make(map[string]bool),
		hiddenSections:    hiddenSections,
		validationResult:  validationResult,
		context:           ctx,
		contextSource:     contextSource,
		adapter:           adapter,
		searchInput:       ti,
		cursor:            0,
		scrollOffset:      0,
		viewportHeight:    20,
		mcpExtraParams:     make(map[string]map[string]bool),
		globalExtraParams:  make(map[string]bool),
	}
}

func buildDisplayItemsWithValidation(mcps []MCP, globalCfg *GlobalConfig, collapsed map[string]bool, validationResult *ValidationResult) []ListItem {
	var items []ListItem

	if validationResult != nil && validationResult.HasIssues() {
		if collapsed["validation"] {
			items = append(items, ListItem{
				Type: ListItemSeparator,
				Name: "Validation (collapsed)",
			})
		} else {
			errorCount := validationResult.ErrorCount()
			warningCount := validationResult.WarningCount()
			header := "Validation"
			if errorCount > 0 && warningCount > 0 {
				header = fmt.Sprintf("Validation (%d errors, %d warnings)", errorCount, warningCount)
			} else if errorCount > 0 {
				header = fmt.Sprintf("Validation (%d errors)", errorCount)
			} else if warningCount > 0 {
				header = fmt.Sprintf("Validation (%d warnings)", warningCount)
			}
			items = append(items, ListItem{
				Type: ListItemSeparator,
				Name: header,
			})
			for i := range validationResult.Issues {
				issue := &validationResult.Issues[i]
				items = append(items, ListItem{
					Type:            ListItemValidation,
					Name:            issue.Source,
					ValidationIssue: issue,
				})
			}
		}
	}

	items = append(items, buildDisplayItemsWithHidden(mcps, globalCfg, collapsed)...)
	return items
}

func buildDisplayItemsWithHidden(mcps []MCP, globalCfg *GlobalConfig, collapsed map[string]bool) []ListItem {
	var items []ListItem

	if globalCfg != nil && len(globalCfg.Presets) > 0 {
		if collapsed["presets"] {
			items = append(items, ListItem{
				Type: ListItemSeparator,
				Name: "Presets (collapsed)",
			})
		} else {
			items = append(items, ListItem{
				Type: ListItemSeparator,
				Name: "Presets",
			})
			for presetName, mcpNames := range globalCfg.Presets {
				items = append(items, ListItem{
					Type:     ListItemPreset,
					Name:     presetName,
					MCPNames: mcpNames,
				})
			}
		}
	}

	tagMap := make(map[string][]string)
	for _, mcp := range mcps {
		for _, tag := range mcp.Tags {
			tagMap[tag] = append(tagMap[tag], mcp.Name)
		}
	}

	if len(tagMap) > 0 {
		if collapsed["tags"] {
			items = append(items, ListItem{
				Type: ListItemSeparator,
				Name: "Tags (collapsed)",
			})
		} else {
			items = append(items, ListItem{
				Type: ListItemSeparator,
				Name: "Tags",
			})
			for tagName, mcpNames := range tagMap {
				items = append(items, ListItem{
					Type:     ListItemTag,
					Name:     tagName,
					MCPNames: mcpNames,
				})
			}
		}
	}

	if len(mcps) > 0 {
		fileMap := make(map[string][]MCP)
		var fileOrder []string
		for i := range mcps {
			sourceFile := mcps[i].SourceFile
			if len(fileMap[sourceFile]) == 0 {
				fileOrder = append(fileOrder, sourceFile)
			}
			fileMap[sourceFile] = append(fileMap[sourceFile], mcps[i])
		}

		for _, sourceFile := range fileOrder {
			fileMCPs := fileMap[sourceFile]
			sectionKey := "mcps:" + sourceFile

			if collapsed[sectionKey] {
				items = append(items, ListItem{
					Type: ListItemSeparator,
					Name: sourceFile + " (collapsed)",
				})
			} else {
				items = append(items, ListItem{
					Type: ListItemSeparator,
					Name: sourceFile,
				})
				for i := range fileMCPs {
					items = append(items, ListItem{
						Type: ListItemMCP,
						Name: fileMCPs[i].Name,
						MCP:  &fileMCPs[i],
					})
				}
			}
		}
	}

	return items
}

func (m *Model) buildFilteredDisplayItems(filteredMCPs []MCP, query string) []ListItem {
	var items []ListItem
	queryLower := strings.ToLower(query)

	if query != "" {
		if !m.hiddenSections["presets"] && m.globalConfig != nil {
			for presetName, mcpNames := range m.globalConfig.Presets {
				if strings.Contains(strings.ToLower(presetName), queryLower) {
					items = append(items, ListItem{
						Type:     ListItemPreset,
						Name:     presetName,
						MCPNames: mcpNames,
					})
				}
			}
		}

		if !m.hiddenSections["tags"] {
			tagMap := make(map[string][]string)
			for _, mcp := range m.availableMCPs {
				for _, tag := range mcp.Tags {
					tagMap[tag] = append(tagMap[tag], mcp.Name)
				}
			}
			for tagName, mcpNames := range tagMap {
				if strings.Contains(strings.ToLower(tagName), queryLower) {
					items = append(items, ListItem{
						Type:     ListItemTag,
						Name:     tagName,
						MCPNames: mcpNames,
					})
				}
			}
		}

		if len(items) > 0 {
			items = append(items, ListItem{
				Type: ListItemSeparator,
				Name: "MCPs",
			})
		}
	}

	if !m.hiddenSections["mcps"] {
		for i := range filteredMCPs {
			items = append(items, ListItem{
				Type: ListItemMCP,
				Name: filteredMCPs[i].Name,
				MCP:  &filteredMCPs[i],
			})
		}
	}

	return items
}

func (m *Model) rebuildFilteredItems() []ListItem {
	var items []ListItem

	for _, item := range m.displayItems {
		items = append(items, item)

		if item.Type == ListItemPreset && m.expandedPresets[item.Name] {
			for _, mcpName := range item.MCPNames {
				if mcp, ok := m.catalog.Get(mcpName); ok {
					items = append(items, ListItem{
						Type: ListItemMCP,
						Name: "  " + mcp.Name,
						MCP:  &mcp,
					})
				}
			}
		}

		if item.Type == ListItemTag && m.expandedTags[item.Name] {
			for _, mcpName := range item.MCPNames {
				if mcp, ok := m.catalog.Get(mcpName); ok {
					items = append(items, ListItem{
						Type: ListItemMCP,
						Name: "  " + mcp.Name,
						MCP:  &mcp,
					})
				}
			}
		}
	}

	return items
}

func (m *Model) rebuildDisplayItems() {
	m.displayItems = buildDisplayItemsWithValidation(m.availableMCPs, m.globalConfig, m.hiddenSections, m.validationResult)
	m.filteredItems = m.rebuildFilteredItems()
	if m.cursor >= len(m.filteredItems) {
		m.cursor = 0
	}
	m.scrollOffset = 0
}

func (m *Model) runValidation() {
	m.validationResult = Validate(m.catalog, m.globalConfig, m.adapter, m.context.Scope)
	m.rebuildDisplayItems()
}

func (m *Model) findSectionStarts() []int {
	var starts []int

	for i, item := range m.filteredItems {
		if item.Type == ListItemSeparator {
			for j := i + 1; j < len(m.filteredItems); j++ {
				if m.filteredItems[j].Type != ListItemSeparator {
					starts = append(starts, j)
					break
				}
			}
		}
	}

	return starts
}
