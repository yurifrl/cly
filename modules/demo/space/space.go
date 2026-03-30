package space

import (
	"fmt"
	"image/color"
	"math/rand"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
)

type model struct {
	colors     [][]color.Color
	lastWidth  int
	lastHeight int
	frameCount int
	width      int
	height     int
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tickCmd())
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second/60, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

type tickMsg struct{}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.width != m.lastWidth || m.height != m.lastHeight {
			m.setupColors()
			m.lastWidth = m.width
			m.lastHeight = m.height
		}
	case tickMsg:
		m.frameCount++
		return m, tickCmd()
	}
	return m, nil
}

func (m *model) setupColors() {
	height := m.height * 2
	m.colors = make([][]color.Color, height)
	for y := range height {
		m.colors[y] = make([]color.Color, m.width)
		randomnessFactor := float64(height-y) / float64(height)
		for x := range m.width {
			baseValue := randomnessFactor * (float64(height-y) / float64(height))
			randomOffset := (rand.Float64() * 0.2) - 0.1
			value := clampF(baseValue+randomOffset, 0, 1)
			gray := uint8(value * 255)
			m.colors[y][x] = lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", gray, gray, gray))
		}
	}
}

func clampF(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func (m model) View() tea.View {
	title := lipgloss.NewStyle().Bold(true).Render("Space")
	var s strings.Builder
	height := m.height - 1
	for y := range height {
		for x := range m.width {
			xi := (x + m.frameCount) % m.width
			fg := m.colors[y*2][xi]
			bg := m.colors[y*2+1][xi]
			st := lipgloss.NewStyle().Foreground(fg).Background(bg)
			s.WriteString(st.Render("▀"))
		}
		if y < height-1 {
			s.WriteString("\n")
		}
	}
	v := tea.NewView(strings.Join([]string{title, s.String()}, "\n"))
	v.AltScreen = true
	return v
}

func run(cmd *cobra.Command, args []string) error {
	p := tea.NewProgram(model{}, tea.WithFPS(120))
	_, err := p.Run()
	return err
}
