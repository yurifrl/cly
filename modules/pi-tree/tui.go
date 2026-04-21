package pitree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ── Styles ────────────────────────────────────────────────────────────────────

var (
	wsStyle        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	curWsStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("215"))
	closedWsStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	sessionStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	curSessStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("215")).Bold(true)
	closedSessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	closedSizeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	closedDateStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	nameStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("183")).Italic(true)
	closedNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Italic(true)
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

// ── Model ─────────────────────────────────────────────────────────────────────

type tuiModel struct {
	liveNodes []WorkspaceNode // original live tree (never overwritten)
	allNodes  []WorkspaceNode // current source (live or snapshot)
	nodes     []WorkspaceNode // current view (after filters)
	snapshots []Snapshot
	cur       cursor

	openWS      map[string]bool   // workspace names currently open in cmux
	openSess    map[string]bool   // session IDs currently active (most recent per workspace)
	filterMode  filterMode
	searchInput textinput.Model
	sinceHours  float64 // 0 = no filter; set from --since flag

	showHist     bool
	histIdx      int  // -1 = Latest (live), 0..n-1 = snapshot index
	hideAutoSave bool // when true, autosaved snapshots are hidden from history (default: true)
	height       int
	width        int
	scrollOff    int // scroll offset for tree view
	message      string
	quit         bool

	pendingExec  string // command to exec after TUI exits (replaces process)
}

// isSnapVisible returns whether a snapshot at index i should be shown in history.
func (m tuiModel) isSnapVisible(i int) bool {
	if !m.hideAutoSave {
		return true
	}
	if i < 0 || i >= len(m.snapshots) {
		return true
	}
	return !m.snapshots[i].AutoSave
}

// nextVisibleSnap returns the next visible snapshot index moving from cur in direction dir (+1 or -1).
// Returns -2 if none found.
func (m tuiModel) nextVisibleSnap(cur, dir int) int {
	for i := cur + dir; i >= 0 && i < len(m.snapshots); i += dir {
		if m.isSnapVisible(i) {
			return i
		}
	}
	return -2 // sentinel: nothing found
}

// lastVisibleSnap returns the highest-index visible snapshot, or -2 if none.
func (m tuiModel) lastVisibleSnap() int {
	for i := len(m.snapshots) - 1; i >= 0; i-- {
		if m.isSnapVisible(i) {
			return i
		}
	}
	return -2
}

// firstVisibleSnap returns the lowest-index visible snapshot, or -2 if none.
func (m tuiModel) firstVisibleSnap() int {
	for i := 0; i < len(m.snapshots); i++ {
		if m.isSnapVisible(i) {
			return i
		}
	}
	return -2
}

func newTUIModel(nodes []WorkspaceNode, snapshots []Snapshot, sinceHours float64) tuiModel {
	ti := textinput.New()
	ti.Placeholder = "search workspace or session…"
	ti.CharLimit = 80

	// Always start with live view
	histIdx := -1

	// Build open sets from live nodes
	openWS := make(map[string]bool)
	openSess := make(map[string]bool)
	for _, ws := range nodes {
		openWS[ws.Name] = true
		// Most recent session per workspace is the active one
		if len(ws.Sessions) > 0 {
			openSess[ws.Sessions[0].SessionID] = true
		}
	}

	// If user previously selected a snapshot, show that version's tree
	displayNodes := nodes
	if histIdx >= 0 && histIdx < len(snapshots) {
		displayNodes = snapshots[histIdx].Tree
	}

	m := tuiModel{
		liveNodes:   nodes,
		allNodes:    displayNodes,
		nodes:       displayNodes,
		snapshots:    snapshots,
		histIdx:      histIdx,
		cur:          cursor{ws: 0, sess: 0},
		openWS:       openWS,
		openSess:     openSess,
		searchInput:  ti,
		sinceHours:   sinceHours,
		hideAutoSave: true,

	}

	// If current tree is empty, fall back to last snapshot
	if len(nodes) == 0 && len(snapshots) > 0 {
		snap := snapshots[len(snapshots)-1]
		m.histIdx = len(snapshots) - 1
		m.allNodes = snap.Tree
		m.liveNodes = snap.Tree
		m.showHist = true
		m.message = fmt.Sprintf("no live sessions -- showing snapshot from %s", fmtSnapTime(snap))
	}

	m.applyFilters()
	return m
}

// refreshMsg is sent when a live rescan completes.
type refreshMsg struct{ nodes []WorkspaceNode }

func scanLive() tea.Msg {
	nodes, _ := ScanTree()
	return refreshMsg{nodes: nodes}
}

func (m tuiModel) Init() tea.Cmd { return scanLive }

// ── Filtering ─────────────────────────────────────────────────────────────────

func (m *tuiModel) applyFilters() {
	query := strings.ToLower(strings.TrimSpace(m.searchInput.Value()))
	var out []WorkspaceNode
	for _, ws := range m.allNodes {
		wsMatch := query == "" || strings.Contains(strings.ToLower(ws.Name), query)
		var sessions []PiSession
		for _, s := range ws.Sessions {
			// only show open sessions (live view only)
			if m.histIdx == -1 && !s.IsOpen {
				continue
			}
			// since filter
			if m.sinceHours > 0 {
				t, err := time.Parse("2006-01-02 15:04", s.StartedAt)
				if err == nil && time.Since(t) > time.Duration(m.sinceHours*float64(time.Hour)) {
					continue
				}
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
		if len(sessions) > 0 {
			wsCopy := ws
			wsCopy.Sessions = sessions
			out = append(out, wsCopy)
		}
	}
	m.nodes = out

	if m.cur.ws >= len(m.nodes) {
		m.cur = cursor{ws: 0, sess: 0}
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

// ensureVisible adjusts scrollOff so the cursor line stays visible with context.
func (m *tuiModel) ensureVisible() {
	curLine := m.cursorLineIndex()
	viewHeight := m.viewableHeight()
	if viewHeight <= 0 {
		return
	}

	// Scroll margin: start scrolling before cursor hits the edge
	margin := 3
	if viewHeight < 10 {
		margin = 1
	}

	// When scrolling up, include the workspace header above the cursor
	topLine := curLine
	if m.cur.sess >= 0 {
		headerLine := curLine - m.cur.sess - 1
		if headerLine >= 0 {
			topLine = headerLine
		}
	}

	// Scroll up: keep margin above the workspace header
	if topLine < m.scrollOff+margin {
		m.scrollOff = topLine - margin
		if m.scrollOff < 0 {
			m.scrollOff = 0
		}
	}
	// Scroll down: keep margin below the cursor
	if curLine > m.scrollOff+viewHeight-margin-1 {
		m.scrollOff = curLine - viewHeight + margin + 1
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
	// Reserve lines for: title, search bar, since badge, message, help bar, detail panel
	reserved := 10 // 4 base + 6 for detail panel (always reserved for stable layout)
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

// previewHistSelection updates the background tree to match the current history selection.
func (m *tuiModel) previewHistSelection() {
	if m.histIdx == -1 {
		m.allNodes = m.liveNodes
	} else if m.histIdx >= 0 && m.histIdx < len(m.snapshots) {
		m.allNodes = m.snapshots[m.histIdx].Tree
	}
	m.cur = cursor{ws: 0, sess: 0}
	m.scrollOff = 0
	m.applyFilters()
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Search mode: route keys to text input
	if m.filterMode == filterSearch {
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
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
	case refreshMsg:
		m.liveNodes = msg.nodes
		m.openWS = make(map[string]bool)
		m.openSess = make(map[string]bool)
		for _, ws := range msg.nodes {
			m.openWS[ws.Name] = true
			if len(ws.Sessions) > 0 {
				m.openSess[ws.Sessions[0].SessionID] = true
			}
		}
		if m.histIdx == -1 {
			m.allNodes = m.liveNodes
			m.applyFilters()
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width

	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quit = true
			return m, tea.Quit

		case "up", "k":
			if m.showHist {
				if m.histIdx == -1 {
					// already at Latest (top), do nothing
				} else {
					// try to move to a newer visible snapshot
					next := m.nextVisibleSnap(m.histIdx, +1)
					if next != -2 {
						m.histIdx = next
					} else {
						// no more visible snapshots above → go to Latest
						m.histIdx = -1
					}
				}
				m.previewHistSelection()
			} else {
				m.moveCursor(-1)
				m.ensureVisible()
			}
			m.message = ""

		case "down", "j":
			if m.showHist {
				if m.histIdx == -1 {
					// Latest → newest visible snapshot
					if idx := m.lastVisibleSnap(); idx != -2 {
						m.histIdx = idx
					}
				} else {
					// move to older visible snapshot
					next := m.nextVisibleSnap(m.histIdx, -1)
					if next != -2 {
						m.histIdx = next
					}
				}
				m.previewHistSelection()
			} else {
				m.moveCursor(1)
				m.ensureVisible()
			}
			m.message = ""


		case "enter":
			if m.showHist {
				m.showHist = false
				if m.histIdx == -1 {
					m.allNodes = m.liveNodes
					m.cur = cursor{ws: 0, sess: 0}
					m.applyFilters()
					SaveLastHistIdx(-1)
					m.message = fmt.Sprintf("viewing latest (%d sessions)", countSessions(m.liveNodes))
				} else if m.histIdx >= 0 && m.histIdx < len(m.snapshots) {
					snap := m.snapshots[m.histIdx]
					m.allNodes = snap.Tree
					m.cur = cursor{ws: 0, sess: 0}
					m.applyFilters()
					SaveLastHistIdx(snap.Version)
					m.message = fmt.Sprintf("viewing %s (%d sessions)", fmtSnapTime(snap), countSessions(snap.Tree))
				}
				break
			}
			// Open session in existing or new workspace
			ws, sess, ok := m.cursorSession()
			if !ok {
				m.message = "nothing to open"
				break
			}
			workDir := sessionDirToWorkingDir(sess.FilePath)
			cwd, _ := os.Getwd()
			if workDir != "" && cwd != "" && strings.EqualFold(cwd, workDir) {
				m.pendingExec = fmt.Sprintf("pi --session %s", sess.SessionID)
				m.quit = true
				return m, tea.Quit
			}
			m.message = "opening..."
			cmd, err := openSession(ws.Name, sess.SessionID, sess.FilePath)
			if err != nil {
				m.message = fmt.Sprintf("error: %v", err)
			} else {
				_, callerSurf := callerSurface()
				if callerSurf != "" {
					_, _ = runCommand("cmux", "close-surface", "--surface", callerSurf)
				}
				m.quit = true
				m.message = fmt.Sprintf("[OK] %s", cmd)
				return m, tea.Quit
			}

		case "d":
			if m.showHist && m.histIdx >= 0 && len(m.snapshots) > 1 {
				ver := m.snapshots[m.histIdx].Version
				timeStr := fmtSnapTime(m.snapshots[m.histIdx])
				if err := DeleteSnapshot(ver); err != nil {
					m.message = fmt.Sprintf("delete error: %v", err)
					break
				}
				all, _ := LoadSnapshots(); m.snapshots = ActiveSnapshots(all)
				if m.histIdx >= len(m.snapshots) {
					m.histIdx = len(m.snapshots) - 1
				}
				m.message = fmt.Sprintf("deleted %s", timeStr)
				break
			} else if m.showHist && m.histIdx == -1 {
				m.message = "can't delete latest"
				break
			} else if m.showHist {
				m.message = "can't delete last snapshot"
				break
			}
			ws, sess, ok := m.cursorSession()
			if !ok {
				m.message = "nothing to open"
				break
			}
			// Check if we're already in the session's working directory
			workDir := sessionDirToWorkingDir(sess.FilePath)
			cwd, _ := os.Getwd()
			if workDir != "" && cwd != "" && strings.EqualFold(cwd, workDir) {
				// Same dir — just exec pi locally and quit TUI
				m.pendingExec = fmt.Sprintf("pi --session %s", sess.SessionID)
				m.quit = true
				return m, tea.Quit
			}
			// Different dir — open in target workspace and stay there
			m.message = "opening..."
			cmd, err := openSession(ws.Name, sess.SessionID, sess.FilePath)
			if err != nil {
				m.message = fmt.Sprintf("error: %v", err)
			} else {
				// Close the current cmux surface/pane so we follow the new session
				_, callerSurf := callerSurface()
				if callerSurf != "" {
					_, _ = runCommand("cmux", "close-surface", "--surface", callerSurf)
				}
				m.quit = true
				m.message = fmt.Sprintf("✓ %s", cmd)
				return m, tea.Quit
			}

		case "o":
			if m.showHist {
				break
			}
			ws, sess, ok := m.cursorSession()
			if !ok {
				m.message = "nothing to open"
				break
			}
			// Open in target workspace and close this window (follow)
			workDir := sessionDirToWorkingDir(sess.FilePath)
			cwd, _ := os.Getwd()
			if workDir != "" && cwd != "" && strings.EqualFold(cwd, workDir) {
				m.pendingExec = fmt.Sprintf("pi --session %s", sess.SessionID)
				m.quit = true
				return m, tea.Quit
			}
			m.message = "opening..."
			cmd, err := openSession(ws.Name, sess.SessionID, sess.FilePath)
			if err != nil {
				m.message = fmt.Sprintf("error: %v", err)
			} else {
				// Close the current cmux surface/pane so we follow the new session
				_, callerSurf := callerSurface()
				if callerSurf != "" {
					_, _ = runCommand("cmux", "close-surface", "--surface", callerSurf)
				}
				m.quit = true
				m.message = fmt.Sprintf("✓ %s", cmd)
				return m, tea.Quit
			}

		case "O":
			if m.showHist {
				break
			}
			ws, sess, ok := m.cursorSession()
			if !ok {
				m.message = "nothing to open"
				break
			}
			// Open in target workspace but keep this window (stay)
			m.message = "opening..."
			cmd, err := openSession(ws.Name, sess.SessionID, sess.FilePath)
			if err != nil {
				m.message = fmt.Sprintf("error: %v", err)
			} else {
				m.message = fmt.Sprintf("✓ %s (staying here)", cmd)
			}



		case "s":
			// Save current view as a new snapshot version
			if len(m.allNodes) == 0 {
				m.message = "nothing to save"
				break
			}
			snap, isNew, err := Upsert(m.allNodes, true)
			if err != nil {
				m.message = fmt.Sprintf("save error: %v", err)
			} else if isNew {
				all, _ := LoadSnapshots(); m.snapshots = ActiveSnapshots(all)
				m.histIdx = len(m.snapshots) - 1
				SaveLastHistIdx(snap.Version)
				m.message = fmt.Sprintf("📸 saved %s (%d sessions)", fmtSnapTime(snap), countSessions(m.allNodes))
			} else {
				m.message = "no changes to save"
			}

		case "/":
			m.filterMode = filterSearch
			m.searchInput.Focus()
			m.message = ""
			return m, textinput.Blink

		case "esc":
			if m.showHist {
				// Cancel picker — commit selection (same as h-close)
				m.showHist = false
				if m.histIdx >= 0 && m.histIdx < len(m.snapshots) {
					m.allNodes = m.snapshots[m.histIdx].Tree
					SaveLastHistIdx(m.snapshots[m.histIdx].Version)
				} else {
					m.allNodes = m.liveNodes
					SaveLastHistIdx(-1)
				}
				m.cur = cursor{ws: 0, sess: 0}
				m.applyFilters()
				m.message = ""
			} else if m.searchInput.Value() != "" {
				m.searchInput.SetValue("")
				m.applyFilters()
				m.message = "filter cleared"
			}

		case "r":
			m.message = "refreshing..."
			return m, scanLive

		case "h":
			m.showHist = !m.showHist
			if !m.showHist {
				// Keep the selected snapshot's tree (don't reset to live)
				if m.histIdx >= 0 && m.histIdx < len(m.snapshots) {
					m.allNodes = m.snapshots[m.histIdx].Tree
					SaveLastHistIdx(m.snapshots[m.histIdx].Version)
				} else {
					m.allNodes = m.liveNodes
				}
				m.cur = cursor{ws: 0, sess: 0}
				m.applyFilters()
			}
			m.message = ""


		}
	}
	return m, nil
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m tuiModel) View() tea.View {
	v := tea.View{}
	v.AltScreen = true
	if m.quit {
		return v
	}

	var b strings.Builder

	// Title row with current version indicator
	title := "π session tree"
	if m.histIdx == -1 {
		title += "  " + lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("46")).Render("● LIVE")
	} else if m.histIdx >= 0 && m.histIdx < len(m.snapshots) {
		snap := m.snapshots[m.histIdx]
		autoTag := ""
		if snap.AutoSave {
			autoTag = dimStyle2.Render(" [auto]")
		}
		title += "  " + lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196")).Render(
			fmt.Sprintf("◆ SNAPSHOT %s (%d sessions)", fmtSnapTime(snap), countSessions(snap.Tree)),
		) + autoTag
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
			sessOpen := s.IsOpen

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

			// Compact line: name then date
			displayName := s.SessionName
			if displayName == "" {
				if len(s.SessionID) > 8 {
					displayName = s.SessionID[:8]
				} else {
					displayName = s.SessionID
				}
			}

			var nameStr string
			if isCur {
				nameStr = curSessStyle.Render(displayName)
			} else if sessOpen {
				nameStr = nameStyle.Render(displayName)
			} else {
				nameStr = closedNameStyle.Render(displayName)
			}

			line := fmt.Sprintf("%s%s%s",
				sessPrefix, branchStr, nameStr)

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
		visible := treeLines[start:end]
		// Pad to fill the full viewable area so overlay composites correctly
		for len(visible) < viewH {
			visible = append(visible, "")
		}
		treeStr = strings.Join(visible, "\n")
	}

	// Build the complete page content (tree + message + help)
	b.WriteString(treeStr)
	b.WriteString("\n")
	if m.message != "" {
		b.WriteString(msgStyle.Render("  "+m.message) + "\n")
	}

	// Help bar
	var helpParts []string
	if m.filterMode == filterSearch {
		helpParts = []string{"esc: done", "ctrl+c: quit"}
	} else {
		helpParts = []string{
			"↑/↓: move",
			"o: open & follow",
			"O: open & stay",
			"/: search",
			"s: save",
			"r: refresh",
		}
		helpParts = append(helpParts, "h: history")
		if m.showHist {
			helpParts = append(helpParts, "↑/↓: revision")
			helpParts = append(helpParts, "d: delete")
		}
		helpParts = append(helpParts, "q: quit")
	}
	b.WriteString(helpStyle.Render("  " + strings.Join(helpParts, "  •  ")))

	pageContent := b.String()

	// Fill the full screen as background
	w := m.width
	h := m.height
	if w < 80 {
		w = 80
	}
	if h < 24 {
		h = 24
	}
	bg := lipgloss.Place(w, h, lipgloss.Top, lipgloss.Left, pageContent)

	if m.showHist {
		var hlines []string
		// Latest (live) entry
		latestMarker := "  "
		if m.histIdx == -1 {
			latestMarker = dimStyle2.Render("▶ ")
		}
		hlines = append(hlines, fmt.Sprintf("%sLatest  (live, %d sessions)",
			latestMarker, countSessions(m.liveNodes),
		))
		// Render snapshots newest-first (reverse order)
		for i := len(m.snapshots) - 1; i >= 0; i-- {
			if !m.isSnapVisible(i) {
				continue
			}
			snap := m.snapshots[i]
			marker := "  "
			if i == m.histIdx {
				marker = dimStyle2.Render("▶ ")
			}
			count := countSessions(snap.Tree)
			autoTag := ""
			if snap.AutoSave {
				autoTag = dimStyle2.Render(" [auto]")
			}
			hlines = append(hlines, fmt.Sprintf("%s%s  %d sessions%s",
				marker, fmtSnapTime(snap), count, autoTag,
			))
		}
		autoLabel := ""
		if m.hideAutoSave {
			autoLabel = dimStyle2.Render("  (hiding auto)")
		}
		histPanel := histPanelStyle.Render("History  (esc to close)" + autoLabel + "\n\n" + strings.Join(hlines, "\n"))
		panelW := lipgloss.Width(histPanel)
		panelH := lipgloss.Height(histPanel)
		px := (w - panelW) / 2
		py := (h - panelH) / 2
		if px < 0 {
			px = 0
		}
		if py < 0 {
			py = 0
		}

		root := lipgloss.NewLayer(bg)
		modal := lipgloss.NewLayer(histPanel).X(px).Y(py).Z(1)
		layers := []*lipgloss.Layer{root, modal}
		if detail := m.renderDetailPanel(); detail != "" {
			detailLayer := lipgloss.NewLayer(detail).X(1).Y(h - lipgloss.Height(detail) - 1).Z(2)
			layers = append(layers, detailLayer)
		}
		comp := lipgloss.NewCompositor(layers...)
		v.SetContent(comp.Render())
	} else {
		root := lipgloss.NewLayer(bg)
		if detail := m.renderDetailPanel(); detail != "" {
			detailLayer := lipgloss.NewLayer(detail).X(1).Y(h - lipgloss.Height(detail) - 1).Z(1)
			comp := lipgloss.NewCompositor(root, detailLayer)
			v.SetContent(comp.Render())
		} else {
			v.SetContent(bg)
		}
	}

	return v
}

// renderDetailPanel returns a styled detail box for the currently selected session.
// Returns empty string if no session is selected.
func (m tuiModel) renderDetailPanel() string {
	_, sess, ok := m.cursorSession()
	if !ok {
		return ""
	}

	var lines []string
	lines = append(lines, dimStyle2.Render("id   ")+sessionStyle.Render(sess.SessionID))
	lines = append(lines, dimStyle2.Render("size ")+sizeStyle.Render(formatSize(sess.SizeBytes)))
	lines = append(lines, dimStyle2.Render("date ")+dateStyle2.Render(sess.StartedAt))
	if sess.FilePath != "" {
		workDir := sessionDirToWorkingDir(sess.FilePath)
		if workDir != "" {
			lines = append(lines, dimStyle2.Render("path ")+dimStyle2.Render(workDir))
		}
	}

	detailStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)

	return detailStyle.Render(strings.Join(lines, "\n"))
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

	if dirName == "" {
		return ""
	}

	// The encoding replaces / with - but directory names can also contain -.
	// Use backtracking to find a real path on disk.
	parts := strings.Split(dirName, "-")
	result := resolveEncodedPath(parts)
	if result != "" {
		return result
	}

	// Fallback: naive replacement (all dashes become /)
	return "/" + strings.ReplaceAll(dirName, "-", "/")
}

// resolveEncodedPath tries all possible splits of dash-separated parts
// into path segments, returning the first that exists on disk.
func resolveEncodedPath(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	return resolveEncodedPathRec(parts, "")
}

func resolveEncodedPathRec(parts []string, prefix string) string {
	if len(parts) == 0 {
		// Check if the full path is a directory
		if info, err := os.Stat(prefix); err == nil && info.IsDir() {
			return prefix
		}
		return ""
	}
	// Try joining increasingly more parts as one segment name
	for i := 1; i <= len(parts); i++ {
		segment := strings.Join(parts[:i], "-")
		candidate := prefix + "/" + segment
		// If there are remaining parts, this must be a valid directory prefix
		if i < len(parts) {
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				if result := resolveEncodedPathRec(parts[i:], candidate); result != "" {
					return result
				}
			}
		} else {
			// Last segment — this is the final path
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				return candidate
			}
		}
	}
	return ""
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
		sendCmd := piCmd
		if workDir != "" {
			sendCmd = fmt.Sprintf("cd %s && %s", workDir, piCmd)
		}
		_, err = runCommand("cmux", "send-panel", "--panel", surfRef, "--workspace", wsRef, sendCmd+"\n")
		if err != nil {
			return "", fmt.Errorf("send-panel: %w", err)
		}

		return fmt.Sprintf("opened in workspace %q (%s) → %s", wsName, wsRef, surfRef), nil
	}

	// Workspace doesn't exist — create new one with working directory
	// cmux new-workspace only supports --command, so we cd first
	fullCmd := piCmd
	if workDir != "" {
		fullCmd = fmt.Sprintf("cd %s && %s", workDir, piCmd)
	}
	args := []string{"new-workspace", "--command", fullCmd}
	cmd := fmt.Sprintf("cmux new-workspace --command %q", fullCmd)
	_, err := runCommand("cmux", args...)
	if err != nil {
		return cmd, fmt.Errorf("new-workspace: %w", err)
	}
	return cmd, nil
}

// ── Entry point ───────────────────────────────────────────────────────────────

// stripAnsi removes ANSI escape sequences from a string.
func RunTUI(nodes []WorkspaceNode, snapshots []Snapshot, sinceHours float64) error {
	m := newTUIModel(nodes, snapshots, sinceHours)
	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return err
	}
	// If a local exec was requested, replace this process with pi
	if fm, ok := result.(tuiModel); ok && fm.pendingExec != "" {
		binary, err := exec.LookPath("pi")
		if err != nil {
			return fmt.Errorf("pi not found: %w", err)
		}
		args := strings.Fields(fm.pendingExec)
		return syscall.Exec(binary, args, os.Environ())
	}
	return nil
}
