package lg_list_grocery

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/list"
)

// groceryItem represents a single item in the grocery list.
type groceryItem struct {
	name      string
	purchased bool
}

// groceryList holds the items for building the list.
var groceries = []groceryItem{
	{name: "Bananas", purchased: true},
	{name: "Almond Milk", purchased: true},
	{name: "Eggs", purchased: false},
	{name: "Bread", purchased: true},
	{name: "Fish Cake", purchased: false},
	{name: "Leeks", purchased: false},
	{name: "Papaya", purchased: true},
	{name: "Cashews", purchased: false},
}

// renderGroceryList builds and returns the rendered grocery list string.
func renderGroceryList() string {
	// Styles
	purchasedColor := lipgloss.Color("28")  // green
	pendingColor := lipgloss.Color("212")   // pink
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99")).MarginBottom(1)

	// Custom enumerator: checkmark for purchased, bullet for pending.
	enumerator := func(_ list.Items, index int) string {
		if groceries[index].purchased {
			return "✓"
		}
		return "•"
	}

	// Enumerator style: green for purchased, pink for pending.
	enumeratorStyleFunc := func(_ list.Items, index int) lipgloss.Style {
		if groceries[index].purchased {
			return lipgloss.NewStyle().Foreground(purchasedColor).PaddingRight(1)
		}
		return lipgloss.NewStyle().Foreground(pendingColor).PaddingRight(1)
	}

	// Item style: strikethrough + dim for purchased, normal for pending.
	itemStyleFunc := func(_ list.Items, index int) lipgloss.Style {
		if groceries[index].purchased {
			return lipgloss.NewStyle().Strikethrough(true).Faint(true).Foreground(purchasedColor)
		}
		return lipgloss.NewStyle().Foreground(pendingColor)
	}

	// Build item name strings for the list.
	names := make([]any, len(groceries))
	for i, g := range groceries {
		names[i] = g.name
	}

	l := list.New(names...).
		Enumerator(enumerator).
		EnumeratorStyleFunc(enumeratorStyleFunc).
		ItemStyleFunc(itemStyleFunc)

	return titleStyle.Render("🛒 Grocery List") + "\n" + l.String()
}
