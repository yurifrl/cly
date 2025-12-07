package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/evertras/bubble-table/table"
)

// Example demonstrates how to use the Table component
func ExampleTable() {
	// Example data structure
	type User struct {
		ID    int
		Name  string
		Email string
		Age   int
	}

	// Sample data
	users := []User{
		{ID: 1, Name: "Alice Johnson", Email: "alice@example.com", Age: 30},
		{ID: 2, Name: "Bob Smith", Email: "bob@example.com", Age: 25},
		{ID: 3, Name: "Charlie Brown", Email: "charlie@example.com", Age: 35},
		{ID: 4, Name: "Diana Prince", Email: "diana@example.com", Age: 28},
	}

	// Define columns
	columns := []table.Column{
		table.NewColumn("id", "ID", 5),
		table.NewFlexColumn("name", "Name", 2),
		table.NewFlexColumn("email", "Email", 3),
		table.NewColumn("age", "Age", 5),
	}

	// Row function to convert data to table rows
	rowFunc := func(user User) table.Row {
		return table.NewRow(table.RowData{
			"id":    fmt.Sprintf("%d", user.ID),
			"name":  user.Name,
			"email": user.Email,
			"age":   fmt.Sprintf("%d", user.Age),
		})
	}

	// Selection function (optional)
	selectionFunc := func(user User) tea.Cmd {
		fmt.Printf("Selected user: %s (ID: %d)\n", user.Name, user.ID)
		return tea.Quit
	}

	// Custom options (optional)
	customOptions := []CustomOption{
		{
			Key:   "d",
			Title: "delete",
			Function: func(selectedRow interface{}) tea.Cmd {
				if user, ok := selectedRow.(User); ok {
					fmt.Printf("Deleting user: %s\n", user.Name)
				}
				return nil
			},
		},
		{
			Key:   "e",
			Title: "edit",
			Function: func(selectedRow interface{}) tea.Cmd {
				if user, ok := selectedRow.(User); ok {
					fmt.Printf("Editing user: %s\n", user.Name)
				}
				return nil
			},
		},
	}

	// Create table options
	options := TableOption[User]{
		Title:         "User Management",
		Columns:       columns,
		Data:          users,
		RowFunc:       rowFunc,
		SelectionFunc: selectionFunc,
		CustomOptions: customOptions,
		Footer:        "Use ↑/↓ to navigate • Enter to select • d to delete • e to edit • q to quit",
		EnableFilter:  true,
	}

	// Run the table (this would be used in a real application)
	// ui.RunTable(options)

	// For example purposes, just create the table
	tableModel := NewTable(options)
	fmt.Printf("Table created with %d rows\n", len(tableModel.GetData()))

	// Output: Table created with 4 rows
}

// Example of programmatic table updates
func ExampleTable_UpdateData() {
	type Item struct {
		ID   int
		Name string
	}

	// Initial data
	initialData := []Item{
		{ID: 1, Name: "Item 1"},
		{ID: 2, Name: "Item 2"},
	}

	// Create table
	options := TableOption[Item]{
		Title:   "Items",
		Columns: []table.Column{table.NewColumn("id", "ID", 5)},
		Data:    initialData,
		RowFunc: func(item Item) table.Row {
			return table.NewRow(table.RowData{
				"id": fmt.Sprintf("%d", item.ID),
			})
		},
	}

	tableModel := NewTable(options)
	fmt.Printf("Initial data: %d items\n", len(tableModel.GetData()))

	// Update data
	newData := []Item{
		{ID: 1, Name: "Item 1"},
		{ID: 2, Name: "Item 2"},
		{ID: 3, Name: "Item 3"},
	}

	tableModel.UpdateData(newData)
	fmt.Printf("Updated data: %d items\n", len(tableModel.GetData()))

	// Output: Initial data: 2 items
	// Updated data: 3 items
}
