package pitree

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Styles ────────────────────────────────────────────────────────────────────

var (
	wsStyle        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	curWsStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("215"))
	sessionStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	curSessStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("215")).Bold(true)
	hiddenStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Strikethrough(true)
	dimStyle2      = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	sizeStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	dateStyle2     = lipgloss.NewStyle().Foreground(lipgloss.Color("248"))
	msgStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("178"))
	searchStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("117"))
	histPanelStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99")).Padding(0, 0, 1, 2)
)

// ── Cursor ────────────────────────────────────────────────────────────────────

type cursor struct {
	ws   int
	sess int // -1 = workspace header row
}

// ── Filter ────────────────────────────────────────────────────────────────────

type filterMode int

const (
	filterNone   filterMode = iota
	filterSearch            // '/' text search active
)

// ── Hide key ──────────────────────────────────────────────────────────────────

type hideKey struct{ ws, sess int }

// ── Model ─────────────────────────────────────────────────────────────────────

type tuiModel struct {
	liveNodes []WorkspaceNode // original live tree (never overwritten)
	allNodes  []WorkspaceNode // current source (live or snapshot)
	nodes     []WorkspaceNode // current view (after filters)
	snapshots []Snapshot
	cur       cursor
	hidden    map[hideKey]bool // hidden sessions (sess=-1 hides whole workspace)

	filterMode  filterMode
	searchInput textinput.Model
	sinceHours  float64 // 0 = no filter; set from --since flag

	showHist    bool
	histIdx     int
	height      int
	width       int
	scrollOff   int // scroll offset for tree view
	message string
	quit    bool
}

func newTUIModel(nodes []WorkspaceNode, snapshots []Snapshot, sinceHours float64) tuiModel {
	ti := textinput.New()
	ti.Placeholder = "search workspace or session…"
	ti.CharLimit = 80

	m := tuiModel{
		liveNodes:   nodes,
		allNodes:    nodes,
		nodes:       nodes,
		snapshots:   snapshots,
		histIdx:     len(snapshots) - 1,
		cur:         cursor{ws: 0, sess: -1},
		hidden:      make(map[hideKey]bool),
		searchInput: ti,
		sinceHours:  sinceHours,
	}

	// If current tree is empty, fall back to last snapshot
	if len(nodes) == 0 && len(snapshots) > 0 {
		last := snapshots[len(snapshots)-1]
		m.allNodes = last.Tree
		m.liveNodes = last.Tree
		m.showHist = true
		m.message = fmt.Sprintf("no live sessions -- showing snapshot v%d", last.Version)
	}

	m.applyFilters()
	return m
}

func (m tuiModel) Init() tea.Cmd { return nil }

// ── Filtering ─────────────────────────────────────────────────────────────────

func (m *tuiModel) applyFilters() {
	query := strings.ToLower(strings.TrimSpace(m.searchInput.Value()))
	var out []WorkspaceNode
	for _, ws := range m.allNodes {
		wsMatch := query == "" || strings.Contains(strings.ToLower(ws.Name), query)
		var sessions []PiSession
		for _, s := range ws.Sessions {
			// since filter
			if m.sinceHours > 0 {
				t, err := time.Parse("2006-01-02 15:04", s.StartedAt)
				if err == nil && time.Since(t) > time.Duration(m.sinceHours*float64(time.Hour)) {
					continue
				}
			}
			// session-level hide
			si := sessionIndex(ws, s.SessionID)
			wi := workspaceIndex(m.allNodes, ws.Name)
			if m.hidden[hideKey{wi, si}] {
				continue
			}
			// search
			if query != "" && !wsMatch {
				if !strings.Contains(strings.ToLower(s.SessionID), query) &&
					!strings.Contains(strings.ToLower(s.StartedAt), query) {
					continue
				}
			}
			sessions = append(sessions, s)
		}
		// workspace-level hide
		wi := workspaceIndex(m.allNodes, ws.Name)
		if m.hidden[hideKey{wi, -1}] {
			continue
		}
		if len(sessions) > 0 {
			wsCopy := ws
			wsCopy.Sessions = sessions
			out = append(out, wsCopy)
		}
	}
	m.nodes = out
	if m.cur.ws >= len(m.nodes) {
		m.cur = cursor{ws: 0, sess: -1}
	}
}

func workspaceIndex(nodes []WorkspaceNode, name string) int {
	for i, n := range nodes {
		if n.Name == name {
			return i
		}
	}
	return -1
}

func sessionIndex(ws WorkspaceNode, id string) int {
	for i, s := range ws.Sessions {
		if s.SessionID == id {
			return i
		}
	}
	return -1
}

// ── Cursor movement ───────────────────────────────────────────────────────────

type pos struct{ ws, sess int }

func (m tuiModel) positions() []pos {
	var out []pos
	for wi, ws := range m.nodes {
		out = append(out, pos{wi, -1})
		for si := range ws.Sessions {
			out = append(out, pos{wi, si})
		}
	}
	return out
}

func (m tuiModel) cursorIndex() int {
	for i, p := range m.positions() {
		if p.ws == m.cur.ws && p.sess == m.cur.sess {
			return i
		}
	}
	return 0
}

func (m *tuiModel) moveCursor(delta int) {
	ps := m.positions()
	if len(ps) == 0 {
		return
	}
	idx := m.cursorIndex() + delta
	if idx < 0 {
		idx = 0
	}
	if idx >= len(ps) {
		idx = len(ps) - 1
	}
	m.cur.ws = ps[idx].ws
	m.cur.sess = ps[idx].sess
}

// ensureVisible adjusts scrollOff so the cursor line is visible.
func (m *tuiModel) ensureVisible() {
	curLine := m.cursorLineIndex()
	viewHeight := m.viewableHeight()
	if viewHeight <= 0 {
		return
	}
	if curLine < m.scrollOff {
		m.scrollOff = curLine
	}
	if curLine >= m.scrollOff+viewHeight {
		m.scrollOff = curLine - viewHeight + 1
	}
}

// cursorLineIndex returns which line in treeLines the cursor is on.
func (m tuiModel) cursorLineIndex() int {
	line := 0
	for wi, ws := range m.nodes {
		if m.cur.ws == wi && m.cur.sess == -1 {
			return line
		}
		line++ // workspace header
		for si := range ws.Sessions {
			if m.cur.ws == wi && m.cur.sess == si {
				return line
			}
			line++ // session line
		}
		line++ // blank line after workspace
	}
	return 0
}

// viewableHeight returns how many tree lines fit on screen.
func (m tuiModel) viewableHeight() int {
	// Reserve lines for: title, search bar, since badge, message, help bar
	reserved := 4
	if m.filterMode == filterSearch || m.searchInput.Value() != "" {
		reserved += 2
	}
	if m.sinceHours > 0 {
		reserved += 2
	}
	h := m.height - reserved
	if h < 3 {
		h = 3
	}
	return h
}

// ── Hide ──────────────────────────────────────────────────────────────────────

func (m *tuiModel) hideUnderCursor() {
	if m.cur.ws >= len(m.nodes) {
		return
	}
	ws := m.nodes[m.cur.ws]
	wi := workspaceIndex(m.allNodes, ws.Name)
	if wi < 0 {
		return
	}
	if m.cur.sess == -1 {
		m.hidden[hideKey{wi, -1}] = true
	} else {
		si := sessionIndex(m.allNodes[wi], ws.Sessions[m.cur.sess].SessionID)
		if si >= 0 {
			m.hidden[hideKey{wi, si}] = true
		}
	}
	m.applyFilters()
	m.message = "hidden (U to unhide all)"
}

func (m *tuiModel) unhideAll() {
	m.hidden = make(map[hideKey]bool)
	m.applyFilters()
	m.message = "all unhidden"
}

// ── Cursor session for open ───────────────────────────────────────────────────

func (m tuiModel) cursorSession() (WorkspaceNode, PiSession, bool) {
	if m.cur.ws >= len(m.nodes) {
		return WorkspaceNode{}, PiSession{}, false
	}
	// Only return a session if cursor is on a session row, not workspace header
	if m.cur.sess == -1 {
		return WorkspaceNode{}, PiSession{}, false
	}
	ws := m.nodes[m.cur.ws]
	if m.cur.sess >= 0 && m.cur.sess < len(ws.Sessions) {
		return ws, ws.Sessions[m.cur.sess], true
	}
	return WorkspaceNode{}, PiSession{}, false
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Search mode: route keys to text input
	if m.filterMode == filterSearch {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				m.filterMode = filterNone
				m.searchInput.Blur()
				m.applyFilters()
				m.message = ""
				return m, nil
			case "enter":
				m.filterMode = filterNone
				m.searchInput.Blur()
				m.applyFilters()
				m.message = ""
				return m, nil
			case "ctrl+c":
				m.quit = true
				return m, tea.Quit
			}
		}
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		m.applyFilters()
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quit = true
			return m, tea.Quit

		case "up", "k":
			if m.showHist {
				if m.histIdx > 0 {
					m.histIdx--
					m.allNodes = m.snapshots[m.histIdx].Tree
					m.cur = cursor{ws: 0, sess: -1}
					m.applyFilters()
				}
			} else {
				m.moveCursor(-1)
				m.ensureVisible()
			}
			m.message = ""

		case "down", "j":
			if m.showHist {
				if m.histIdx < len(m.snapshots)-1 {
					m.histIdx++
					m.allNodes = m.snapshots[m.histIdx].Tree
					m.cur = cursor{ws: 0, sess: -1}
					m.applyFilters()
				}
			} else {
				m.moveCursor(1)
				m.ensureVisible()
			}
			m.message = ""

		case "enter", "o":
			if m.showHist {
				m.showHist = false
				m.message = fmt.Sprintf("viewing snapshot v%d", m.snapshots[m.histIdx].Version)
				break
			}
			ws, sess, ok := m.cursorSession()
			if !ok {
				m.message = "nothing to open"
				break
			}
			// Remember where we are so we can focus back
			callerWs, callerSurf := callerSurface()
			m.message = "opening..."
			cmd, err := openSession(ws.Name, sess.SessionID, sess.FilePath)
			if err != nil {
				m.message = fmt.Sprintf("error: %v", err)
			} else {
				// Focus back to pi-tree's workspace/surface
				if callerWs != "" {
					_, _ = runCommand("cmux", "select-workspace", "--workspace", callerWs)
				}
				if callerSurf != "" {
					time.Sleep(200 * time.Millisecond)
					_, _ = runCommand("cmux", "focus-panel", "--panel", callerSurf)
				}
				m.message = fmt.Sprintf("✓ %s", cmd)
			}

		case "x":
			m.hideUnderCursor()

		case "U":
			m.unhideAll()

		case "/":
			m.filterMode = filterSearch
			m.searchInput.Focus()
			m.message = ""
			return m, textinput.Blink

		case "esc":
			if m.searchInput.Value() != "" {
				m.searchInput.SetValue("")
				m.applyFilters()
				m.message = "filter cleared"
			}

		case "h":
			m.showHist = !m.showHist
			if !m.showHist {
				// Restore live tree when closing history
				m.allNodes = m.liveNodes
				m.cur = cursor{ws: 0, sess: -1}
				m.applyFilters()
			}
			m.message = ""

		}
	}
	return m, nil
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m tuiModel) View() string {
	if m.quit {
		return ""
	}

	hiddenCount := len(m.hidden)

	var b strings.Builder

	// Title row
	title := "π session tree"
	if hiddenCount > 0 {
		title += dimStyle2.Render(fmt.Sprintf("  [%d hidden]", hiddenCount))
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")

	// Search bar
	if m.filterMode == filterSearch {
		b.WriteString(searchStyle.Render("  / ") + m.searchInput.View() + "\n\n")
	} else if m.searchInput.Value() != "" {
		b.WriteString(searchStyle.Render(fmt.Sprintf("  / %s", m.searchInput.Value())) +
			dimStyle2.Render("  (esc to clear)") + "\n\n")
	}

	// Since filter badge
	if m.sinceHours > 0 {
		label := fmt.Sprintf("%.0fh", m.sinceHours)
		if m.sinceHours < 1 {
			label = fmt.Sprintf("%.0fm", m.sinceHours*60)
		}
		b.WriteString(dimStyle2.Render(fmt.Sprintf("  ⏱  showing last %s\n\n", label)))
	}

	var treeLines []string
	for wi, ws := range m.nodes {
		isCurWs := m.cur.ws == wi && m.cur.sess == -1

		var wsPrefix string
		if isCurWs {
			wsPrefix = dimStyle2.Render("▶ ")
		} else {
			wsPrefix = "  "
		}

		if isCurWs {
			treeLines = append(treeLines, wsPrefix+curWsStyle.Render(ws.Name))
		} else {
			treeLines = append(treeLines, wsPrefix+wsStyle.Render(ws.Name))
		}

		for si, s := range ws.Sessions {
			isCur := m.cur.ws == wi && m.cur.sess == si

			branch := "├── "
			if si == len(ws.Sessions)-1 {
				branch = "└── "
			}

			var sessPrefix string
			if isCur {
				sessPrefix = "  " + dimStyle2.Render("▶ ")
			} else {
				sessPrefix = "    "
			}

			branchStr := dimStyle2.Render(branch)
			size := sizeStyle.Render(fmt.Sprintf("%7s", formatSize(s.SizeBytes)))
			date := dateStyle2.Render(s.StartedAt)

			var sidStr string
			if isCur {
				sidStr = curSessStyle.Render(s.SessionID)
			} else {
				sidStr = sessionStyle.Render(s.SessionID)
			}

			// Working directory suffix
			workDirStr := ""
			if s.FilePath != "" {
				workDir := sessionDirToWorkingDir(s.FilePath)
				if workDir != "" {
					workDirStr = dimStyle2.Render("  " + workDir)
				}
			}

			line := fmt.Sprintf("%s%s%s  %s  %s%s",
				sessPrefix, branchStr, sidStr, size, date, workDirStr)

			_ = hiddenStyle // keep import used
			treeLines = append(treeLines, line)
		}
		treeLines = append(treeLines, "")
	}

	if len(m.nodes) == 0 {
		treeLines = append(treeLines, dimStyle2.Render("  no sessions match"))
		treeLines = append(treeLines, "")
	}

	treeStr := ""
	if len(treeLines) > 0 {
		viewH := m.viewableHeight()
		end := m.scrollOff + viewH
		if end > len(treeLines) {
			end = len(treeLines)
		}
		start := m.scrollOff
		if start > len(treeLines) {
			start = len(treeLines)
		}
		treeStr = strings.Join(treeLines[start:end], "\n")
	}

	if m.showHist && len(m.snapshots) > 0 {
		var hlines []string
		for i, snap := range m.snapshots {
			marker := "  "
			if i == m.histIdx {
				marker = dimStyle2.Render("▶ ")
			}
			count := 0
			for _, ws := range snap.Tree {
				count += len(ws.Sessions)
			}
			hlines = append(hlines, fmt.Sprintf("%sv%-2d  %s  %d sessions",
				marker, snap.Version,
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
		b.WriteString(msgStyle.Render("  "+m.message) + "\n")
	}

	// Help bar
	var helpParts []string
	if m.filterMode == filterSearch {
		helpParts = []string{"enter/esc: done", "ctrl+c: quit"}
	} else {
		helpParts = []string{
			"↑/↓: move",
			"enter: open",
			"/: search",
			"x: hide",
		}
		if hiddenCount > 0 {
			helpParts = append(helpParts, "U: unhide all")
		}
		helpParts = append(helpParts, "h: history")
		if m.showHist {
			helpParts = append(helpParts, "↑/↓: revision")
		}
		helpParts = append(helpParts, "q: quit")
	}
	b.WriteString(helpStyle.Render("  " + strings.Join(helpParts, "  •  ")))

	return b.String()
}

// ── Open actions ──────────────────────────────────────────────────────────────

// sessionDirToWorkingDir converts a pi session directory name to the original working directory.
// e.g. "--Users-yuri-Workdir-Yuri-cly--" → "/Users/yuri/Workdir/Yuri/cly"
func sessionDirToWorkingDir(filePath string) string {
	if filePath == "" {
		return ""
	}
	dir := filepath.Dir(filePath)
	dirName := filepath.Base(dir)

	dirName = strings.TrimPrefix(dirName, "--")
	dirName = strings.TrimSuffix(dirName, "--")

	result := "/" + strings.ReplaceAll(dirName, "-", "/")
	return result
}

// resolveWorkspaceRef finds the current workspace ref by name from cmux.
func resolveWorkspaceRef(name string) string {
	out, err := runCommand("cmux", "list-workspaces")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		s := strings.TrimSpace(line)
		s = strings.TrimLeft(s, "* ")
		parts := strings.SplitN(s, "  ", 2)
		if len(parts) != 2 {
			continue
		}
		ref := strings.TrimSpace(parts[0])
		wsName := strings.TrimSpace(parts[1])
		wsName = strings.TrimSuffix(wsName, "  [selected]")
		wsName = strings.TrimSpace(wsName)
		if strings.EqualFold(wsName, name) {
			return ref
		}
	}
	return ""
}

// callerSurface returns the surface ref where this process is running.
func callerSurface() (wsRef string, surfRef string) {
	out, err := runCommand("cmux", "identify")
	if err != nil {
		return "", ""
	}
	// Quick parse — look for caller workspace_ref and surface_ref
	lines := strings.Split(string(out), "\n")
	inCaller := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, `"caller"`) {
			inCaller = true
			continue
		}
		if inCaller {
			if strings.Contains(trimmed, `"workspace_ref"`) {
				wsRef = extractJSONString(trimmed)
			}
			if strings.Contains(trimmed, `"surface_ref"`) {
				surfRef = extractJSONString(trimmed)
			}
			if strings.HasPrefix(trimmed, "}") {
				break
			}
		}
	}
	return wsRef, surfRef
}

func extractJSONString(line string) string {
	// "surface_ref" : "surface:35",
	idx := strings.Index(line, ": ")
	if idx == -1 {
		return ""
	}
	val := strings.TrimSpace(line[idx+2:])
	val = strings.Trim(val, `",`)
	return strings.TrimSpace(val)
}

// openSession opens a pi session in the matching workspace (or creates a new one).
func openSession(wsName, sessionID, filePath string) (string, error) {
	piCmd := fmt.Sprintf("pi --session %s", sessionID)
	workDir := sessionDirToWorkingDir(filePath)

	// Check if workspace already exists by name
	wsRef := resolveWorkspaceRef(wsName)
	if wsRef != "" {
		// Switch to existing workspace, create new tab, send pi command
		_, _ = runCommand("cmux", "select-workspace", "--workspace", wsRef)
		time.Sleep(300 * time.Millisecond)

		surfOut, err := runCommand("cmux", "new-surface", "--type", "terminal", "--workspace", wsRef)
		if err != nil {
			return "", fmt.Errorf("new-surface: %w", err)
		}
		var surfRef string
		for _, f := range strings.Fields(string(surfOut)) {
			if strings.HasPrefix(f, "surface:") {
				surfRef = f
				break
			}
		}
		if surfRef == "" {
			return "", fmt.Errorf("could not parse surface ref from: %s", string(surfOut))
		}

		time.Sleep(500 * time.Millisecond)
		_, err = runCommand("cmux", "send-panel", "--panel", surfRef, "--workspace", wsRef, piCmd+"\n")
		if err != nil {
			return "", fmt.Errorf("send-panel: %w", err)
		}

		return fmt.Sprintf("opened in workspace %q (%s) → %s", wsName, wsRef, surfRef), nil
	}

	// Workspace doesn't exist — create new one with working directory
	args := []string{"new-workspace", "--command", piCmd}
	cmd := fmt.Sprintf("cmux new-workspace --command %q", piCmd)
	if workDir != "" {
		args = append(args, "--working-directory", workDir)
		cmd += fmt.Sprintf(" --working-directory %s", workDir)
	}
	_, err := runCommand("cmux", args...)
	if err != nil {
		return cmd, fmt.Errorf("new-workspace: %w", err)
	}
	return cmd, nil
}

// ── Entry point ───────────────────────────────────────────────────────────────

func RunTUI(nodes []WorkspaceNode, snapshots []Snapshot, sinceHours float64) error {
	m := newTUIModel(nodes, snapshots, sinceHours)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
