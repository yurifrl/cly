package lg_table_chess

import (
	"fmt"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

// Chess pieces in Unicode for the starting position.
var boardRows = [][]string{
	{"♜", "♞", "♝", "♛", "♚", "♝", "♞", "♜"},
	{"♟", "♟", "♟", "♟", "♟", "♟", "♟", "♟"},
	{" ", " ", " ", " ", " ", " ", " ", " "},
	{" ", " ", " ", " ", " ", " ", " ", " "},
	{" ", " ", " ", " ", " ", " ", " ", " "},
	{" ", " ", " ", " ", " ", " ", " ", " "},
	{"♙", "♙", "♙", "♙", "♙", "♙", "♙", "♙"},
	{"♖", "♘", "♗", "♕", "♔", "♗", "♘", "♖"},
}

// renderTable builds and returns a chess board with file/rank labels.
func renderTable() string {
	lightSquare := lipgloss.Color("229") // warm cream
	darkSquare := lipgloss.Color("94")   // brown
	lightPiece := lipgloss.Color("255")  // white pieces
	darkPiece := lipgloss.Color("232")   // black pieces

	t := table.New().
		Rows(boardRows...).
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("237"))).
		BorderRow(true).
		BorderColumn(true).
		StyleFunc(func(row, col int) lipgloss.Style {
			s := lipgloss.NewStyle().
				Align(lipgloss.Center).
				Width(3)

			// Alternate square colors like a real chess board.
			isLight := (row+col)%2 == 0
			if isLight {
				s = s.Background(lightSquare)
			} else {
				s = s.Background(darkSquare)
			}

			// Color pieces: rows 0-1 are black, rows 6-7 are white.
			if row <= 1 {
				s = s.Foreground(darkPiece).Bold(true)
			} else if row >= 6 {
				s = s.Foreground(lightPiece).Bold(true)
			}

			return s
		})

	board := t.Render()

	// Build rank labels (8 down to 1) to the left of the board.
	// Count the rendered lines to compute per-row height.
	boardLines := splitLines(board)
	totalLines := len(boardLines)

	// With 8 rows and BorderRow(true), each data row has a content line
	// plus a border line. The table also has top and bottom borders.
	// Layout: top-border, (row-content, row-border) × 8, except last has bottom-border.
	// We'll compute row height dynamically.
	dataRows := 8
	// Subtract 1 for top border line; remaining lines are (content + separator) pairs
	// The last row ends with the bottom border instead of a separator.
	innerLines := totalLines - 1 // exclude top border
	rowHeight := innerLines / dataRows

	rankStyle := lipgloss.NewStyle().
		Width(2).
		Align(lipgloss.Right).
		Foreground(lipgloss.Color("245"))

	// Build rank column: one label per row, vertically centered in the row's height
	var rankLabels string
	// The top border occupies 1 line, so start with a blank to align
	rankLabels = rankStyle.Render(" ") + "\n"
	for i := 0; i < dataRows; i++ {
		rank := fmt.Sprintf("%d", 8-i)
		for j := 0; j < rowHeight; j++ {
			if j == 0 {
				rankLabels += rankStyle.Render(rank) + "\n"
			} else {
				rankLabels += rankStyle.Render(" ") + "\n"
			}
		}
	}

	// Build file labels (a-h) along the bottom
	files := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	fileStyle := lipgloss.NewStyle().
		Width(3).
		Align(lipgloss.Center).
		Foreground(lipgloss.Color("245"))

	// Build the file label row — need to account for border column characters
	// Use the same width as the cells with a 2-char left margin for rank column + border
	fileRow := "   " // offset for rank labels + left border
	for _, f := range files {
		fileRow += fileStyle.Render(f)
		fileRow += " " // border column separator
	}

	// Combine rank labels + board horizontally, then add file labels below
	withRanks := lipgloss.JoinHorizontal(lipgloss.Top, rankLabels, board)
	result := lipgloss.JoinVertical(lipgloss.Left, withRanks, fileRow)

	return result
}

// splitLines splits a string into lines.
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
