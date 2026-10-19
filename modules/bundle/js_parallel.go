package bundle

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Status of a single package install.
type pkgStatus int

const (
	pkgPending pkgStatus = iota
	pkgRunning
	pkgDone
	pkgFailed
)

// Result of a single package install goroutine.
type pkgResultMsg struct {
	index  int
	output string
	note   string // fallback-registry notice; non-empty when fallback was used
	err    error
}

// Tick to poll progress.
type tickMsg struct{}

// parallelModel is the Bubbletea model for parallel JS installs.
type parallelModel struct {
	actions  []syncAction
	statuses []pkgStatus
	outputs  []string // final output line per package
	notes    []string // fallback-registry notice per package
	errors   []error

	spinner  spinner.Model
	progress progress.Model

	total     int
	completed int
	failed    int
	width     int

	resultCh  chan pkgResultMsg
	done      bool
	installFn func(pkg string) (string, string, error)
}

var (
	pkgStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	doneStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	failStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	pendingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	noteStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)

func newParallelModel(installFn func(pkg string) (string, string, error), actions []syncAction) parallelModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))

	p := progress.New(progress.WithDefaultBlend())
	p.SetWidth(50)

	return parallelModel{
		actions:   actions,
		statuses:  make([]pkgStatus, len(actions)),
		outputs:   make([]string, len(actions)),
		notes:     make([]string, len(actions)),
		errors:    make([]error, len(actions)),
		spinner:   s,
		progress:  p,
		total:     len(actions),
		resultCh:  make(chan pkgResultMsg, len(actions)),
		installFn: installFn,
	}
}

func (m parallelModel) Init() tea.Cmd {
	// Launch all installs as goroutines
	var wg sync.WaitGroup
	for i, action := range m.actions {
		wg.Add(1)
		go func(idx int, spec string) {
			defer wg.Done()
			out, note, err := m.installFn(spec)
			m.resultCh <- pkgResultMsg{index: idx, output: out, note: note, err: err}
		}(i, action.pkg)
	}

	// Close channel when all done
	go func() {
		wg.Wait()
		close(m.resultCh)
	}()

	// Mark all as running
	for i := range m.statuses {
		m.statuses[i] = pkgRunning
	}

	return tea.Batch(m.spinner.Tick, pollResults(m.resultCh))
}

func (m parallelModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		w := msg.Width - 8
		if w > 80 {
			w = 80
		}
		if w > 0 {
			m.progress.SetWidth(w)
		}
		return m, nil

	case pkgResultMsg:
		m.statuses[msg.index] = pkgDone
		m.outputs[msg.index] = summarizePnpmOutput(msg.output)
		m.notes[msg.index] = msg.note
		m.completed++
		if msg.err != nil {
			m.statuses[msg.index] = pkgFailed
			m.errors[msg.index] = msg.err
			m.failed++
		}

		var cmds []tea.Cmd
		pct := float64(m.completed) / float64(m.total)
		cmds = append(cmds, m.progress.SetPercent(pct))

		if m.completed >= m.total {
			m.done = true
			cmds = append(cmds, tea.Sequence(
				tea.Tick(500*time.Millisecond, func(_ time.Time) tea.Msg { return nil }), // 500ms pause
				tea.Quit,
			))
		} else {
			cmds = append(cmds, pollResults(m.resultCh))
		}
		return m, tea.Batch(cmds...)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case progress.FrameMsg:
		var cmd tea.Cmd
		m.progress, cmd = m.progress.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m parallelModel) View() tea.View {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(titleStyle.Render(fmt.Sprintf("  Syncing %d packages", m.total)))
	b.WriteString("\n\n")

	// Overall progress bar
	b.WriteString("  " + m.progress.View() + "\n")
	b.WriteString(pendingStyle.Render(fmt.Sprintf("  %d/%d completed", m.completed, m.total)))
	if m.failed > 0 {
		b.WriteString(failStyle.Render(fmt.Sprintf("  (%d failed)", m.failed)))
	}
	b.WriteString("\n\n")

	// Per-package status
	for i, action := range m.actions {
		switch m.statuses[i] {
		case pkgRunning:
			b.WriteString(fmt.Sprintf("  %s %s\n", m.spinner.View(), pkgStyle.Render(action.label())))
		case pkgDone:
			out := m.outputs[i]
			if out == "" {
				out = action.doneLabel()
			}
			b.WriteString(fmt.Sprintf("  %s %s %s\n",
				doneStyle.Render("✓"),
				pkgStyle.Render(action.base),
				pendingStyle.Render(out),
			))
			if note := m.notes[i]; note != "" {
				b.WriteString("        " + noteStyle.Render(note) + "\n")
			}
		case pkgFailed:
			errMsg := "failed"
			if m.errors[i] != nil {
				errMsg = m.errors[i].Error()
				// Trim to first line
				if idx := strings.Index(errMsg, "\n"); idx > 0 {
					errMsg = errMsg[:idx]
				}
			}
			b.WriteString(fmt.Sprintf("  %s %s %s\n",
				failStyle.Render("✗"),
				pkgStyle.Render(action.label()),
				failStyle.Render(errMsg),
			))
		default:
			b.WriteString(fmt.Sprintf("  %s %s\n",
				pendingStyle.Render("○"),
				pendingStyle.Render(action.label()),
			))
		}
	}

	if m.done {
		b.WriteString("\n")
		if m.failed == 0 {
			b.WriteString(doneStyle.Render("  "+syncSummary(m.actions, len(m.actions))) + "\n")
		} else {
			b.WriteString(failStyle.Render(fmt.Sprintf("  %d packages failed to install", m.failed)) + "\n")
		}
	}

	b.WriteString("\n")
	return tea.NewView(b.String())
}

// pollResults reads one result from the channel and returns it as a Cmd.
func pollResults(ch <-chan pkgResultMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

// summarizePnpmOutput extracts a useful one-liner from pnpm/npm output.
func summarizePnpmOutput(output string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	// Look for version info or "added" lines
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "added") || strings.Contains(line, "Already up to date") ||
			strings.Contains(line, "updated") || strings.Contains(line, "+") {
			return line
		}
	}
	// Last non-empty line
	for i := len(lines) - 1; i >= 0; i-- {
		if l := strings.TrimSpace(lines[i]); l != "" {
			return l
		}
	}
	return ""
}

// ttyAvailable reports whether an interactive terminal exists, so the
// Bubbletea progress TUI can run. Without one (scripts, CI) the caller
// falls back to serial installs.
func ttyAvailable() bool {
	f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return false
	}
	return f.Close() == nil
}

// runParallelInstall runs the TUI for parallel package installation.
// Returns error only if the TUI itself fails; individual package errors are displayed.
func runParallelInstall(installFn func(pkg string) (string, string, error), actions []syncAction) error {
	m := newParallelModel(installFn, actions)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	// Check if any packages failed
	fm := finalModel.(parallelModel)
	if fm.failed > 0 {
		return fmt.Errorf("%d packages failed to install", fm.failed)
	}
	return nil
}
