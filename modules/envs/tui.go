package envs

import (
	"context"
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7DD3FC"))
	mutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#86EFAC"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FCA5A5"))
)

type resultMsg fetchResult

type model struct {
	ctx       context.Context
	profile   string
	opBinary  string
	tokens    sessions
	results   <-chan fetchResult
	total     int
	complete  int
	variables int
	logs      []string
	fields    []Field
	spinner   spinner.Model
	launchctl bool
	fish      bool
	err       error
}

func newModel(ctx context.Context, config Config, profile, opBinary string, tokens sessions, launchctl, fish bool) model {
	indicator := spinner.New()
	indicator.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7DD3FC"))
	return model{
		ctx:       ctx,
		profile:   profile,
		opBinary:  opBinary,
		tokens:    tokens,
		results:   fetchAll(ctx, opBinary, config.Secrets, tokens),
		total:     len(config.Secrets),
		spinner:   indicator,
		launchctl: launchctl,
		fish:      fish,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, waitForResult(m.results))
}

func (m model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.KeyPressMsg:
		if message.String() == "ctrl+c" || message.String() == "q" {
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var command tea.Cmd
		m.spinner, command = m.spinner.Update(message)
		return m, command
	case resultMsg:
		m.complete++
		if message.Err != nil {
			m.logs = append(m.logs, errorStyle.Render("✗ ")+message.Secret.Name+mutedStyle.Render("  "+message.Err.Error()))
		} else {
			selected := resolveEval(m.ctx, SelectFields(message.Fields, m.profile))
			m.fields = append(m.fields, selected...)
			m.variables += len(selected)
			details := fmt.Sprintf("  %d variables", len(selected))
			m.logs = append(m.logs, successStyle.Render("✓ ")+message.Secret.Name+mutedStyle.Render(details))
		}
		if m.complete == m.total {
			m.logs = append(m.logs, successStyle.Render("✓ ")+fmt.Sprintf("Loaded %d variables from %d items", m.variables, m.total))
			if m.launchctl {
				if err := setLaunchctlEnvs(m.fields); err != nil {
					m.logs = append(m.logs, errorStyle.Render("✗ launchctl: ")+err.Error())
				} else {
					m.logs = append(m.logs, successStyle.Render("✓ ")+fmt.Sprintf("Injected %d vars via launchctl setenv", len(m.fields)))
				}
			}
			return m, tea.Quit
		}
		return m, waitForResult(m.results)
	}
	return m, nil
}

func (m model) View() tea.View {
	header := titleStyle.Render("1Password environment loader") + "  " + mutedStyle.Render("profile: "+m.profile)
	activity := m.spinner.View() + " " + mutedStyle.Render(fmt.Sprintf("Fetching items in parallel · %d/%d items", m.complete, m.total))
	body := strings.Join(m.logs, "\n")
	if body == "" {
		body = mutedStyle.Render("Waiting for 1Password…")
	}
	return tea.NewView(header + "\n" + activity + "\n\n" + body)
}

func (m *model) runPlain() error {
	for m.complete < m.total {
		result := <-m.results
		m.complete++
		if result.Err != nil {
			fmt.Fprintf(os.Stderr, "✗ %s: %v\n", result.Secret.Name, result.Err)
			continue
		}
		selected := resolveEval(m.ctx, SelectFields(result.Fields, m.profile))
		m.fields = append(m.fields, selected...)
		m.variables += len(selected)
	}
	writeOutput(os.Stdout, m.fields, m.fish)
	if !m.fish {
		fmt.Fprintf(os.Stderr, "✓ Loaded %d variables from %d items\n", m.variables, m.total)
	}
	if m.launchctl {
		if err := setLaunchctlEnvs(m.fields); err != nil {
			return fmt.Errorf("launchctl: %w", err)
		}
		fmt.Fprintf(os.Stderr, "✓ Injected %d vars via launchctl setenv\n", len(m.fields))
	}
	return nil
}

func waitForResult(results <-chan fetchResult) tea.Cmd {
	return func() tea.Msg { return resultMsg(<-results) }
}

func fishLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "\\'") + "'"
}

func envQuote(value string) string {
	if !strings.ContainsAny(value, " \t\n'\"\\$`!#&|;(){}<>?") {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func writeOutput(w *os.File, fields []Field, fish bool) {
	for _, field := range fields {
		// Prefixed version based on section
		prefix := "GENERAL_"
		if field.Section != "" {
			prefix = strings.ToUpper(field.Section) + "_"
		}
		if fish {
			fmt.Fprintf(w, "set -gx %s %s\n", prefix+field.Label, fishLiteral(field.Value))
			fmt.Fprintf(w, "set -gx %s %s\n", field.Label, fishLiteral(field.Value))
		} else {
			fmt.Fprintf(w, "%s%s=%s\n", prefix, field.Label, envQuote(field.Value))
			fmt.Fprintf(w, "%s=%s\n", field.Label, envQuote(field.Value))
		}
	}
}
