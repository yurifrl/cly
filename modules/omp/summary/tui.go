package ompsummary

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/yurifrl/cly/pkg/cmux"
)

// Wake is sent whenever the engine or the summarizer mutated something the
// TUI should re-render.
type Wake struct{}

var (
	styleHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	styleBadge  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	styleDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	styleFocus  = lipgloss.NewStyle().Bold(true)
	styleOK     = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styleRun    = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	stylePend   = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styleErr    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
)

// hit is the row range (terminal y rows, inclusive) a target occupies, rebuilt
// on every render so mouse clicks map back to targets.
type hit struct {
	y0, y1 int
	idx    int
}

// Model is the sidebar TUI over the summary engine.
type Model struct {
	eng    *Engine
	sum    *Summarizer
	width  int
	height int
	hits   []hit
	status string // transient feedback ("copied"), cleared on next wake
}

// NewModel builds the sidebar model.
func NewModel(eng *Engine, sum *Summarizer) Model {
	return Model{eng: eng, sum: sum}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case Wake:
		m.status = ""
		return m, nil

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}

	case tea.MouseClickMsg:
		mouse := msg.Mouse()
		targets, summaries := m.eng.Snapshot()
		idx := hitRow(m.hits, mouse.Y)
		if idx < 0 || idx >= len(targets) {
			return m, nil
		}
		t := targets[idx]
		switch mouse.Button {
		case tea.MouseLeft:
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			go func() {
				defer cancel()
				_ = cmux.FocusPane(ctx, t.PaneRef)
			}()
			m.status = "focused " + t.Title
		case tea.MouseRight:
			if s, ok := summaries[t.SessionID]; ok && s.Text != "" {
				m.status = "copied"
				return m, tea.SetClipboard(s.Text)
			}
			m.status = "nothing to copy"
		}
		return m, nil
	}
	return m, nil
}

// View implements tea.Model. Inline (no alt screen): the sidebar pane IS the
// viewport. Mouse mode is enabled so clicks map to rows.
func (m Model) View() tea.View {
	targets, summaries := m.eng.Snapshot()
	m.hits = m.hits[:0]

	var b strings.Builder
	w := m.width
	if w <= 0 {
		w = 50
	}

	// Header: title left, summarizer badge right.
	left := styleHeader.Render("omp summary")
	right := styleBadge.Render(m.sum.Provider + "/" + m.sum.Model)
	if !m.sum.HasKey {
		right = styleErr.Render("no ai key")
	}
	b.WriteString(joinLeftRight(left, right, w))
	b.WriteByte('\n')

	if len(targets) == 0 {
		b.WriteString(styleDim.Render("\n  no live omp sessions\n"))
	}

	wrap := lipgloss.NewStyle().Width(w - 3)
	for i, t := range targets {
		m.hits = append(m.hits, hit{y0: strings.Count(b.String(), "\n"), idx: i})

		marker := "  "
		if t.Focused {
			marker = styleFocus.Render("▸ ")
		}
		s := summaries[t.SessionID]
		glyph, glyphStyle := statusGlyph(s)
		head := fmt.Sprintf("%s%s %s%s", marker, glyphStyle.Render(glyph),
			truncate(t.Title, max(0, w-14)), styleBadge.Render(" "+relAge(t.UpdatedAt)))
		b.WriteString(head)
		b.WriteByte('\n')

		var text string
		switch {
		case s == nil:
			text = styleDim.Render("waiting…")
		case s.Status == "running":
			text = styleRun.Render("summarizing…")
		case s.Status == "pending":
			text = stylePend.Render("queued…")
		case s.Status == "error" && s.Error != "":
			text = styleErr.Render(truncate("⚠ "+s.Error, max(0, w-4)))
		case s.Text == "idle":
			text = styleDim.Render("idle")
		case s.Text != "":
			text = wrap.Render(s.Text)
			if lines := strings.Split(text, "\n"); len(lines) > 3 {
				text = strings.Join(lines[:3], "\n") + styleDim.Render(" …")
			}
		default:
			text = styleDim.Render("…")
		}
		b.WriteString(indent(text))
		m.hits[len(m.hits)-1].y1 = strings.Count(b.String(), "\n")
		b.WriteByte('\n')
	}

	if m.status != "" {
		b.WriteString(styleOK.Render("\n " + m.status + "\n"))
	}
	return tea.NewView(b.String())
}

// --- helpers -------------------------------------------------------------

// hitRow maps a terminal y to a target index; -1 when outside any row.
func hitRow(hits []hit, y int) int {
	for _, h := range hits {
		if y >= h.y0 && y <= h.y1 {
			return h.idx
		}
	}
	return -1
}

func statusGlyph(s *Summary) (string, lipgloss.Style) {
	switch {
	case s == nil:
		return "○", styleDim
	case s.Status == "running":
		return "◐", styleRun
	case s.Status == "pending":
		return "○", stylePend
	case s.Status == "error":
		return "⚠", styleErr
	default:
		return "●", styleOK
	}
}

func relAge(unix float64) string {
	d := time.Since(time.Unix(int64(unix), 0))
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	default:
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
}

// truncate hard-caps a plain (style-free) string; callers style the result.
func truncate(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	if n > 1 {
		return s[:n-1] + "…"
	}
	return "…"
}

// indent indents every line of a possibly multi-line block by two spaces.
func indent(block string) string {
	lines := strings.Split(strings.TrimRight(block, "\n"), "\n")
	for i, l := range lines {
		lines[i] = "  " + l
	}
	return strings.Join(lines, "\n")
}

// joinLeftRight places right at the end of a width-wide line, keeping only
// the left side when the two don't fit.
func joinLeftRight(left, right string, width int) string {
	r := lipgloss.Width(right)
	l := lipgloss.Width(left)
	if l+r+1 > width {
		return left
	}
	pad := strings.Repeat(" ", width-r-l)
	return left + pad + right
}
