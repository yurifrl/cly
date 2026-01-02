package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type page int

const (
	pageMenu page = iota
	pageAdd
	pageSwitch
	pagePalette
)

type command struct {
	name string
	desc string
	page page
}

var commands = []command{
	{"add", "Add worktree", pageAdd},
	{"switch", "Switch worktree", pageSwitch},
	{"menu", "Back to menu", pageMenu},
}

// --- Model ---

type model struct {
	page     page
	prevPage page // for returning from palette

	// Palette state
	paletteInput    textinput.Model
	paletteFiltered []command
	paletteCursor   int

	// Menu state
	menuCursor int
	menuItems  []string

	message string
}

func newPaletteInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "type to filter..."
	ti.Focus()
	ti.CharLimit = 30
	ti.Width = 28
	ti.Prompt = "> "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return ti
}

func initialModel() model {
	return model{
		page:            pageMenu,
		paletteInput:    newPaletteInput(),
		paletteFiltered: commands,
		menuItems:       []string{"Add worktree", "Switch worktree", "Status", "Push", "Quit"},
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m *model) filterPalette() {
	query := strings.ToLower(m.paletteInput.Value())
	m.paletteFiltered = nil
	for _, c := range commands {
		if query == "" || strings.Contains(c.name, query) || strings.Contains(strings.ToLower(c.desc), query) {
			m.paletteFiltered = append(m.paletteFiltered, c)
		}
	}
	if m.paletteCursor >= len(m.paletteFiltered) {
		m.paletteCursor = max(0, len(m.paletteFiltered)-1)
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		// Global: ctrl+p opens palette from any page
		if key == "ctrl+p" && m.page != pagePalette {
			m.prevPage = m.page
			m.page = pagePalette
			m.paletteInput = newPaletteInput()
			m.paletteFiltered = commands
			m.paletteCursor = 0
			return m, textinput.Blink
		}

		// Global quit
		if key == "ctrl+c" {
			return m, tea.Quit
		}

		// Page-specific handling
		switch m.page {
		case pagePalette:
			return m.updatePalette(msg)
		case pageMenu:
			return m.updateMenu(msg)
		case pageAdd:
			return m.updateAdd(msg)
		case pageSwitch:
			return m.updateSwitch(msg)
		}
	}
	return m, nil
}

func (m model) updatePalette(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.page = m.prevPage
		return m, nil
	case "enter":
		if len(m.paletteFiltered) > 0 {
			selected := m.paletteFiltered[m.paletteCursor]
			m.message = fmt.Sprintf("→ %s", selected.name)
			m.page = selected.page
		}
		return m, nil
	case "down", "ctrl+n":
		if m.paletteCursor < len(m.paletteFiltered)-1 {
			m.paletteCursor++
		}
		return m, nil
	case "up", "ctrl+k":
		if m.paletteCursor > 0 {
			m.paletteCursor--
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.paletteInput, cmd = m.paletteInput.Update(msg)
	m.filterPalette()
	return m, cmd
}

func (m model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "down", "j":
		if m.menuCursor < len(m.menuItems)-1 {
			m.menuCursor++
		}
	case "up", "k":
		if m.menuCursor > 0 {
			m.menuCursor--
		}
	case "enter":
		switch m.menuCursor {
		case 0:
			m.page = pageAdd
		case 1:
			m.page = pageSwitch
		case 4:
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) updateAdd(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.page = pageMenu
	}
	return m, nil
}

func (m model) updateSwitch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.page = pageMenu
	}
	return m, nil
}

// --- Styles ---

var (
	headerStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("62"))
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	helpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	selStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	boxStyle     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")).Padding(1, 2)
	paletteStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("205")).Padding(0, 1).Width(36)
)

func (m model) View() string {
	switch m.page {
	case pagePalette:
		return m.viewPalette()
	case pageMenu:
		return m.viewMenu()
	case pageAdd:
		return m.viewAdd()
	case pageSwitch:
		return m.viewSwitch()
	}
	return ""
}

func (m model) viewPalette() string {
	var b strings.Builder
	b.WriteString(m.paletteInput.View())
	b.WriteString("\n\n")

	for i, cmd := range m.paletteFiltered {
		cursor := "  "
		style := dimStyle
		if i == m.paletteCursor {
			cursor = "> "
			style = selStyle
		}
		b.WriteString(fmt.Sprintf("%s%-10s %s\n", cursor, style.Render(cmd.name), dimStyle.Render(cmd.desc)))
	}

	if len(m.paletteFiltered) == 0 {
		b.WriteString(dimStyle.Render("  no matches\n"))
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/↓ • enter • esc"))

	return paletteStyle.Render(b.String())
}

func (m model) viewMenu() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("Worktree"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("────────────────────────────"))
	b.WriteString("\n\n")

	for i, item := range m.menuItems {
		cursor := "  "
		style := dimStyle
		if i == m.menuCursor {
			cursor = "> "
			style = selStyle
		}
		b.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(item)))
	}

	if m.message != "" {
		b.WriteString(fmt.Sprintf("\n%s\n", selStyle.Render(m.message)))
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/↓ • enter • ctrl+p • q"))

	return boxStyle.Render(b.String())
}

func (m model) viewAdd() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("Worktree > Add"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("────────────────────────────"))
	b.WriteString("\n\n")
	b.WriteString("  Add worktree page\n")
	b.WriteString("  (placeholder)\n")
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("esc back • ctrl+p palette"))

	return boxStyle.Render(b.String())
}

func (m model) viewSwitch() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("Worktree > Switch"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("────────────────────────────"))
	b.WriteString("\n\n")
	b.WriteString("  Switch worktree page\n")
	b.WriteString("  (placeholder)\n")
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("esc back • ctrl+p palette"))

	return boxStyle.Render(b.String())
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
