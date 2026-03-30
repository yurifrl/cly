package progressdownload

import (
	"strings"
	"time"

	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	padding  = 2
	maxWidth = 80
)

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render

type progressMsg float64

type progressErrMsg struct{ err error }

type model struct {
	progress progress.Model
	err      error
}

func initialModel() model {
	return model{
		progress: progress.New(progress.WithDefaultBlend()),
	}
}

func (m model) Init() tea.Cmd {
	return simulateDownload()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m, tea.Quit

	case tea.WindowSizeMsg:
		w := msg.Width - padding*2 - 4
		if w > maxWidth {
			w = maxWidth
		}
		m.progress.SetWidth(w)
		return m, nil

	case progressErrMsg:
		m.err = msg.err
		return m, tea.Quit

	case progressMsg:
		var cmds []tea.Cmd

		if msg >= 1.0 {
			cmds = append(cmds, tea.Sequence(finalPause(), tea.Quit))
		}

		cmds = append(cmds, m.progress.SetPercent(float64(msg)))
		return m, tea.Batch(cmds...)

	case progress.FrameMsg:
		var cmd tea.Cmd
		m.progress, cmd = m.progress.Update(msg)
		return m, cmd

	default:
		return m, nil
	}
}

func (m model) View() tea.View {
	if m.err != nil {
		return tea.NewView("Error: " + m.err.Error() + "\n")
	}

	pad := strings.Repeat(" ", padding)
	return tea.NewView("\n" +
		pad + m.progress.View() + "\n\n" +
		pad + helpStyle("Press any key to quit"))
}

func finalPause() tea.Cmd {
	return tea.Tick(time.Millisecond*750, func(_ time.Time) tea.Msg {
		return nil
	})
}

func simulateDownload() tea.Cmd {
	return tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg {
		// Simulate download progress
		return progressMsg(t.Unix()%5 / 5.0)
	})
}
