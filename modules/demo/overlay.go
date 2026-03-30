package demo

import (
	"fmt"
	"os"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
)

var (
	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(true).
			Padding(0, 1)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")).
			MarginBottom(1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	searchStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	countStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

type overlayModel struct {
	demos    []demoEntry
	filtered []demoEntry
	cursor   int
	search   string
	searching bool
	width    int
	height   int
	chosen   string
}

type demoEntry struct {
	name  string
	short string
	cmd   *cobra.Command
}

func getDemos(parent *cobra.Command) []demoEntry {
	var demos []demoEntry
	for _, cmd := range parent.Commands() {
		if cmd.Hidden {
			continue
		}
		demos = append(demos, demoEntry{
			name:  cmd.Use,
			short: cmd.Short,
			cmd:   cmd,
		})
	}
	sort.Slice(demos, func(i, j int) bool {
		return demos[i].name < demos[j].name
	})
	return demos
}

func newOverlayModel(parent *cobra.Command) overlayModel {
	demos := getDemos(parent)
	return overlayModel{
		demos:    demos,
		filtered: demos,
	}
}

func (m overlayModel) Init() tea.Cmd {
	return nil
}

func (m overlayModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		if m.searching {
			switch msg.String() {
			case "esc":
				m.searching = false
				m.search = ""
				m.filtered = m.demos
				m.cursor = 0
			case "backspace":
				if len(m.search) > 0 {
					m.search = m.search[:len(m.search)-1]
					m.applyFilter()
				}
			case "enter":
				m.searching = false
				if len(m.filtered) > 0 {
					m.chosen = m.filtered[m.cursor].name
					return m, tea.Quit
				}
			default:
				if len(msg.Text) > 0 {
					m.search += msg.Text
					m.applyFilter()
				}
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "/":
			m.searching = true
			m.search = ""
			return m, nil
		case "enter":
			if len(m.filtered) > 0 {
				m.chosen = m.filtered[m.cursor].name
				return m, tea.Quit
			}
		case "h", "left":
			if m.cursor > 0 {
				m.cursor--
			}
		case "l", "right":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case "j", "down":
			cols := m.cols()
			if m.cursor+cols < len(m.filtered) {
				m.cursor += cols
			}
		case "k", "up":
			cols := m.cols()
			if m.cursor-cols >= 0 {
				m.cursor -= cols
			}
		}
	}
	return m, nil
}

func (m *overlayModel) applyFilter() {
	if m.search == "" {
		m.filtered = m.demos
	} else {
		var f []demoEntry
		q := strings.ToLower(m.search)
		for _, d := range m.demos {
			if strings.Contains(strings.ToLower(d.name), q) || strings.Contains(strings.ToLower(d.short), q) {
				f = append(f, d)
			}
		}
		m.filtered = f
	}
	m.cursor = 0
}

func (m overlayModel) cols() int {
	if m.width < 40 {
		return 1
	}
	cols := m.width / 30
	if cols < 1 {
		cols = 1
	}
	return cols
}

func (m overlayModel) View() tea.View {
	var b strings.Builder

	b.WriteString(titleStyle.Render("🎨 Demo Browser"))
	b.WriteString("  ")
	b.WriteString(countStyle.Render(fmt.Sprintf("(%d demos)", len(m.filtered))))
	b.WriteString("\n\n")

	if m.searching {
		b.WriteString(searchStyle.Render("/ " + m.search + "█"))
		b.WriteString("\n\n")
	}

	cols := m.cols()
	colWidth := 30
	if m.width > 0 {
		colWidth = m.width / cols
		if colWidth > 40 {
			colWidth = 40
		}
	}

	for i, d := range m.filtered {
		label := d.name
		if len(label) > colWidth-4 {
			label = label[:colWidth-7] + "..."
		}

		var cell string
		if i == m.cursor {
			cell = selectedStyle.Width(colWidth - 2).Render("▸ " + label)
		} else {
			cell = normalStyle.Width(colWidth - 2).Render("  " + label)
		}

		b.WriteString(cell)
		if (i+1)%cols == 0 {
			b.WriteString("\n")
		}
	}

	if len(m.filtered)%cols != 0 {
		b.WriteString("\n")
	}

	b.WriteString("\n")
	if m.searching {
		b.WriteString(helpStyle.Render("type to filter • enter select • esc cancel"))
	} else {
		b.WriteString(helpStyle.Render("h/l ←/→ navigate • j/k ↑/↓ rows • / search • enter run • q quit"))
	}

	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}

func runOverlay(parent *cobra.Command) error {
	for {
		m := newOverlayModel(parent)
		p := tea.NewProgram(m)
		result, err := p.Run()
		if err != nil {
			return err
		}

		final := result.(overlayModel)
		if final.chosen == "" {
			return nil // user quit
		}

		// Find and run the chosen demo
		for _, cmd := range parent.Commands() {
			if cmd.Use == final.chosen {
				// Run the demo; when it exits (ctrl+c), we loop back
				cmd.SetOut(os.Stdout)
				cmd.SetErr(os.Stderr)
				cmd.SetIn(os.Stdin)
				if cmd.RunE != nil {
					_ = cmd.RunE(cmd, nil)
				} else if cmd.Run != nil {
					cmd.Run(cmd, nil)
				}
				break
			}
		}
		// Loop back to overlay
	}
}
