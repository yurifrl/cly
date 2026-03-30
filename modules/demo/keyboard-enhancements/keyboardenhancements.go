package keyboardenhancements

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
)

type styles struct {
	ui lipgloss.Style
}

type model struct {
	supportsDisambiguation bool
	supportsEventTypes     bool
	styles                 styles
}

func (m model) Init() tea.Cmd {
	return tea.RequestBackgroundColor
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyboardEnhancementsMsg:
		m.supportsDisambiguation = true
		m.supportsEventTypes = msg.SupportsEventTypes()
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		default:
			return m, tea.Println("  press: " + msg.String())
		}
	case tea.KeyReleaseMsg:
		return m, tea.Printf("release: %s", msg.String())
	case tea.BackgroundColorMsg:
		m.updateStyles(msg.IsDark())
	}
	return m, nil
}

func (m model) View() tea.View {
	var v tea.View
	var b strings.Builder
	fmt.Fprintf(&b, "Terminal supports key releases: %v\n", m.supportsEventTypes)
	fmt.Fprintf(&b, "Terminal supports key disambiguation: %v\n", m.supportsDisambiguation)
	fmt.Fprint(&b, "This demo logs key events. Press ctrl+c to quit.")
	v.SetContent(b.String() + "\n")
	v.KeyboardEnhancements.ReportEventTypes = true
	return v
}

func (m *model) updateStyles(isDark bool) {
	lightDark := lipgloss.LightDark(isDark)
	grey := lightDark(lipgloss.Color("239"), lipgloss.Color("245"))
	darkGray := lightDark(lipgloss.Color("245"), lipgloss.Color("239"))
	m.styles.ui = lipgloss.NewStyle().
		Foreground(grey).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(darkGray)
}

func initialModel() model {
	m := model{}
	m.updateStyles(true)
	return m
}

func run(cmd *cobra.Command, args []string) error {
	p := tea.NewProgram(initialModel())
	_, err := p.Run()
	return err
}
