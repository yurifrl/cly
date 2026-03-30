package lg_table_languages

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

// renderTable builds and returns a table of language greetings.
func renderTable() string {
	purple := lipgloss.Color("99")
	dimPurple := lipgloss.Color("63")
	faintPurple := lipgloss.Color("60")

	// Row index 2 (0-based) is Arabic
	const arabicRow = 2

	t := table.New().
		Headers("LANGUAGE", "FORMAL", "INFORMAL").
		Row("Chinese", "您好", "你好").
		Row("Japanese", "こんにちは", "やあ").
		Row("Arabic", "السلام عليكم", "أهلين").
		Row("Russian", "Здравствуйте", "Привет").
		Row("Spanish", "¿Cómo está?", "¿Qué tal?").
		Row("English", "You alright?", "Heya").
		Border(lipgloss.ThickBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(faintPurple)).
		StyleFunc(func(row, col int) lipgloss.Style {
			s := lipgloss.NewStyle().Padding(0, 1)

			if row == table.HeaderRow {
				return s.
					Bold(true).
					Foreground(purple).
					Align(lipgloss.Center)
			}

			// Alternating row colors: even rows lighter, odd rows dimmer
			if row%2 == 0 {
				s = s.Foreground(lipgloss.Color("252"))
			} else {
				s = s.Foreground(lipgloss.Color("245"))
			}

			// Highlight the language name column
			if col == 0 {
				s = s.Foreground(dimPurple).Bold(true)
			}

			// Arabic row is right-aligned for RTL script
			if row == arabicRow {
				s = s.Align(lipgloss.Right)
			}

			return s
		})

	return t.Render()
}
