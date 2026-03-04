package lg_list_glow

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/list"
)

// glowItem represents a multi-line list entry with a title and description.
type glowItem struct {
	title       string
	description string
}

// items holds the document entries displayed in the list.
var items = []glowItem{
	{
		title:       "Getting Started",
		description: "Install the CLI and run your first command",
	},
	{
		title:       "Configuration",
		description: "Customize settings, themes, and keybindings",
	},
	{
		title:       "Workflows",
		description: "Automate repetitive tasks with pipelines",
	},
	{
		title:       "Deployment",
		description: "Ship to production with zero downtime",
	},
	{
		title:       "Troubleshooting",
		description: "Common issues and how to resolve them",
	},
}

// selectedIndex is the index of the "selected" item to highlight.
const selectedIndex = 2

// renderGlowList builds and returns the rendered list with selection glow.
func renderGlowList() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63")).
		MarginBottom(1)

	selectedBg := lipgloss.Color("57")   // purple background for glow
	selectedFg := lipgloss.Color("230")  // warm white text
	normalFg := lipgloss.Color("252")    // light gray text
	dimFg := lipgloss.Color("243")       // dim gray for descriptions
	accentColor := lipgloss.Color("63")  // accent purple for selected enumerator

	// Custom enumerator: "│" for selected item, " " (space) for others.
	enumerator := func(_ list.Items, index int) string {
		if index == selectedIndex {
			return "│"
		}
		return " "
	}

	// Enumerator style: accent color for selected, dim for others.
	enumeratorStyleFunc := func(_ list.Items, index int) lipgloss.Style {
		if index == selectedIndex {
			return lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true).
				PaddingRight(1)
		}
		return lipgloss.NewStyle().
			Foreground(dimFg).
			PaddingRight(1)
	}

	// Item style: highlighted background for selected, normal for others.
	itemStyleFunc := func(_ list.Items, index int) lipgloss.Style {
		if index == selectedIndex {
			return lipgloss.NewStyle().
				Foreground(selectedFg).
				Background(selectedBg).
				Bold(true).
				Padding(0, 1)
		}
		return lipgloss.NewStyle().
			Foreground(normalFg)
	}

	// Build multi-line item strings: title + newline + description.
	entries := make([]any, len(items))
	for i, item := range items {
		if i == selectedIndex {
			entries[i] = item.title + "\n" + item.description
		} else {
			// For non-selected items, dim the description line.
			entries[i] = item.title + "\n" +
				lipgloss.NewStyle().Foreground(dimFg).Render(item.description)
		}
	}

	l := list.New(entries...).
		Enumerator(enumerator).
		EnumeratorStyleFunc(enumeratorStyleFunc).
		ItemStyleFunc(itemStyleFunc)

	return titleStyle.Render("📄 Documentation") + "\n" + l.String()
}
