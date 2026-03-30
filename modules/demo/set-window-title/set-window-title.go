package setwindowtitle

// A simple example illustrating how to set a window title.

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const windowTitle = "Hello, Bubble Tea"

type model struct{}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyPressMsg:
		return m, tea.Quit
	}
	return m, nil
}

func (m model) View() tea.View {
	wrap := lipgloss.NewStyle().Width(78).Render
	v := tea.NewView(wrap("The window title has been set to '"+windowTitle+"'. It will be cleared on exit.") +
		"\n\nPress any key to quit.")
	v.WindowTitle = windowTitle
	return v
}

func initialModel() model { return model{} }
