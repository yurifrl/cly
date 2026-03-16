package helpy

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

var (
	helpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render
	searchStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	searchPrompt = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render("/")
	noMatchStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render
	paletteStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1).
			Width(50)
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	normalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
)

type header struct {
	title string
	line  int
}

type model struct {
	viewport        viewport.Model
	searchInput     textinput.Model
	paletteInput    textinput.Model
	headers         []header
	filteredHeaders []header
	paletteIndex    int
	content         string
	rendered        string
	searching       bool
	paletteOpen     bool
	chatOpen        bool
	chat            chatModel
	chatEnabled     bool
	searchQuery     string
	matches         []int
	matchIndex      int
	width           int
	height          int
	lastWidth       int
	ready           bool
}

func initialModel(content string) (*model, error) {
	ti := textinput.New()
	ti.Placeholder = "search..."
	ti.CharLimit = 100
	ti.Width = 30

	pi := textinput.New()
	pi.Placeholder = "type to filter..."
	pi.CharLimit = 50
	pi.Width = 46

	headers := extractHeaders(content)

	// Pre-render with default width to avoid "Loading..." delay
	rendered := content
	if r, err := glamour.NewTermRenderer(glamour.WithAutoStyle(), glamour.WithWordWrap(76)); err == nil {
		if out, err := r.Render(content); err == nil {
			rendered = out
		}
	}

	// Create viewport with default size
	vp := viewport.New(80, 24)
	vp.Style = lipgloss.NewStyle().PaddingLeft(2).PaddingRight(2)
	vp.SetContent(rendered)

	return &model{
		content:         content,
		rendered:        rendered,
		viewport:        vp,
		lastWidth:       80,
		ready:           true,
		searchInput:     ti,
		paletteInput:    pi,
		headers:         headers,
		filteredHeaders: headers,
	}, nil
}

func extractHeaders(content string) []header {
	var headers []header
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		if strings.HasPrefix(line, "#") {
			title := strings.TrimLeft(line, "#")
			title = strings.TrimSpace(title)
			if title != "" {
				headers = append(headers, header{title: title, line: i})
			}
		}
	}

	return headers
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// When chat is open, delegate most messages to chat
	if m.chatOpen {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
			m.resizeForChat()
			// Forward to chat too
			var chatCmd tea.Cmd
			m.chat, chatCmd = m.chat.Update(tea.WindowSizeMsg{
				Width:  msg.Width,
				Height: m.chatHeight(),
			})
			cmds = append(cmds, chatCmd)
			return m, tea.Batch(cmds...)

		case tea.KeyMsg:
			if msg.String() == "esc" && !m.chat.loading {
				m.chatOpen = false
				m.resizeForDoc()
				return m, nil
			}

		case streamChunkMsg, spinner.TickMsg:
			// These always go to chat
		}

		var chatCmd tea.Cmd
		m.chat, chatCmd = m.chat.Update(msg)
		cmds = append(cmds, chatCmd)
		return m, tea.Batch(cmds...)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 2
		m.setContent(m.content, msg.Width)

	case tea.KeyMsg:
		if m.paletteOpen {
			switch msg.String() {
			case "enter":
				if len(m.filteredHeaders) > 0 && m.paletteIndex < len(m.filteredHeaders) {
					m.jumpToHeader(m.filteredHeaders[m.paletteIndex])
				}
				m.closePalette()
				return m, nil
			case "esc", "ctrl+c", "ctrl+p":
				m.closePalette()
				return m, nil
			case "up", "ctrl+k":
				if m.paletteIndex > 0 {
					m.paletteIndex--
				}
				return m, nil
			case "down", "ctrl+j":
				if m.paletteIndex < len(m.filteredHeaders)-1 {
					m.paletteIndex++
				}
				return m, nil
			default:
				var cmd tea.Cmd
				m.paletteInput, cmd = m.paletteInput.Update(msg)
				m.filterHeaders()
				return m, cmd
			}
		}

		if m.searching {
			switch msg.String() {
			case "enter":
				m.searchQuery = m.searchInput.Value()
				m.searching = false
				m.searchInput.Blur()
				m.findMatches()
				if len(m.matches) > 0 {
					m.jumpToMatch(0)
				}
				return m, nil
			case "esc":
				m.searching = false
				m.searchInput.Reset()
				m.searchInput.Blur()
				return m, nil
			default:
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(msg)
				return m, cmd
			}
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.clearSearch()
			return m, nil
		case "/":
			m.searching = true
			m.searchInput.Focus()
			return m, textinput.Blink
		case "ctrl+p":
			m.openPalette()
			return m, textinput.Blink
		case "ctrl+a":
			if m.chatEnabled {
				m.chatOpen = true
				m.resizeForChat()
				cmd := m.chat.Init()
				return m, cmd
			}
			return m, nil
		case "n":
			if len(m.matches) > 0 {
				m.matchIndex = (m.matchIndex + 1) % len(m.matches)
				m.jumpToMatch(m.matchIndex)
			}
			return m, nil
		case "N":
			if len(m.matches) > 0 {
				m.matchIndex = (m.matchIndex - 1 + len(m.matches)) % len(m.matches)
				m.jumpToMatch(m.matchIndex)
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *model) View() string {
	if !m.ready {
		return "Loading..."
	}

	if m.paletteOpen {
		return m.paletteView()
	}

	var footer string
	if m.searching {
		footer = "\n  " + searchPrompt + m.searchInput.View() + "\n"
	} else if m.searchQuery != "" {
		if len(m.matches) == 0 {
			footer = noMatchStyle("\n  No matches for: " + m.searchQuery + " (esc to clear)\n")
		} else {
			footer = helpStyle("\n  ") + searchStyle.Render(fmt.Sprintf("match %d/%d", m.matchIndex+1, len(m.matches))) + helpStyle(" • n/N: next/prev • esc: clear\n")
		}
	} else {
		footer = m.helpView()
	}

	if m.chatOpen {
		return m.viewport.View() + "\n" + m.chat.View()
	}

	return m.viewport.View() + footer
}

func (m *model) paletteView() string {
	var b strings.Builder

	// Input line
	b.WriteString(m.paletteInput.View())
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 48))
	b.WriteString("\n")

	// Show filtered items (max 10)
	maxItems := min(10, len(m.filteredHeaders))
	for i := 0; i < maxItems; i++ {
		h := m.filteredHeaders[i]
		if i == m.paletteIndex {
			b.WriteString(selectedStyle.Render("> " + h.title))
		} else {
			b.WriteString(normalStyle.Render("  " + h.title))
		}
		if i < maxItems-1 {
			b.WriteString("\n")
		}
	}

	if len(m.filteredHeaders) == 0 {
		b.WriteString(helpStyle("  no matches"))
	}

	content := paletteStyle.Render(b.String())

	// Center the palette
	padTop := (m.height - lipgloss.Height(content)) / 3
	if padTop < 0 {
		padTop = 0
	}

	return strings.Repeat("\n", padTop) + lipgloss.PlaceHorizontal(m.width, lipgloss.Center, content)
}

func (m *model) helpView() string {
	hint := "\n  ↑/↓: Navigate • /: Search • ^P: Headers"
	if m.chatEnabled {
		hint += " • ^A: AI Chat"
	}
	hint += " • q: Quit\n"
	return helpStyle(hint)
}

func (m *model) openPalette() {
	m.paletteOpen = true
	m.paletteInput.Reset()
	m.paletteInput.Focus()
	m.paletteIndex = 0
	m.filteredHeaders = m.headers
}

func (m *model) closePalette() {
	m.paletteOpen = false
	m.paletteInput.Blur()
	m.paletteInput.Reset()
}

func (m *model) filterHeaders() {
	query := strings.ToLower(m.paletteInput.Value())
	if query == "" {
		m.filteredHeaders = m.headers
		m.paletteIndex = 0
		return
	}

	var filtered []header
	for _, h := range m.headers {
		if strings.Contains(strings.ToLower(h.title), query) {
			filtered = append(filtered, h)
		}
	}
	m.filteredHeaders = filtered
	m.paletteIndex = 0
}

func (m *model) setContent(content string, width int) {
	// Only re-render if width changed significantly or not yet rendered
	if m.rendered != "" && m.lastWidth > 0 && abs(width-m.lastWidth) < 10 {
		m.viewport.SetContent(m.rendered)
		return
	}
	m.lastWidth = width

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width-4),
	)
	if err != nil {
		m.rendered = content
		m.viewport.SetContent(content)
		return
	}

	rendered, err := renderer.Render(content)
	if err != nil {
		m.rendered = content
		m.viewport.SetContent(content)
		return
	}

	m.rendered = rendered
	m.viewport.SetContent(rendered)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (m *model) findMatches() {
	m.matches = nil
	m.matchIndex = 0

	if m.searchQuery == "" {
		return
	}

	query := strings.ToLower(m.searchQuery)
	lines := strings.Split(m.rendered, "\n")

	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), query) {
			m.matches = append(m.matches, i)
		}
	}
}

func (m *model) jumpToMatch(index int) {
	if index < 0 || index >= len(m.matches) {
		return
	}
	lineNum := m.matches[index]
	m.viewport.SetYOffset(lineNum)
}

func (m *model) jumpToHeader(h header) {
	// Search rendered content for header title and jump there
	lines := strings.Split(m.rendered, "\n")
	title := strings.ToLower(h.title)
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), title) {
			m.viewport.SetYOffset(i)
			return
		}
	}
}

func (m *model) clearSearch() {
	m.searchQuery = ""
	m.matches = nil
	m.matchIndex = 0
	m.searchInput.Reset()
}

// chatHeight returns the height allocated to the chat panel.
func (m *model) chatHeight() int {
	h := m.height * 40 / 100 // 40% of screen
	if h < 10 {
		h = 10
	}
	return h
}

// resizeForChat shrinks the doc viewport and sizes the chat panel.
func (m *model) resizeForChat() {
	chatH := m.chatHeight()
	m.viewport.Height = m.height - chatH - 1
	m.viewport.Width = m.width
	m.setContent(m.content, m.width)

	m.chat.width = m.width
	m.chat.height = chatH
	m.chat.viewport.Width = m.width
	m.chat.textarea.SetWidth(m.width)
	m.chat.viewport.Height = chatH - m.chat.textarea.Height() - 3
	m.chat.refreshViewport()
}

// resizeForDoc restores the doc viewport to full height.
func (m *model) resizeForDoc() {
	m.viewport.Height = m.height - 2
	m.viewport.Width = m.width
	m.setContent(m.content, m.width)
}
