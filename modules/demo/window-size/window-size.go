package windowsize

import (
	tea "github.com/charmbracelet/bubbletea"
)

type model struct{}

func initialModel() model {
	return model{}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if s := msg.String(); s == "ctrl+c" || s == "q" || s == "esc" {
			return m, tea.Quit
		}

		return m, tea.WindowSize()

	case tea.WindowSizeMsg:
		return m, tea.Printf("%dx%d", msg.Width, msg.Height)
	}

	return m, nil
}

func (m model) View() string {
	s := "When you're done press q to quit. Press any other key to query the window-size.\n"

	return s
}
