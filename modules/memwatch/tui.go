package memwatch

import (
	"context"
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
	tuiBoxStyle      = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(0, 1).
				MarginLeft(2)
)

// ─── list item ─────────────────────────────────────────────────────────────

type piItem struct {
	proc PiProc
}

func (i piItem) FilterValue() string {
	return strings.Join([]string{
		strconv.Itoa(i.proc.PID),
		i.proc.Workspace,
		i.proc.Label,
		i.proc.CWD,
		strings.Join(i.proc.SessionNames, " "),
	}, " ")
}

func (i piItem) headline() string {
	ws := i.proc.Workspace
	if ws == "" {
		ws = i.proc.Label
	}
	if ws == "" {
		ws = "?"
	}
	ref := i.proc.WorkspaceRef
	if ref == "" {
		ref = "—"
	}
	name := strings.Join(i.proc.SessionNames, " │ ")
	if name == "" {
		name = tuiDimStyle.Render("(no session)")
	}
	return fmt.Sprintf("%s  %s  %s  %s  %s",
		tuiPidStyle.Render(fmt.Sprintf("%-7d", i.proc.PID)),
		tuiSizeStyle.Render(fmt.Sprintf("%10s", FormatSize(i.proc.RSSKB))),
		tuiWsStyle.Render(fmt.Sprintf("%-20s", truncate(ws, 20))),
		tuiDimStyle.Render(fmt.Sprintf("%-13s", ref)),
		truncate(name, 60),
	)
}

type piDelegate struct{}

func (piDelegate) Height() int                             { return 1 }
func (piDelegate) Spacing() int                            { return 0 }
func (piDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (piDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(piItem)
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

// ─── messages ──────────────────────────────────────────────────────────────

type refreshedMsg struct {
	procs    []PiProc
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
	totalPiKB  int64
	detail     bool
}

func initialTUIModel() tuiModel {
	l := list.New(nil, piDelegate{}, 0, 18)
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
		procs, err := PiProcesses(ctx)
		sample, _ := Read(ctx)
		swapUsed, swapTot := readSwap(ctx)
		return refreshedMsg{procs: procs, sample: sample, swapUsed: swapUsed, swapTot: swapTot, err: err}
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

// itemsFromProcs builds the list-item slice, marking the current pi PID with a hint.
func (m tuiModel) itemsFromProcs(procs []PiProc) []list.Item {
	items := make([]list.Item, 0, len(procs))
	for _, p := range procs {
		items = append(items, piItem{proc: p})
	}
	return items
}

func (m tuiModel) selected() (PiProc, bool) {
	it, ok := m.list.SelectedItem().(piItem)
	if !ok {
		return PiProc{}, false
	}
	return it.proc, true
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
		m.list.SetItems(m.itemsFromProcs(msg.procs))
		m.sample = msg.sample
		m.swapUsed = msg.swapUsed
		m.swapTot = msg.swapTot
		var total int64
		for _, p := range msg.procs {
			total += p.RSSKB
		}
		m.totalPiKB = total
		m.status = fmt.Sprintf("%d pi instances", len(msg.procs))
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
			if p.PID == m.currentPID {
				return m, func() tea.Msg {
					return actionDoneMsg{ok: false, msg: "refusing to kill current session"}
				}
			}
			m.prompt = promptKill
			m.detail = false
			m.input.Placeholder = fmt.Sprintf("kill PID %d (SIGTERM)? type y to confirm", p.PID)
			m.input.Focus()
			return m, textinput.Blink
		case "X", "D":
			p, ok := m.selected()
			if !ok {
				return m, nil
			}
			if p.PID == m.currentPID {
				return m, func() tea.Msg {
					return actionDoneMsg{ok: false, msg: "refusing to kill current session"}
				}
			}
			m.prompt = promptKill9
			m.detail = false
			m.input.Placeholder = fmt.Sprintf("FORCE KILL PID %d (SIGKILL)? type y to confirm", p.PID)
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
	colHdr := tuiDimStyle.Render(fmt.Sprintf("  %-7s  %10s  %-20s  %-13s  %s",
		"PID", "RSS", "WORKSPACE", "REF", "SESSION"))

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
		"j/k move · g/G top/bot · / search · enter detail · o open · s send · x kill · X force-kill · r refresh · q quit",
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
	if m.totalPiKB > 0 {
		parts = append(parts, tuiDimStyle.Render("pi ")+tuiSizeStyle.Render(FormatSize(m.totalPiKB)))
	}
	return "  " + strings.Join(parts, tuiDimStyle.Render("  ·  "))
}

func (m tuiModel) detailView() string {
	p, ok := m.selected()
	if !ok {
		return "  " + tuiDimStyle.Render("no selection")
	}
	lines := []string{
		tuiTitleStyle.Render(fmt.Sprintf("Detail — PID %d", p.PID)),
		"",
		tuiDimStyle.Render("RSS         ") + tuiSizeStyle.Render(FormatSize(p.RSSKB)),
		tuiDimStyle.Render("Workspace   ") + tuiWsStyle.Render(orDash(p.Workspace)),
		tuiDimStyle.Render("Ref         ") + orDash(p.WorkspaceRef),
		tuiDimStyle.Render("Label       ") + orDash(p.Label),
		tuiDimStyle.Render("CWD         ") + orDash(p.CWD),
	}
	if len(p.SessionNames) > 0 {
		lines = append(lines, tuiDimStyle.Render("Sessions    ")+strings.Join(p.SessionNames, "\n            "))
	} else {
		lines = append(lines, tuiDimStyle.Render("Sessions    ")+tuiDimStyle.Render("(none)"))
	}
	if p.PID == m.currentPID {
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

func gotoCmd(p PiProc) tea.Cmd {
	return func() tea.Msg {
		if p.WorkspaceRef == "" {
			return actionDoneMsg{ok: false, msg: fmt.Sprintf("PID %d: no cmux workspace ref", p.PID)}
		}
		if _, err := exec.LookPath("cmux"); err != nil {
			return actionDoneMsg{ok: false, msg: "cmux binary not on PATH"}
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		out, err := exec.CommandContext(ctx, "cmux", "select-workspace", "--workspace", p.WorkspaceRef).CombinedOutput()
		if err != nil {
			return actionDoneMsg{ok: false, msg: fmt.Sprintf("select-workspace failed: %v: %s", err, strings.TrimSpace(string(out)))}
		}
		return actionDoneMsg{ok: true, msg: "switched to " + p.WorkspaceRef + " (" + p.Workspace + ")"}
	}
}

func sendCmd(p PiProc, text string) tea.Cmd {
	return func() tea.Msg {
		if p.WorkspaceRef == "" {
			return actionDoneMsg{ok: false, msg: fmt.Sprintf("PID %d: no cmux workspace ref", p.PID)}
		}
		if _, err := exec.LookPath("cmux"); err != nil {
			return actionDoneMsg{ok: false, msg: "cmux binary not on PATH"}
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		// Send the text, then press Enter to submit.
		if out, err := exec.CommandContext(ctx, "cmux", "send", "--workspace", p.WorkspaceRef, text).CombinedOutput(); err != nil {
			return actionDoneMsg{ok: false, msg: fmt.Sprintf("send failed: %v: %s", err, strings.TrimSpace(string(out)))}
		}
		if out, err := exec.CommandContext(ctx, "cmux", "send-key", "--workspace", p.WorkspaceRef, "enter").CombinedOutput(); err != nil {
			return actionDoneMsg{ok: false, msg: fmt.Sprintf("send-key enter failed: %v: %s", err, strings.TrimSpace(string(out)))}
		}
		return actionDoneMsg{ok: true, msg: fmt.Sprintf("sent %q → %s", truncate(text, 40), p.WorkspaceRef)}
	}
}

func killCmd(p PiProc, sig syscall.Signal, currentPID int) tea.Cmd {
	return func() tea.Msg {
		if p.PID == currentPID {
			return actionDoneMsg{ok: false, msg: "refusing to kill current session"}
		}
		if err := syscall.Kill(p.PID, sig); err != nil {
			return actionDoneMsg{ok: false, msg: fmt.Sprintf("kill %d (%v): %v", p.PID, sig, err)}
		}
		return actionDoneMsg{ok: true, msg: fmt.Sprintf("sent %v to PID %d", sig, p.PID)}
	}
}

// ─── cobra ─────────────────────────────────────────────────────────────────

func newTUICmd() *cobra.Command {
	return &cobra.Command{
		Use:     "tui",
		Aliases: []string{"interactive", "i"},
		Short:   "Interactive TUI to navigate, search, and send commands to running pi sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := tea.NewProgram(initialTUIModel())
			_, err := p.Run()
			return err
		},
	}
}
