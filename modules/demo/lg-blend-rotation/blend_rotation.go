package lg_blend_rotation

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
)

const (
	fps   = 15
	steps = 5
)

type tickMsg struct {
	Value int
}

func tick(current int) tea.Cmd {
	return tea.Tick(time.Second/time.Duration(fps), func(_ time.Time) tea.Msg {
		return tickMsg{Value: current + steps}
	})
}

type model struct {
	borderRotation int
}

func (m model) Init() tea.Cmd {
	return tick(0)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	case tickMsg:
		m.borderRotation = msg.Value
		return m, tick(msg.Value)
	}
	return m, nil
}

func (m model) View() tea.View {
	v := tea.NewView(lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForegroundBlend(
			lipgloss.Color("#00FA68"),
			lipgloss.Color("#9900FF"),
			lipgloss.Color("#ED5353"),
			lipgloss.Color("#9900FF"),
			lipgloss.Color("#00FA68"),
		).
		BorderForegroundBlendOffset(m.borderRotation).
		Width(60).
		Height(15).
		Render("Hello, world!"))
	v.AltScreen = true
	return v
}

func run(cmd *cobra.Command, args []string) error {
	_, err := tea.NewProgram(model{}).Run()
	return err
}
