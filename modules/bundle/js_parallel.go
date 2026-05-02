package bundle

import (
	"fmt"
	"os/exec"
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
	err    error
}

// Tick to poll progress.
type tickMsg struct{}

// parallelModel is the Bubbletea model for parallel JS installs.
type parallelModel struct {
	packages []string
	statuses []pkgStatus
	outputs  []string // final output line per package
	errors   []error

	spinner  spinner.Model
	progress progress.Model

	total     int
	completed int
	failed    int
	width     int

	resultCh chan pkgResultMsg
	done     bool
	force    bool
	verbose  bool
}

var (
	pkgStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	doneStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	failStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	pendingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
)

func newParallelModel(packages []string, force bool, verbose bool) parallelModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))

	p := progress.New(progress.WithDefaultBlend())
	p.SetWidth(50)

	return parallelModel{
		packages: packages,
		statuses: make([]pkgStatus, len(packages)),
		outputs:  make([]string, len(packages)),
		errors:   make([]error, len(packages)),
		spinner:  s,
		progress: p,
		total:    len(packages),
		resultCh: make(chan pkgResultMsg, len(packages)),
		force:    force,
		verbose:  verbose,
	}
}

func (m parallelModel) Init() tea.Cmd {
	// Launch all installs as goroutines
	var wg sync.WaitGroup
	for i, pkg := range m.packages {
		wg.Add(1)
		go func(idx int, p string) {
			defer wg.Done()
			out, err := runPnpmInstall(p, m.force)
			m.resultCh <- pkgResultMsg{index: idx, output: out, err: err}
		}(i, pkg)
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
	b.WriteString(titleStyle.Render(fmt.Sprintf("  Installing %d packages", m.total)))
	b.WriteString("\n\n")

	// Overall progress bar
	b.WriteString("  " + m.progress.View() + "\n")
	b.WriteString(pendingStyle.Render(fmt.Sprintf("  %d/%d completed", m.completed, m.total)))
	if m.failed > 0 {
		b.WriteString(failStyle.Render(fmt.Sprintf("  (%d failed)", m.failed)))
	}
	b.WriteString("\n\n")

	// Per-package status
	for i, pkg := range m.packages {
		base := extractJsBasePkg(pkg)
		switch m.statuses[i] {
		case pkgRunning:
			b.WriteString(fmt.Sprintf("  %s %s\n", m.spinner.View(), pkgStyle.Render(base)))
		case pkgDone:
			out := m.outputs[i]
			if out == "" {
				out = "installed"
			}
			b.WriteString(fmt.Sprintf("  %s %s %s\n",
				doneStyle.Render("✓"),
				pkgStyle.Render(base),
				pendingStyle.Render(out),
			))
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
				pkgStyle.Render(base),
				failStyle.Render(errMsg),
			))
		default:
			b.WriteString(fmt.Sprintf("  %s %s\n",
				pendingStyle.Render("○"),
				pendingStyle.Render(base),
			))
		}
	}

	if m.done {
		b.WriteString("\n")
		if m.failed == 0 {
			b.WriteString(doneStyle.Render("  All packages installed successfully!") + "\n")
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

// runPnpmInstall runs `pnpm add -g <pkg>` and returns combined output.
func runPnpmInstall(pkg string, force bool) (string, error) {
	args := []string{"add", "-g", pkg}
	if force {
		args = append(args, "--force")
	}
	cmd := exec.Command("pnpm", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("pnpm add failed: %s", strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

// summarizePnpmOutput extracts a useful one-liner from pnpm output.
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

// runParallelInstall runs the TUI for parallel package installation.
// Returns error only if the TUI itself fails; individual package errors are displayed.
func runParallelInstall(packages []string, force bool, verbose bool) error {
	m := newParallelModel(packages, force, verbose)
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
