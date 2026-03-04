package lg_table_mindy

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

// swatchCols is the number of color swatches per row (each takes 2 table columns: label + swatch).
const swatchCols = 6

// renderTable builds a 256-color ANSI swatch display organized into logical groups,
// separated by visual gaps. Each table row contains 6 pairs of (number label, colored block).
//
// Layout:
//   - Standard colors 0-5, gap row, 6-7 (padded)
//   - Gap row
//   - Bright colors 8-13, gap row, 14-15 (padded)
//   - Gap row
//   - 216-color cube 16-231 in rows of 6
//   - Gap row
//   - Grayscale ramp 232-255 in rows of 6
func renderTable() string {
	// Build the color groups
	var rows [][]string

	// --- Standard colors 0-7 ---
	rows = append(rows, buildSwatchRow(0, 6))   // 0-5
	rows = append(rows, buildSwatchRow(6, 2))   // 6-7 (padded to 6 wide)
	rows = append(rows, buildGapRow())           // visual separator

	// --- Bright colors 8-15 ---
	rows = append(rows, buildSwatchRow(8, 6))   // 8-13
	rows = append(rows, buildSwatchRow(14, 2))  // 14-15 (padded to 6 wide)
	rows = append(rows, buildGapRow())           // visual separator

	// --- 216-color cube (16-231) in rows of 6 ---
	for i := 16; i <= 231; i += swatchCols {
		n := swatchCols
		if i+n > 232 {
			n = 232 - i
		}
		rows = append(rows, buildSwatchRow(i, n))
	}
	rows = append(rows, buildGapRow()) // visual separator

	// --- Grayscale ramp (232-255) in rows of 6 ---
	for i := 232; i <= 255; i += swatchCols {
		n := swatchCols
		if i+n > 256 {
			n = 256 - i
		}
		rows = append(rows, buildSwatchRow(i, n))
	}

	// Table has 12 columns: 6 × (label, swatch)
	tableCols := swatchCols * 2

	t := table.New().
		Rows(rows...).
		Border(lipgloss.HiddenBorder()).
		BorderColumn(false).
		BorderRow(false).
		BorderTop(false).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		StyleFunc(func(row, col int) lipgloss.Style {
			// Gap rows get minimal styling
			if row >= 0 && row < len(rows) && len(rows[row]) > 0 && rows[row][0] == "" {
				return lipgloss.NewStyle().Width(1).Height(0)
			}

			isLabel := col%2 == 0
			if isLabel {
				// Label column: right-aligned number
				return lipgloss.NewStyle().
					Foreground(lipgloss.Color("245")).
					Align(lipgloss.Right).
					Width(4).
					PaddingRight(1)
			}

			// Swatch column: figure out which color index this cell represents
			pairIdx := col / 2
			colorIdx := colorIndexFromRow(rows, row, pairIdx)
			if colorIdx < 0 || colorIdx > 255 {
				return lipgloss.NewStyle().Width(5)
			}

			bg := lipgloss.Color(fmt.Sprintf("%d", colorIdx))
			fg := lipgloss.Color("255")
			if isLightColor(colorIdx) {
				fg = lipgloss.Color("0")
			}

			return lipgloss.NewStyle().
				Background(bg).
				Foreground(fg).
				Align(lipgloss.Center).
				Width(5)
		}).
		Width(tableCols * 5)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("213")).
		MarginBottom(1)

	return titleStyle.Render("🎨 256 ANSI Color Swatches") + "\n" + t.Render()
}

// buildSwatchRow creates a row of n swatches starting at color index `start`.
// Each swatch occupies 2 columns: [label, block]. Remaining columns are empty.
func buildSwatchRow(start, count int) []string {
	row := make([]string, swatchCols*2)
	for i := 0; i < swatchCols; i++ {
		if i < count {
			c := start + i
			row[i*2] = fmt.Sprintf("%d", c)
			row[i*2+1] = fmt.Sprintf("%3d", c) // visible number inside swatch
		} else {
			row[i*2] = ""
			row[i*2+1] = ""
		}
	}
	return row
}

// buildGapRow creates an empty row used as a visual separator between color groups.
func buildGapRow() []string {
	row := make([]string, swatchCols*2)
	// All empty strings — the StyleFunc detects this via rows[row][0] == ""
	return row
}

// colorIndexFromRow extracts the color index for a given row and pair index
// by parsing the label column content.
func colorIndexFromRow(rows [][]string, row, pairIdx int) int {
	if row < 0 || row >= len(rows) {
		return -1
	}
	labelCol := pairIdx * 2
	if labelCol >= len(rows[row]) {
		return -1
	}
	label := strings.TrimSpace(rows[row][labelCol])
	if label == "" {
		return -1
	}
	var idx int
	_, err := fmt.Sscanf(label, "%d", &idx)
	if err != nil {
		return -1
	}
	return idx
}

// isLightColor returns true if the ANSI 256 color index is perceptually light.
func isLightColor(c int) bool {
	// Standard colors 0-15: light if in the bright range or specific light colors
	if c <= 15 {
		switch c {
		case 7, 9, 10, 11, 12, 13, 14, 15:
			return true
		default:
			return false
		}
	}
	// 216-color cube (16-231): calculate approximate luminance from RGB components.
	if c >= 16 && c <= 231 {
		idx := c - 16
		r := idx / 36
		g := (idx % 36) / 6
		b := idx % 6
		// Each component is 0-5; rough luminance threshold
		lum := 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
		return lum > 2.5
	}
	// Grayscale ramp (232-255): 232 is dark, 255 is light.
	if c >= 232 {
		return c >= 244
	}
	return false
}
