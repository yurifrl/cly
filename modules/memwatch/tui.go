package memwatch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
)

// ─── styles ────────────────────────────────────────────────────────────────

var (
	tuiTitleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("141")).Bold(true)
	tuiItemStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	tuiSelectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	tuiDimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	tuiSizeStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("117"))
	tuiWsStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("72"))
	tuiPidStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	tuiWarnStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	tuiErrStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	tuiOkStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	tuiOmpStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("176"))
	tuiBoxStyle      = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(0, 1).
				MarginLeft(2)
)

// ─── list item ─────────────────────────────────────────────────────────────

// agentRow is the common view projection of a pi or omp process.
type agentRow struct {
	kind      string // "pi" or "omp"
	pid       int
	rssKB     int64
	cwd       string
	label     string
	workspace string
	ref       string
	sessions  []string // pi only
}

func piRow(p PiProc) agentRow {
	return agentRow{
		kind: "pi", pid: p.PID, rssKB: p.RSSKB, cwd: p.CWD, label: p.Label,
		workspace: p.Workspace, ref: p.WorkspaceRef, sessions: p.SessionNames,
	}
}

func ompRow(p OmpProc) agentRow {
	return agentRow{
		kind: "omp", pid: p.PID, rssKB: p.RSSKB, cwd: p.CWD, label: p.Label,
		workspace: p.Workspace, ref: p.WorkspaceRef,
	}
}

type agentItem struct {
	row agentRow
}

func (i agentItem) FilterValue() string {
	return strings.Join([]string{
		i.row.kind,
		strconv.Itoa(i.row.pid),
		i.row.workspace,
		i.row.label,
		i.row.cwd,
		strings.Join(i.row.sessions, " "),
	}, " ")
}

// kindLabel renders the agent kind as a fixed-width, colored chip.
func kindLabel(kind string) string {
	s := fmt.Sprintf("%-3s", kind)
	if kind == "omp" {
		return tuiOmpStyle.Render(s)
	}
	return tuiDimStyle.Render(s)
}

func (i agentItem) headline() string {
	ws := i.row.workspace
	if ws == "" {
		ws = i.row.label
	}
	if ws == "" {
		ws = "?"
	}
	ref := i.row.ref
	if ref == "" {
		ref = "—"
	}
	name := strings.Join(i.row.sessions, " │ ")
	if name == "" {
		name = tuiDimStyle.Render("(no session)")
	}
	return fmt.Sprintf("%s  %s  %s  %s  %s  %s",
		tuiPidStyle.Render(fmt.Sprintf("%-7d", i.row.pid)),
		kindLabel(i.row.kind),
		tuiSizeStyle.Render(fmt.Sprintf("%10s", FormatSize(i.row.rssKB))),
		tuiWsStyle.Render(fmt.Sprintf("%-20s", truncate(ws, 20))),
		tuiDimStyle.Render(fmt.Sprintf("%-13s", ref)),
		truncate(name, 60),
	)
}

type agentDelegate struct{}

func (agentDelegate) Height() int                                { return 1 }
func (agentDelegate) Spacing() int                               { return 0 }
func (agentDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd    { return nil }
func (agentDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(agentItem)
	if !ok {
		return
	}
	if index == m.Index() {
		fmt.Fprint(w, tuiSelectedStyle.Render("▌ "+it.headline()))
	} else {
		fmt.Fprint(w, tuiItemStyle.Render("  "+it.headline()))
	}
}

// ─── prompt mode ───────────────────────────────────────────────────────────

type promptMode int

const (
	promptNone promptMode = iota
	promptSend
	promptKill
	promptKill9
)

// ─── view mode ─────────────────────────────────────────────────────────────

type viewMode int

const (
	viewPi viewMode = iota
	viewOmp
)

func (v viewMode) label() string {
	if v == viewOmp {
		return "omp"
	}
	return "pi"
}

// ─── messages ──────────────────────────────────────────────────────────────

type refreshedMsg struct {
	piProcs  []PiProc
	ompProcs []OmpProc
	sample   *Sample
	swapUsed float64
	swapTot  float64
	err      error
}

type actionDoneMsg struct {
	ok  bool
	msg string
}

// ─── model ─────────────────────────────────────────────────────────────────

type tuiModel struct {
	list       list.Model
	input      textinput.Model
	prompt     promptMode
	currentWS  string // CMUX_WORKSPACE_ID at startup, never killed
	currentPID int
	status     string
	statusErr  bool
	width      int
	height     int
	sample     *Sample
	swapUsed   float64
	swapTot    float64
	piProcs    []PiProc
	ompProcs   []OmpProc
	totalPiKB  int64
	totalOmpKB int64
	view       viewMode
	detail     bool
}

func initialTUIModel() tuiModel {
	l := list.New(nil, agentDelegate{}, 0, 18)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)

	ti := textinput.New()
	ti.Prompt = "› "
	ti.CharLimit = 1024

	return tuiModel{
		list:       l,
		input:      ti,
		currentWS:  os.Getenv("CMUX_WORKSPACE_ID"),
		currentPID: os.Getpid(),
	}
}

func (m tuiModel) Init() tea.Cmd { return refreshCmd() }

func refreshCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		piProcs, piErr := PiProcesses(ctx)
		ompProcs, ompErr := OMPProcesses(ctx)
		sample, _ := Read(ctx)
		swapUsed, swapTot := readSwap(ctx)
		return refreshedMsg{piProcs: piProcs, ompProcs: ompProcs, sample: sample,
			swapUsed: swapUsed, swapTot: swapTot, err: errors.Join(piErr, ompErr)}
	}
}

// readSwap parses `sysctl vm.swapusage`. Returns (used MB, total MB).
func readSwap(ctx context.Context) (float64, float64) {
	out, err := exec.CommandContext(ctx, "sysctl", "-n", "vm.swapusage").Output()
	if err != nil {
		return 0, 0
	}
	re := regexp.MustCompile(`total = ([0-9.]+)M\s+used = ([0-9.]+)M`)
	m := re.FindStringSubmatch(string(out))
	if len(m) != 3 {
		return 0, 0
	}
	tot, _ := strconv.ParseFloat(m[1], 64)
	used, _ := strconv.ParseFloat(m[2], 64)
	return used, tot
}

// itemsFromProcs builds the list-item slice for the active view.
func (m tuiModel) itemsFromProcs() []list.Item {
	items := []list.Item{}
	if m.view == viewPi {
		for _, p := range m.piProcs {
			items = append(items, agentItem{row: piRow(p)})
		}
	} else {
		for _, p := range m.ompProcs {
			items = append(items, agentItem{row: ompRow(p)})
		}
	}
	return items
}

func (m tuiModel) selected() (agentRow, bool) {
	it, ok := m.list.SelectedItem().(agentItem)
	if !ok {
		return agentRow{}, false
	}
	return it.row, true
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height-8)
		m.input.SetWidth(msg.Width - 4)
		return m, nil

	case refreshedMsg:
		if msg.err != nil {
			m.status = "refresh error: " + msg.err.Error()
			m.statusErr = true
			return m, nil
		}
		m.piProcs = msg.piProcs
		m.ompProcs = msg.ompProcs
		m.list.SetItems(m.itemsFromProcs())
		m.sample = msg.sample
		m.swapUsed = msg.swapUsed
		m.swapTot = msg.swapTot
		m.totalPiKB = 0
		m.totalOmpKB = 0
		for _, p := range msg.piProcs {
			m.totalPiKB += p.RSSKB
		}
		for _, p := range msg.ompProcs {
			m.totalOmpKB += p.RSSKB
		}
		m.status = m.agentStatus()
		m.statusErr = false
		return m, nil

	case actionDoneMsg:
		m.status = msg.msg
		m.statusErr = !msg.ok
		if msg.ok {
			return m, refreshCmd()
		}
		return m, nil

	case tea.KeyPressMsg:
		// Prompt-input mode (text or confirmation).
		if m.prompt != promptNone {
			switch msg.String() {
			case "esc":
				m.prompt = promptNone
				m.input.Blur()
				m.input.SetValue("")
				m.status = "cancelled"
				return m, nil
			case "enter":
				p, ok := m.selected()
				if !ok {
					m.prompt = promptNone
					m.input.Blur()
					return m, nil
				}
				val := strings.TrimSpace(m.input.Value())
				mode := m.prompt
				m.prompt = promptNone
				m.input.Blur()
				m.input.SetValue("")
				switch mode {
				case promptSend:
					if val == "" {
						return m, func() tea.Msg { return actionDoneMsg{ok: false, msg: "empty input"} }
					}
					return m, sendCmd(p, val)
				case promptKill:
					if strings.EqualFold(val, "y") || strings.EqualFold(val, "yes") {
						return m, killCmd(p, syscall.SIGTERM, m.currentPID)
					}
					return m, func() tea.Msg { return actionDoneMsg{ok: false, msg: "kill cancelled"} }
				case promptKill9:
					if strings.EqualFold(val, "y") || strings.EqualFold(val, "yes") {
						return m, killCmd(p, syscall.SIGKILL, m.currentPID)
					}
					return m, func() tea.Msg { return actionDoneMsg{ok: false, msg: "kill cancelled"} }
				}
				return m, nil
			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}

		// Normal-mode keys. Skip when the list is in filtering mode.
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.detail {
				m.detail = false
				return m, nil
			}
		case "r":
			m.status = "refreshing…"
			m.statusErr = false
			return m, refreshCmd()
		case "p":
			if m.view == viewPi {
				m.view = viewOmp
			} else {
				m.view = viewPi
			}
			m.list.SetItems(m.itemsFromProcs())
			m.status = m.agentStatus()
			m.statusErr = false
			return m, nil
		case "enter":
			if _, ok := m.selected(); !ok {
				return m, nil
			}
			m.detail = !m.detail
			return m, nil
		case "o":
			p, ok := m.selected()
			if !ok {
				return m, nil
			}
			return m, gotoCmd(p)
		case "s", "i":
			if _, ok := m.selected(); !ok {
				return m, nil
			}
			m.prompt = promptSend
			m.detail = false
			m.input.Placeholder = "text to send (Enter to confirm, Esc to cancel)"
			m.input.Focus()
			return m, textinput.Blink
		case "x", "d":
			p, ok := m.selected()
			if !ok {
				return m, nil
			}
			if p.pid == m.currentPID {
				return m, func() tea.Msg {
					return actionDoneMsg{ok: false, msg: "refusing to kill current session"}
				}
			}
			m.prompt = promptKill
			m.detail = false
			m.input.Placeholder = fmt.Sprintf("kill PID %d (SIGTERM)? type y to confirm", p.pid)
			m.input.Focus()
			return m, textinput.Blink
		case "X", "D":
			p, ok := m.selected()
			if !ok {
				return m, nil
			}
			if p.pid == m.currentPID {
				return m, func() tea.Msg {
					return actionDoneMsg{ok: false, msg: "refusing to kill current session"}
				}
			}
			m.prompt = promptKill9
			m.detail = false
			m.input.Placeholder = fmt.Sprintf("FORCE KILL PID %d (SIGKILL)? type y to confirm", p.pid)
			m.input.Focus()
			return m, textinput.Blink
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m tuiModel) View() tea.View {
	header := m.memHeader()
	colHdr := tuiDimStyle.Render(fmt.Sprintf("  %-7s  %-3s  %10s  %-20s  %-13s  %s",
		"PID", "KIND", "RSS", "WORKSPACE", "REF", "SESSION"))

	body := m.list.View()

	// Status / prompt / detail row.
	var bottom string
	switch {
	case m.prompt != promptNone:
		bottom = tuiBoxStyle.Render(m.input.View())
	case m.detail:
		bottom = m.detailView()
	default:
		statusStyle := tuiOkStyle
		if m.statusErr {
			statusStyle = tuiErrStyle
		}
		if m.status != "" {
			bottom = "  " + statusStyle.Render(m.status)
		}
	}

	help := tuiDimStyle.Render(
		"j/k move · g/G top/bot · / search · enter detail · o open · s send · x kill · X force-kill · p view pi/omp · r refresh · q quit",
	)

	out := strings.Join([]string{
		"",
		header,
		"",
		colHdr,
		body,
		bottom,
		"",
		"  " + help,
	}, "\n")
	v := tea.NewView(out)
	v.AltScreen = true
	return v
}

// ─── action commands ───────────────────────────────────────────────────────

// agentStatus summarizes the active view and the other kind's count.
func (m tuiModel) agentStatus() string {
	if m.view == viewPi {
		return fmt.Sprintf("%d pi instances · %d omp running", len(m.piProcs), len(m.ompProcs))
	}
	return fmt.Sprintf("%d omp instances · %d pi running", len(m.ompProcs), len(m.piProcs))
}

func (m tuiModel) memHeader() string {
	parts := []string{tuiTitleStyle.Render("memwatch")}
	if m.sample != nil && m.sample.FreePercent >= 0 {
		lvl := m.sample.PressureLvl
		lvlStyle := tuiOkStyle
		switch lvl {
		case "warn":
			lvlStyle = tuiWarnStyle
		case "critical":
			lvlStyle = tuiErrStyle
		}
		parts = append(parts,
			tuiDimStyle.Render("free ")+tuiSizeStyle.Render(fmt.Sprintf("%d%%", m.sample.FreePercent)),
			tuiDimStyle.Render("pressure ")+lvlStyle.Render(lvl),
		)
	}
	if m.swapTot > 0 {
		pct := m.swapUsed / m.swapTot * 100
		swapStyle := tuiOkStyle
		switch {
		case pct >= 75:
			swapStyle = tuiErrStyle
		case pct >= 50:
			swapStyle = tuiWarnStyle
		}
		parts = append(parts,
			tuiDimStyle.Render("swap ")+swapStyle.Render(
				fmt.Sprintf("%.1f/%.1f GB (%.0f%%)", m.swapUsed/1024, m.swapTot/1024, pct)),
		)
	}
	parts = append(parts, tuiDimStyle.Render("view ")+tuiWsStyle.Render(m.view.label()))
	if m.totalPiKB > 0 {
		parts = append(parts, tuiDimStyle.Render("pi ")+tuiSizeStyle.Render(FormatSize(m.totalPiKB)))
	}
	if m.totalOmpKB > 0 {
		parts = append(parts, tuiDimStyle.Render("omp ")+tuiOmpStyle.Render(FormatSize(m.totalOmpKB)))
	}
	return "  " + strings.Join(parts, tuiDimStyle.Render("  ·  "))
}

func (m tuiModel) detailView() string {
	p, ok := m.selected()
	if !ok {
		return "  " + tuiDimStyle.Render("no selection")
	}
	lines := []string{
		tuiTitleStyle.Render(fmt.Sprintf("Detail — PID %d", p.pid)),
		"",
		tuiDimStyle.Render("Kind        ") + kindLabel(p.kind),
		tuiDimStyle.Render("RSS         ") + tuiSizeStyle.Render(FormatSize(p.rssKB)),
		tuiDimStyle.Render("Workspace   ") + tuiWsStyle.Render(orDash(p.workspace)),
		tuiDimStyle.Render("Ref         ") + orDash(p.ref),
		tuiDimStyle.Render("Label       ") + orDash(p.label),
		tuiDimStyle.Render("CWD         ") + orDash(p.cwd),
	}
	if len(p.sessions) > 0 {
		lines = append(lines, tuiDimStyle.Render("Sessions    ")+strings.Join(p.sessions, "\n            "))
	} else {
		lines = append(lines, tuiDimStyle.Render("Sessions    ")+tuiDimStyle.Render("(none)"))
	}
	if p.pid == m.currentPID {
		lines = append(lines, "", tuiWarnStyle.Render("★ current session — protected from kill"))
	}
	return tuiBoxStyle.Render(strings.Join(lines, "\n"))
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func gotoCmd(p agentRow) tea.Cmd {
	return func() tea.Msg {
		if p.ref == "" {
			return actionDoneMsg{ok: false, msg: fmt.Sprintf("PID %d: no cmux workspace ref", p.pid)}
		}
		if _, err := exec.LookPath("cmux"); err != nil {
			return actionDoneMsg{ok: false, msg: "cmux binary not on PATH"}
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		out, err := exec.CommandContext(ctx, "cmux", "select-workspace", "--workspace", p.ref).CombinedOutput()
		if err != nil {
			return actionDoneMsg{ok: false, msg: fmt.Sprintf("select-workspace failed: %v: %s", err, strings.TrimSpace(string(out)))}
		}
		return actionDoneMsg{ok: true, msg: "switched to " + p.ref + " (" + p.workspace + ")"}
	}
}

func sendCmd(p agentRow, text string) tea.Cmd {
	return func() tea.Msg {
		if p.ref == "" {
			return actionDoneMsg{ok: false, msg: fmt.Sprintf("PID %d: no cmux workspace ref", p.pid)}
		}
		if _, err := exec.LookPath("cmux"); err != nil {
			return actionDoneMsg{ok: false, msg: "cmux binary not on PATH"}
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		// Send the text, then press Enter to submit.
		if out, err := exec.CommandContext(ctx, "cmux", "send", "--workspace", p.ref, text).CombinedOutput(); err != nil {
			return actionDoneMsg{ok: false, msg: fmt.Sprintf("send failed: %v: %s", err, strings.TrimSpace(string(out)))}
		}
		if out, err := exec.CommandContext(ctx, "cmux", "send-key", "--workspace", p.ref, "enter").CombinedOutput(); err != nil {
			return actionDoneMsg{ok: false, msg: fmt.Sprintf("send-key enter failed: %v: %s", err, strings.TrimSpace(string(out)))}
		}
		return actionDoneMsg{ok: true, msg: fmt.Sprintf("sent %q → %s", truncate(text, 40), p.ref)}
	}
}

func killCmd(p agentRow, sig syscall.Signal, currentPID int) tea.Cmd {
	return func() tea.Msg {
		if p.pid == currentPID {
			return actionDoneMsg{ok: false, msg: "refusing to kill current session"}
		}
		if err := syscall.Kill(p.pid, sig); err != nil {
			return actionDoneMsg{ok: false, msg: fmt.Sprintf("kill %d (%v): %v", p.pid, sig, err)}
		}
		return actionDoneMsg{ok: true, msg: fmt.Sprintf("sent %v to PID %d", sig, p.pid)}
	}
}

// ─── cobra ─────────────────────────────────────────────────────────────────

func newTUICmd() *cobra.Command {
	return &cobra.Command{
		Use:     "tui",
		Aliases: []string{"interactive", "i"},
		Short:   "Interactive TUI to navigate, search, and send commands to running pi/omp sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := tea.NewProgram(initialTUIModel())
			_, err := p.Run()
			return err
		},
	}
}
