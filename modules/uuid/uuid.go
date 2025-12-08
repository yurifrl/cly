package uuid

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
)

type item string

func (i item) FilterValue() string { return string(i) }
func (i item) Title() string       { return string(i) }
func (i item) Description() string { return "" }

type model struct {
	list      list.Model
	choice    string
	quitting  bool
	generated string
}

func initialModel() model {
	items := []list.Item{
		item("UUID v4 (random)"),
		item("UUID v7 (time-ordered)"),
		item("Multiple (5x)"),
	}

	l := list.New(items, list.NewDefaultDelegate(), 40, 10)
	l.Title = "Generate UUID"
	l.SetShowHelp(false)

	return model{list: l}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.choice = string(i)
				// Generate UUID based on choice
				switch m.choice {
				case "UUID v4 (random)":
					m.generated = uuid.New().String()
				case "UUID v7 (time-ordered)":
					m.generated = uuid.Must(uuid.NewV7()).String()
				case "Multiple (5x)":
					for i := 0; i < 5; i++ {
						m.generated += uuid.New().String() + "\n"
					}
				}
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.quitting && m.generated == "" {
		return "Cancelled.\n"
	}
	if m.generated != "" {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
		return style.Render(m.generated) + "\n"
	}
	return "\n" + m.list.View()
}
