package pianon

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type toggleItem struct {
	label   string
	key     string
	checked bool
}

type pickerModel struct {
	items    []toggleItem
	cursor   int
	quitting bool
}

func newPickerModel() pickerModel {
	return pickerModel{
		items: []toggleItem{
			{label: "Skills", key: "skills", checked: true},
			{label: "Extensions", key: "extensions", checked: true},
			{label: "Prompt Templates", key: "prompt_templates", checked: true},
		},
	}
}

func (m pickerModel) Init() tea.Cmd {
	return nil
}

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "space", " ", "x":
			m.items[m.cursor].checked = !m.items[m.cursor].checked
		}
	}
	return m, nil
}

func (m pickerModel) View() tea.View {
	var b strings.Builder

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4")).Render("🔧 Configure session")
	b.WriteString(title + "\n\n")

	for i, item := range m.items {
		cursor := "  "
		if i == m.cursor {
			cursor = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Render("▸ ")
		}

		check := "✓"
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
		if !item.checked {
			check = "✗"
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B"))
		}

		label := item.label
		if i == m.cursor {
			label = lipgloss.NewStyle().Bold(true).Render(label)
		}

		b.WriteString(fmt.Sprintf("%s%s %s\n", cursor, style.Render(check), label))
	}

	help := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render("\n↑/k up • ↓/j down • space toggle • enter confirm • esc cancel")
	b.WriteString(help + "\n")

	return tea.NewView(b.String())
}

// promptToggles runs the TUI picker and returns the selected toggles.
// Returns (toggles, cancelled).
func promptToggles() (Toggles, bool) {
	m := newPickerModel()
	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return Toggles{Skills: true, Extensions: true, PromptTemplates: true}, false
	}

	final, ok := result.(pickerModel)
	if !ok || final.quitting {
		return Toggles{}, true
	}

	t := Toggles{Skills: true, Extensions: true, PromptTemplates: true}
	for _, item := range final.items {
		switch item.key {
		case "skills":
			t.Skills = item.checked
		case "extensions":
			t.Extensions = item.checked
		case "prompt_templates":
			t.PromptTemplates = item.checked
		}
	}
	return t, false
}
