package lg_table_ansi

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

// renderTable builds and returns a 3-row table with a dim second column.
func renderTable() string {
	rows := [][]string{
		{"Red", "#FF0000", "Warm"},
		{"Green", "#00FF00", "Cool"},
		{"Blue", "#0000FF", "Cool"},
	}

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229")).
		Padding(0, 1).
		Align(lipgloss.Center)

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Padding(0, 1)

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Faint(true).
		Padding(0, 1)

	t := table.New().
		Headers("Color", "Hex", "Temp").
		Rows(rows...).
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("238"))).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			// Dim the second column (Hex values)
			if col == 1 {
				return dimStyle
			}
			return normalStyle
		}).
		Width(40)

	return t.Render()
}
