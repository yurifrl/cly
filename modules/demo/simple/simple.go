package simple

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type model int

func initialModel() model {
	return model(5)
}

func (m model) Init() tea.Cmd {
	return tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "ctrl+z":
			return m, tea.Suspend
		}

	case tickMsg:
		m--
		if m <= 0 {
			return m, tea.Quit
		}
		return m, tick
	}
	return m, nil
}

func (m model) View() string {
	return fmt.Sprintf("Hi. This program will exit in %d seconds.\n\nTo quit sooner press ctrl-c, or press ctrl-z to suspend...\n", m)
}

type tickMsg time.Time

func tick() tea.Msg {
	time.Sleep(time.Second)
	return tickMsg{}
}
