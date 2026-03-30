package clickable

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/segmentio/ksuid"
	"github.com/spf13/cobra"
)

type LayerHitMsg struct {
	ID    string
	Mouse tea.MouseMsg
}

const maxDialogs = 999

var (
	bgTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("239")).
			Padding(1, 2)

	bgWhitespace = []lipgloss.WhitespaceOption{
		lipgloss.WithWhitespaceChars("/"),
		lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("238"))),
	}

	dialogWordStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E7E1CC"))

	dialogStyle = dialogWordStyle.
			Width(36).
			Height(8).
			Padding(1, 3).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#874BFD"))

	hoveredDialogStyle = dialogStyle.
				BorderForeground(lipgloss.Color("#F25D94"))

	specialWordLightColor = lipgloss.Color("#43BF6D")
	specialWordDarkColor  = lipgloss.Color("#73F59F")

	buttonStyle = lipgloss.NewStyle().
			Padding(0, 3).
			Foreground(lipgloss.Color("#FFF7DB")).
			Background(lipgloss.Color("#6124DF"))

	hoveredButtonStyle = buttonStyle.
				Background(lipgloss.Color("#FF5F87"))
)

type model struct {
	specialWordStyle lipgloss.Style
	width, height    int
	dialogs          []dialog
	mouseDown        bool
	pressID          string
	dragID           string
	dragOffsetX      int
	dragOffsetY      int
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tea.RequestBackgroundColor)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.BackgroundColorMsg:
		if msg.IsDark() {
			m.specialWordStyle = m.specialWordStyle.Foreground(specialWordDarkColor)
		} else {
			m.specialWordStyle = m.specialWordStyle.Foreground(specialWordLightColor)
		}
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}
	case LayerHitMsg:
		mouse := msg.Mouse.Mouse()
		switch msg.Mouse.(type) {
		case tea.MouseClickMsg:
			if mouse.Button != tea.MouseLeft {
				break
			}
			if !m.mouseDown {
				m.mouseDown = true
				m.pressID = msg.ID
				for i, d := range m.dialogs {
					if d.id != msg.ID {
						continue
					}
					m.dragID = msg.ID
					m.dragOffsetX = mouse.X - d.x
					m.dragOffsetY = mouse.Y - d.y
					if len(m.dialogs) < 2 {
						break
					}
					m.dialogs = m.removeDialog(i)
					m.dialogs = append(m.dialogs, d)
					break
				}
				break
			}
		case tea.MouseMotionMsg:
			if m.mouseDown && m.dragID != "" {
				for i := range m.dialogs {
					d := &m.dialogs[i]
					if d.id != m.dragID {
						continue
					}
					if m.dragID == d.id {
						d.x = clamp(mouse.X-(m.dragOffsetX), 0, m.width-lipgloss.Width(d.windowView()))
						d.y = clamp(mouse.Y-(m.dragOffsetY), 0, m.height-lipgloss.Height(d.windowView()))
					}
					break
				}
			}
			for i := range m.dialogs {
				d := &m.dialogs[i]
				d.hovering = false
				d.hoveringButton = false
				if d.id == msg.ID {
					d.hovering = true
					continue
				}
				if d.buttonID == msg.ID {
					d.hovering = true
					d.hoveringButton = true
					continue
				}
			}
		case tea.MouseReleaseMsg:
			if m.pressID == "" {
				break
			}
			for i, d := range m.dialogs {
				if msg.ID == d.buttonID && m.pressID == d.buttonID {
					m.dialogs = m.removeDialog(i)
					break
				}
			}
			if msg.ID == "bg" && m.pressID == "bg" {
				if len(m.dialogs) < maxDialogs {
					m.dialogs = append(m.dialogs, m.newDialog(mouse.X, mouse.Y))
				}
			}
			m.mouseDown = false
			m.dragID = ""
			m.pressID = ""
		}
	}
	return m, nil
}

func (m model) View() tea.View {
	var v tea.View
	var body string
	n := len(m.dialogs)
	if n > 0 {
		body += "Drag to move. "
	}
	if n == 0 && n < maxDialogs {
		body += "Click to spawn."
	} else if n >= 1 && n < maxDialogs {
		body += fmt.Sprintf("Click to spawn up to %d more.", maxDialogs-len(m.dialogs))
	}
	body += "\n\nPress q to quit."
	bg := lipgloss.Place(m.width, m.height, lipgloss.Top, lipgloss.Left, bgTextStyle.Render(body), bgWhitespace...)
	root := lipgloss.NewLayer(bg).ID("bg")
	for i, d := range m.dialogs {
		root.AddLayers(d.view().Z(i + 1))
	}
	comp := lipgloss.NewCompositor(root)
	v.MouseMode = tea.MouseModeAllMotion
	v.AltScreen = true
	v.OnMouse = func(msg tea.MouseMsg) tea.Cmd {
		return func() tea.Msg {
			mouse := msg.Mouse()
			x, y := mouse.X, mouse.Y
			if id := comp.Hit(x, y).ID(); id != "" {
				return LayerHitMsg{ID: id, Mouse: msg}
			}
			return nil
		}
	}
	v.SetContent(comp.Render())
	return v
}

func (m *model) newDialog(x, y int) (d dialog) {
	d.specialWordStyle = &m.specialWordStyle
	dummyView := d.windowView()
	w := lipgloss.Width(dummyView)
	h := lipgloss.Height(dummyView)
	d.x = clamp(x-w/2, 0, m.width-w)
	d.y = clamp(y-h/2, 0, m.height-h)
	d.text = nextRandomWord()
	d.id = ksuid.New().String()
	d.buttonID = ksuid.New().String()
	return d
}

func (m model) removeDialog(index int) []dialog {
	d := m.dialogs
	if len(d) <= index {
		return m.dialogs
	}
	copy(d[index:], d[index+1:])
	d[len(d)-1] = dialog{}
	return d[:len(d)-1]
}

type dialog struct {
	specialWordStyle *lipgloss.Style
	id               string
	buttonID         string
	x, y             int
	text             string
	hovering         bool
	hoveringButton   bool
}

func (d dialog) buttonView() string {
	const label = "Run Away"
	if d.hoveringButton {
		return hoveredButtonStyle.Render(label)
	}
	return buttonStyle.Render(label)
}

func (d dialog) windowView() string {
	var style lipgloss.Style
	if d.hovering {
		style = hoveredDialogStyle
	} else {
		style = dialogStyle
	}
	s := d.specialWordStyle.Render(d.text) + dialogWordStyle.Render(" draws near. Command?")
	return style.Render(s)
}

func (d dialog) view() *lipgloss.Layer {
	const hGap, vGap = 3, 1
	window := d.windowView()
	button := d.buttonView()
	buttonX := lipgloss.Width(window) - lipgloss.Width(button) - 1 - hGap
	buttonY := lipgloss.Height(window) - lipgloss.Height(button) - 1 - vGap
	buttonLayer := lipgloss.NewLayer(button).ID(d.buttonID).X(buttonX).Y(buttonY)
	return lipgloss.NewLayer(window).ID(d.id).X(d.x).Y(d.y).AddLayers(buttonLayer)
}

func run(cmd *cobra.Command, args []string) error {
	ksuid.SetRand(ksuid.FastRander)
	if _, err := tea.NewProgram(model{}).Run(); err != nil {
		return err
	}
	return nil
}

func clamp(n, min, max int) int {
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}
