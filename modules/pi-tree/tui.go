package pitree

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Styles ────────────────────────────────────────────────────────────────────

var (
	wsStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	selWsStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))
	sessionStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	selSessStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))
	dimStyle2     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	sizeStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("72"))
	dateStyle2    = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	histPanelStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)
	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99")).Padding(0, 0, 1, 2)
)

// ── Cursor ────────────────────────────────────────────────────────────────────

// cursor identifies the selected item: workspace index + session index (-1 = ws row)
type cursor struct {
	ws   int
	sess int // -1 means workspace header is selected
}

// ── Model ─────────────────────────────────────────────────────────────────────

type tuiModel struct {
	nodes      []WorkspaceNode
	snapshots  []Snapshot
	cur        cursor
	showHist   bool
	histIdx    int // index into snapshots (latest = len-1)
	viewOffset int
	height     int
	width      int
	message    string
	quit       bool
}

func newTUIModel(nodes []WorkspaceNode, snapshots []Snapshot) tuiModel {
	m := tuiModel{
		nodes:     nodes,
		snapshots: snapshots,
		histIdx:   len(snapshots) - 1,
		cur:       cursor{ws: 0, sess: -1},
	}
	return m
}

func (m tuiModel) Init() tea.Cmd { return nil }

// totalRows returns the total number of navigable rows.
func (m tuiModel) totalRows() int {
	n := 0
	for _, ws := range m.nodes {
		n++ // workspace header
		n += len(ws.Sessions)
	}
	return n
}

// cursorIndex returns the flat index of the cursor.
func (m tuiModel) cursorIndex() int {
	idx := 0
	for wi, ws := range m.nodes {
		if wi == m.cur.ws && m.cur.sess == -1 {
			return idx
		}
		idx++
		for si := range ws.Sessions {
			if wi == m.cur.ws && si == m.cur.sess {
				return idx
			}
			idx++
		}
	}
	return idx
}

// moveCursor moves up (-1) or down (+1).
func (m *tuiModel) moveCursor(delta int) {
	// Build flat list of (ws, sess) positions
	type pos struct{ ws, sess int }
	var positions []pos
	for wi, ws := range m.nodes {
		positions = append(positions, pos{wi, -1})
		for si := range ws.Sessions {
			positions = append(positions, pos{wi, si})
		}
	}
	if len(positions) == 0 {
		return
	}
	cur := m.cursorIndex()
	next := cur + delta
	if next < 0 {
		next = 0
	}
	if next >= len(positions) {
		next = len(positions) - 1
	}
	m.cur.ws = positions[next].ws
	m.cur.sess = positions[next].sess
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quit = true
			return m, tea.Quit

		case "up", "k":
			m.moveCursor(-1)
			m.message = ""

		case "down", "j":
			m.moveCursor(1)
			m.message = ""

		case "h":
			m.showHist = !m.showHist
			m.message = ""

		case "left":
			if m.showHist && m.histIdx > 0 {
				m.histIdx--
				if m.histIdx < len(m.snapshots) {
					m.nodes = m.snapshots[m.histIdx].Tree
					m.cur = cursor{ws: 0, sess: -1}
				}
			}

		case "right":
			if m.showHist && m.histIdx < len(m.snapshots)-1 {
				m.histIdx++
				m.nodes = m.snapshots[m.histIdx].Tree
				m.cur = cursor{ws: 0, sess: -1}
			}

		case "enter":
			m.openSelected()
			return m, tea.Quit

		case "a":
			m.openWorkspace()
			return m, tea.Quit

		case "A":
			m.openAll()
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *tuiModel) openSelected() {
	if m.cur.ws >= len(m.nodes) {
		return
	}
	ws := m.nodes[m.cur.ws]
	if m.cur.sess == -1 {
		// workspace header → open first session
		if len(ws.Sessions) > 0 {
			openSession(ws.Sessions[0].SessionID)
		}
		return
	}
	if m.cur.sess < len(ws.Sessions) {
		openSession(ws.Sessions[m.cur.sess].SessionID)
	}
}

func (m *tuiModel) openWorkspace() {
	if m.cur.ws >= len(m.nodes) {
		return
	}
	for _, s := range m.nodes[m.cur.ws].Sessions {
		launchSession(s.SessionID)
	}
	m.message = fmt.Sprintf("opened %d sessions", len(m.nodes[m.cur.ws].Sessions))
}

func (m *tuiModel) openAll() {
	count := 0
	for _, ws := range m.nodes {
		for _, s := range ws.Sessions {
			launchSession(s.SessionID)
			count++
		}
	}
	m.message = fmt.Sprintf("opened %d sessions", count)
}

// openSession execs pi replacing current process (for single session use).
func openSession(id string) {
	binary, err := exec.LookPath("pi")
	if err != nil {
		return
	}
	args := []string{"pi", "--session", id}
	_ = syscall.Exec(binary, args, syscall.Environ())
}

// launchSession forks a background pi process.
func launchSession(id string) {
	_ = exec.Command("pi", "--session", id).Start()
}

func (m tuiModel) View() string {
	if m.quit {
		return ""
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("π session tree"))
	b.WriteString("\n")

	var treeLines []string
	for wi, ws := range m.nodes {
		wsSelected := m.cur.ws == wi && m.cur.sess == -1
		wsLabel := ws.Name
		prefix := "  "
		if wsSelected {
			treeLines = append(treeLines, selWsStyle.Render(prefix+wsLabel))
		} else {
			treeLines = append(treeLines, wsStyle.Render(prefix+wsLabel))
		}
		for si, s := range ws.Sessions {
			sessSelected := m.cur.ws == wi && m.cur.sess == si
			branch := "  ├── "
			if si == len(ws.Sessions)-1 {
				branch = "  └── "
			}
			size := sizeStyle.Render(fmt.Sprintf("%7s", formatSize(s.SizeBytes)))
			date := dateStyle2.Render(s.StartedAt)
			sid := dimStyle2.Render(s.SessionID)
			line := fmt.Sprintf("    %s%s  %s  %s", branch, sid, size, date)
			if sessSelected {
				treeLines = append(treeLines, selSessStyle.Render(line))
			} else {
				treeLines = append(treeLines, sessionStyle.Render(line))
			}
		}
		treeLines = append(treeLines, "")
	}

	treeStr := strings.Join(treeLines, "\n")

	if m.showHist && len(m.snapshots) > 0 {
		// Build history panel
		var hlines []string
		for i, snap := range m.snapshots {
			marker := "  "
			if i == m.histIdx {
				marker = "▶ "
			}
			count := 0
			for _, ws := range snap.Tree {
				count += len(ws.Sessions)
			}
			hlines = append(hlines, fmt.Sprintf("%sv%-2d  %s  %d sessions",
				marker,
				snap.Version,
				snap.UpdatedAt.Format("2006-01-02 15:04"),
				count,
			))
		}
		histPanel := histPanelStyle.Render("History\n\n" + strings.Join(hlines, "\n"))
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, treeStr, "  ", histPanel))
	} else {
		b.WriteString(treeStr)
	}

	b.WriteString("\n")
	if m.message != "" {
		b.WriteString(dimStyle2.Render("  "+m.message) + "\n")
	}

	helpParts := []string{
		"↑/↓: navigate",
		"enter: open session",
		"a: open workspace",
		"A: open all",
		"h: history",
	}
	if m.showHist {
		helpParts = append(helpParts, "←/→: revision")
	}
	helpParts = append(helpParts, "q: quit")
	b.WriteString(helpStyle.Render("  " + strings.Join(helpParts, "  •  ")))

	return b.String()
}

// RunTUI starts the interactive TUI.
func RunTUI(nodes []WorkspaceNode, snapshots []Snapshot) error {
	model := newTUIModel(nodes, snapshots)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
