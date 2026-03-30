package windowsize

// A simple program that queries and displays the window-size.

import (
	tea "charm.land/bubbletea/v2"
)

type model struct{}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if s := msg.String(); s == "ctrl+c" || s == "q" || s == "esc" {
			return m, tea.Quit
		}
		return m, tea.RequestWindowSize

	case tea.WindowSizeMsg:
		return m, tea.Printf("The window size is: %dx%d", msg.Width, msg.Height)
	}

	return m, nil
}

func (m model) View() tea.View {
	return tea.NewView("\nWhen you're done press q to quit.\nPress any other key to query the window-size.\n")
}

func initialModel() model { return model{} }
