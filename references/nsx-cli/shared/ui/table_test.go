package ui

import (
	"testing"

	"github.com/evertras/bubble-table/table"
)

// Test data structure
type TestData struct {
	ID   int
	Name string
	Age  int
}

// Test row function
func testRowFunc(data TestData) table.Row {
	return table.NewRow(table.RowData{
		"id":   data.ID,
		"name": data.Name,
		"age":  data.Age,
	})
}

func TestNewTable(t *testing.T) {
	// Test data
	testData := []TestData{
		{ID: 1, Name: "Alice", Age: 30},
		{ID: 2, Name: "Bob", Age: 25},
		{ID: 3, Name: "Charlie", Age: 35},
	}

	// Create columns
	columns := []table.Column{
		table.NewColumn("id", "ID", 5),
		table.NewColumn("name", "Name", 15),
		table.NewColumn("age", "Age", 5),
	}

	// Create table options
	options := TableOption[TestData]{
		Title:   "Test Table",
		Columns: columns,
		Data:    testData,
		RowFunc: testRowFunc,
		Footer:  "Test Footer",
	}

	// Create table
	tableModel := NewTable(options)

	// Verify table creation
	if tableModel == nil {
		t.Fatal("NewTable returned nil")
	}

	if len(tableModel.data) != len(testData) {
		t.Errorf("Expected %d data items, got %d", len(testData), len(tableModel.data))
	}

	if tableModel.title != options.Title {
		t.Errorf("Expected title '%s', got '%s'", options.Title, tableModel.title)
	}

	if tableModel.footer != options.Footer {
		t.Errorf("Expected footer '%s', got '%s'", options.Footer, tableModel.footer)
	}
}

func TestUpdateData(t *testing.T) {
	// Initial data
	initialData := []TestData{
		{ID: 1, Name: "Alice", Age: 30},
	}

	// Create table
	options := TableOption[TestData]{
		Title:   "Test Table",
		Columns: []table.Column{table.NewColumn("id", "ID", 5)},
		Data:    initialData,
		RowFunc: testRowFunc,
	}

	tableModel := NewTable(options)

	// Update data
	newData := []TestData{
		{ID: 1, Name: "Alice", Age: 30},
		{ID: 2, Name: "Bob", Age: 25},
	}

	tableModel.UpdateData(newData)

	// Verify update
	if len(tableModel.data) != len(newData) {
		t.Errorf("Expected %d data items after update, got %d", len(newData), len(tableModel.data))
	}

	// Verify data content
	for i, expected := range newData {
		if tableModel.data[i].ID != expected.ID {
			t.Errorf("Expected ID %d at index %d, got %d", expected.ID, i, tableModel.data[i].ID)
		}
	}
}

func TestSetTitle(t *testing.T) {
	options := TableOption[TestData]{
		Title:   "Original Title",
		Columns: []table.Column{table.NewColumn("id", "ID", 5)},
		Data:    []TestData{},
		RowFunc: testRowFunc,
	}

	tableModel := NewTable(options)

	newTitle := "New Title"
	tableModel.SetTitle(newTitle)

	if tableModel.title != newTitle {
		t.Errorf("Expected title '%s', got '%s'", newTitle, tableModel.title)
	}
}

func TestSetFooter(t *testing.T) {
	options := TableOption[TestData]{
		Title:   "Test Table",
		Columns: []table.Column{table.NewColumn("id", "ID", 5)},
		Data:    []TestData{},
		RowFunc: testRowFunc,
		Footer:  "Original Footer",
	}

	tableModel := NewTable(options)

	newFooter := "New Footer"
	tableModel.SetFooter(newFooter)

	if tableModel.footer != newFooter {
		t.Errorf("Expected footer '%s', got '%s'", newFooter, tableModel.footer)
	}
}

func TestGetData(t *testing.T) {
	testData := []TestData{
		{ID: 1, Name: "Alice", Age: 30},
		{ID: 2, Name: "Bob", Age: 25},
	}

	options := TableOption[TestData]{
		Title:   "Test Table",
		Columns: []table.Column{table.NewColumn("id", "ID", 5)},
		Data:    testData,
		RowFunc: testRowFunc,
	}

	tableModel := NewTable(options)
	retrievedData := tableModel.GetData()

	if len(retrievedData) != len(testData) {
		t.Errorf("Expected %d data items, got %d", len(testData), len(retrievedData))
	}

	for i, expected := range testData {
		if retrievedData[i].ID != expected.ID {
			t.Errorf("Expected ID %d at index %d, got %d", expected.ID, i, retrievedData[i].ID)
		}
	}
}

func TestConvertToRows(t *testing.T) {
	testData := []TestData{
		{ID: 1, Name: "Alice", Age: 30},
		{ID: 2, Name: "Bob", Age: 25},
	}

	rows := convertToRows(testData, testRowFunc)

	if len(rows) != len(testData) {
		t.Errorf("Expected %d rows, got %d", len(testData), len(rows))
	}

	// Verify row data
	for i, data := range testData {
		rowData := rows[i].Data
		if rowData["id"] != data.ID {
			t.Errorf("Expected ID %d in row %d, got %v", data.ID, i, rowData["id"])
		}
		if rowData["name"] != data.Name {
			t.Errorf("Expected name '%s' in row %d, got %v", data.Name, i, rowData["name"])
		}
		if rowData["age"] != data.Age {
			t.Errorf("Expected age %d in row %d, got %v", data.Age, i, rowData["age"])
		}
	}
}
