package oi

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type interactiveModel struct {
	input    string
	cursor   int
	results  []string
	loading  bool
	quitting bool
}

func newInteractiveModel() interactiveModel {
	return interactiveModel{}
}

// msg types
type resultMsg string

func (m interactiveModel) Init() tea.Cmd {
	return nil
}

func (m interactiveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case resultMsg:
		m.results = append(m.results, string(msg))
		m.loading = false
		return m, nil

	case tea.KeyMsg:
		if m.loading {
			// Only allow quit while loading
			if msg.String() == "ctrl+c" || msg.String() == "esc" {
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			input := strings.TrimSpace(m.input)
			if input != "" {
				m.loading = true
				captured := input
				m.input = ""
				m.cursor = 0
				return m, func() tea.Msg {
					return resultMsg(checkForInteractive(captured))
				}
			}
			return m, nil
		case "backspace":
			if m.cursor > 0 {
				m.input = m.input[:m.cursor-1] + m.input[m.cursor:]
				m.cursor--
			}
		case "left":
			if m.cursor > 0 {
				m.cursor--
			}
		case "right":
			if m.cursor < len(m.input) {
				m.cursor++
			}
		default:
			ch := msg.String()
			if len(ch) == 1 || ch == " " {
				m.input = m.input[:m.cursor] + ch + m.input[m.cursor:]
				m.cursor++
			}
		}
	}
	return m, nil
}

var (
	promptS = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	inputS  = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	hintS   = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
	loadS   = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
)

func (m interactiveModel) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}

	var b strings.Builder
	for _, r := range m.results {
		b.WriteString(r)
	}

	if m.loading {
		b.WriteString(loadS.Render("  ⏳ checking...") + "\n")
	} else {
		b.WriteString(promptS.Render("  oi") + " " + hintS.Render("›") + " ")
		b.WriteString(inputS.Render(m.input))
		b.WriteString("█\n")
		b.WriteString(hintS.Render("  enter to check · esc to quit"))
		b.WriteString("\n")
	}

	return tea.NewView(b.String())
}
