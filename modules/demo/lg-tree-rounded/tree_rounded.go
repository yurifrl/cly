package lg_tree_rounded

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/tree"
)

// renderRoundedTree creates a grocery list tree with rounded enumerator and dim style.
func renderRoundedTree() string {
	enumeratorStyle := lipgloss.NewStyle().
		Faint(true)

	t := tree.Root("🛒 Groceries").
		Child(
			tree.Root("Fruits").
				Child("Apples", "Bananas", "Oranges", "Strawberries"),
			tree.Root("Vegetables").
				Child("Carrots", "Broccoli", "Spinach"),
			tree.Root("Dairy").
				Child("Milk", "Cheese", "Yogurt"),
			tree.Root("Bakery").
				Child("Bread", "Croissants"),
		).
		Enumerator(tree.RoundedEnumerator).
		EnumeratorStyle(enumeratorStyle)

	return t.String()
}
