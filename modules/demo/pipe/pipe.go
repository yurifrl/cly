package pipe

// An example illustrating how to pipe in data to a Bubble Tea application.
// More so, this serves as proof that Bubble Tea will automatically listen for
// keystrokes when input is not a TTY, such as when data is piped or redirected
// in.

import (
	"fmt"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type model struct {
	userInput textinput.Model
}

func newModel(initialValue string) (m model) {
	i := textinput.New()
	i.Prompt = ""

	s := i.Styles()
	s.Cursor.Color = lipgloss.Color("63")
	i.SetStyles(s)

	i.SetWidth(48)
	i.SetValue(initialValue)
	i.CursorEnd()
	i.Focus()

	m.userInput = i
	return m
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c", "esc", "enter":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.userInput, cmd = m.userInput.Update(msg)
	return m, cmd
}

func (m model) View() tea.View {
	return tea.NewView(fmt.Sprintf(
		"\nYou piped in: %s\n\nPress ^C to exit",
		m.userInput.View(),
	))
}
