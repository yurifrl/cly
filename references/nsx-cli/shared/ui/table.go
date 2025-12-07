package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"

	"github.com/NSXBet/nsx-cli/shared/style"
)

var defaultCustomOptions = []CustomOption{
	{Key: "↑/↓", Title: "navigate"},
	{Key: "Enter", Title: "select"},
	{Key: "q", Title: "quit"},
}

// CustomOption represents a custom keyboard shortcut with an action
type CustomOption struct {
	Key      string
	Title    string
	Function func(selectedRow interface{}) tea.Cmd
}

func (o CustomOption) String() string {
	key := style.CommandKey.Render(fmt.Sprintf("[%s]", o.Key))
	return key + style.Dimmed.Render(": "+o.Title)
}

// TableOption represents configuration options for the table
type TableOption[T any] struct {
	Title         string
	Columns       []table.Column
	Data          []T
	RowFunc       func(T) table.Row
	SelectionFunc func(T) tea.Cmd
	CustomOptions []CustomOption
	Footer        string
	EnableFilter  bool
	Height        int
	Width         int
}

// Table represents a generic table component
type Table[T any] struct {
	model         table.Model
	data          []T
	rowFunc       func(T) table.Row
	selectionFunc func(T) tea.Cmd
	customOptions []CustomOption
	title         string
	footer        string
	enableFilter  bool
	height        int
	width         int
}

// NewTable creates a new table with the given options
func NewTable[T any](options TableOption[T]) *Table[T] {
	baseStyle := lipgloss.NewStyle().
		Align(lipgloss.Left).
		BorderForeground(style.GetBorderColor())
	// Create the bubble table model
	t := table.New(options.Columns).
		WithRows(convertToRows(options.Data, options.RowFunc)).
		Focused(true).
		WithPageSize(25).
		WithBaseStyle(baseStyle).
		HighlightStyle(style.HighlightBlock)

	// Set width if provided
	if options.Width > 0 {
		t = t.WithTargetWidth(options.Width)
	}

	// Set static footer if provided
	if options.Footer != "" {
		t = t.WithStaticFooter(style.Dimmed.Render(options.Footer))
	}

	return &Table[T]{
		model:         t,
		data:          options.Data,
		rowFunc:       options.RowFunc,
		selectionFunc: options.SelectionFunc,
		customOptions: options.CustomOptions,
		title:         options.Title,
		footer:        options.Footer,
		enableFilter:  options.EnableFilter,
		height:        options.Height,
		width:         options.Width,
	}
}

// Init initializes the table component
func (t *Table[T]) Init() tea.Cmd {
	return nil
}

// Update handles user input and updates the table state
func (t *Table[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Update table dimensions based on window size
		if t.height == 0 {
			t.height = msg.Height - 4 // Reserve space for title and footer
		}
		if t.width == 0 {
			t.width = msg.Width
			// Apply the width to the bubble-table model
			t.model = t.model.WithTargetWidth(t.width)
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return t, tea.Quit

		case "enter":
			// Handle selection if a selection function is provided
			if t.selectionFunc != nil {
				selectedRow := t.getSelectedRow()
				if selectedRow != nil {
					return t, t.selectionFunc(*selectedRow)
				}
			}

		default:
			// Check for custom key options
			for _, option := range t.customOptions {
				if msg.String() == option.Key {
					selectedRow := t.getSelectedRow()
					if selectedRow != nil {
						return t, option.Function(*selectedRow)
					}
				}
			}
		}
	}

	// Update the bubble table model
	t.model, cmd = t.model.Update(msg)
	return t, cmd
}

// View renders the table component
func (t *Table[T]) View() string {
	var footer string
	customOptions := append(defaultCustomOptions, t.customOptions...)

	var options []string
	for _, option := range customOptions {
		options = append(options, option.String())
	}
	footer = lipgloss.JoinHorizontal(
		lipgloss.Left,
		"Use ",
		strings.Join(options, style.Dimmed.Render(" • ")),
	)

	var content strings.Builder

	if len(t.data) == 0 {
		return lipgloss.Place(t.width, t.height, lipgloss.Center, lipgloss.Center, "No Results (Press q to quit)")
	}

	// Add title if provided
	if t.title != "" {
		titleStyle := style.Title.
			Align(lipgloss.Center).
			MarginBottom(1)
		content.WriteString(titleStyle.Render(t.title))
		content.WriteString("\n")
	}

	// Add the table
	content.WriteString(t.model.View())

	if footer != "" {
		t.model = t.model.WithStaticFooter(style.Dimmed.Render(footer))
	}

	return content.String()
}

// getSelectedRow returns the currently selected row data
func (t *Table[T]) getSelectedRow() *T {
	selectedRow := t.model.HighlightedRow()
	if selectedRow.Data == nil {
		return nil
	}

	// Find the corresponding data item
	for i, item := range t.data {
		row := t.rowFunc(item)
		if compareRows(row, selectedRow) {
			return &t.data[i]
		}
	}

	return nil
}

// convertToRows converts data items to table rows
func convertToRows[T any](data []T, rowFunc func(T) table.Row) []table.Row {
	rows := make([]table.Row, len(data))
	for i, item := range data {
		rows[i] = rowFunc(item)
	}
	return rows
}

// compareRows compares two table rows for equality
func compareRows(row1, row2 table.Row) bool {
	data1 := row1.Data
	data2 := row2.Data

	if data1 == nil && data2 == nil {
		return true
	}

	if data1 == nil || data2 == nil {
		return false
	}

	if len(data1) != len(data2) {
		return false
	}

	for key, value1 := range data1 {
		value2, exists := data2[key]
		if !exists || value1 != value2 {
			return false
		}
	}

	return true
}

// UpdateData updates the table data and refreshes the display
func (t *Table[T]) UpdateData(newData []T) {
	t.data = newData
	rows := convertToRows(newData, t.rowFunc)
	t.model = t.model.WithRows(rows)
}

// SetTitle updates the table title
func (t *Table[T]) SetTitle(title string) {
	t.title = title
}

// SetFooter updates the table footer
func (t *Table[T]) SetFooter(footer string) {
	t.footer = footer
	t.model = t.model.WithStaticFooter(style.Dimmed.Render(footer))
}

// GetSelectedIndex returns the index of the currently selected row
func (t *Table[T]) GetSelectedIndex() int {
	selectedRow := t.model.HighlightedRow()
	if selectedRow.Data == nil {
		return -1
	}

	// Find the corresponding data item index
	for i, item := range t.data {
		row := t.rowFunc(item)
		if compareRows(row, selectedRow) {
			return i
		}
	}

	return -1
}

// GetData returns the current table data
func (t *Table[T]) GetData() []T {
	return t.data
}

// SetHeight sets the table height
func (t *Table[T]) SetHeight(height int) {
	t.height = height
}

// SetWidth sets the table width
func (t *Table[T]) SetWidth(width int) {
	t.width = width
	// Apply the width to the bubble-table model
	t.model = t.model.WithTargetWidth(width)
}

// SetFullWidth makes the table use the full available width
func (t *Table[T]) SetFullWidth() {
	t.width = 0 // Reset to auto-size on next window size message
}

// SortByDesc sorts the table by the specified column in descending order
func (t *Table[T]) SortByDesc(columnKey string) {
	t.model = t.model.SortByDesc(columnKey)
}

// SortByAsc sorts the table by the specified column in ascending order
func (t *Table[T]) SortByAsc(columnKey string) {
	t.model = t.model.SortByAsc(columnKey)
}

// RunTable is a convenience function to run a table as a standalone program
func RunTable[T any](options TableOption[T]) error {
	tableModel := NewTable(options)
	program := tea.NewProgram(tableModel, tea.WithAltScreen())
	_, err := program.Run()
	return err
}
