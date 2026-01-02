package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type command struct {
	name string
	desc string
}

var commands = []command{
	{"add", "Add worktree"},
	{"switch", "Switch worktree"},
	{"remove", "Remove worktree"},
	{"status", "Worktree status"},
	{"push", "Push worktree"},
	{"prune", "Prune stale"},
	{"delete", "Delete worktree"},
}

// --- Palette Model ---

type palette struct {
	input    textinput.Model
	filtered []command
	cursor   int
}

func newPalette() palette {
	ti := textinput.New()
	ti.Placeholder = "type to filter..."
	ti.Focus()
	ti.CharLimit = 30
	ti.Width = 30
	ti.Prompt = "> "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return palette{
		input:    ti,
		filtered: commands,
	}
}

func (p *palette) filter() {
	query := strings.ToLower(p.input.Value())
	p.filtered = nil
	for _, c := range commands {
		if query == "" || strings.Contains(c.name, query) || strings.Contains(strings.ToLower(c.desc), query) {
			p.filtered = append(p.filtered, c)
		}
	}
	if p.cursor >= len(p.filtered) {
		p.cursor = max(0, len(p.filtered)-1)
	}
}

func (p palette) selected() string {
	if len(p.filtered) > 0 {
		return p.filtered[p.cursor].name
	}
	return ""
}

// --- Main Model ---

type model struct {
	page        string // current page name
	paletteOpen bool
	palette     palette
	message     string // feedback message
}

func initialModel() model {
	return model{
		page:    "menu",
		palette: newPalette(),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global: ctrl+p toggles palette
		if msg.String() == "ctrl+p" {
			m.paletteOpen = !m.paletteOpen
			if m.paletteOpen {
				m.palette = newPalette()
				return m, textinput.Blink
			}
			return m, nil
		}

		// Palette is open - handle palette keys
		if m.paletteOpen {
			switch msg.String() {
			case "esc":
				m.paletteOpen = false
				return m, nil
			case "enter":
				if sel := m.palette.selected(); sel != "" {
					m.message = fmt.Sprintf("executed: %s", sel)
					m.paletteOpen = false
				}
				return m, nil
			case "down", "ctrl+n":
				if m.palette.cursor < len(m.palette.filtered)-1 {
					m.palette.cursor++
				}
				return m, nil
			case "up", "ctrl+k":
				if m.palette.cursor > 0 {
					m.palette.cursor--
				}
				return m, nil
			}

			// All other keys go to text input
			var cmd tea.Cmd
			m.palette.input, cmd = m.palette.input.Update(msg)
			m.palette.filter()
			return m, cmd
		}

		// Page keys (when palette closed)
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	return m, nil
}

var (
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1).
			Width(40)
	paletteStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")).
			Padding(0, 1).
			Width(36)
	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)
	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62"))
)

func (m model) View() string {
	var b strings.Builder

	// Page content
	b.WriteString(headerStyle.Render("Worktree > Menu"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("──────────────────────────────────"))
	b.WriteString("\n\n")
	b.WriteString("  This is a placeholder page.\n")
	b.WriteString("  Press ctrl+p for command palette.\n")
	if m.message != "" {
		b.WriteString(fmt.Sprintf("\n  %s\n", selectedStyle.Render(m.message)))
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("ctrl+p palette • q quit"))

	page := boxStyle.Render(b.String())

	// Overlay palette if open
	if m.paletteOpen {
		var pb strings.Builder
		pb.WriteString(m.palette.input.View())
		pb.WriteString("\n\n")

		for i, cmd := range m.palette.filtered {
			cursor := "  "
			style := dimStyle
			if i == m.palette.cursor {
				cursor = "> "
				style = selectedStyle
			}
			pb.WriteString(fmt.Sprintf("%s%-10s %s\n", cursor, style.Render(cmd.name), dimStyle.Render(cmd.desc)))
		}

		if len(m.palette.filtered) == 0 {
			pb.WriteString(dimStyle.Render("  no matches\n"))
		}

		pb.WriteString("\n")
		pb.WriteString(helpStyle.Render("↑/↓ nav • enter select • esc close"))

		paletteBox := paletteStyle.Render(pb.String())

		// Stack: page then palette
		return page + "\n\n" + paletteBox
	}

	return page
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
