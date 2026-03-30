package debounce

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
)

const debounceDuration = time.Second

type exitMsg int

type model struct {
	tag int
}

func initialModel() model {
	return model{}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		// Increment the tag on the model...
		m.tag++
		return m, tea.Tick(debounceDuration, func(_ time.Time) tea.Msg {
			// ...and include a copy of that tag value in the message.
			return exitMsg(m.tag)
		})
	case exitMsg:
		// If the tag in the message doesn't match the tag on the model then we
		// know that this message was not the last one sent and another is on
		// the way. If that's the case we know, we can ignore this message.
		// Otherwise, the debounce timeout has passed and this message is a
		// valid debounced one.
		if int(msg) == m.tag {
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m model) View() tea.View {
	return tea.NewView(fmt.Sprintf("Key presses: %d", m.tag) +
		"\nTo exit press any key, then wait for one second without pressing anything.")
}
