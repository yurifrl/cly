package fullscreen

// A simple program that opens the alternate screen buffer then counts down
// from 5 and then exits.

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
)

type model int

type tickMsg time.Time

func (m model) Init() tea.Cmd {
	return tick()
}

func (m model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}

	case tickMsg:
		m--
		if m <= 0 {
			return m, tea.Quit
		}
		return m, tick()
	}

	return m, nil
}

func (m model) View() tea.View {
	v := tea.NewView(fmt.Sprintf("\n\n     Hi. This program will exit in %d seconds...", m))
	v.AltScreen = true
	return v
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func initialModel() model { return model(5) }
