package cmd

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/modules/scraper/browser"
)

var browserPort int

var BrowserCmd = &cobra.Command{
	Use:   "browser",
	Short: "Launch persistent browser for scraping",
	Long: `Launch a Chrome instance with remote debugging enabled.

This browser stays open for multiple scraping sessions, allowing you to:
- Solve CAPTCHA challenges manually once
- Reuse the same session across scrapes
- Debug scraping issues visually`,
	RunE: runBrowser,
}

func init() {
	BrowserCmd.Flags().IntVar(&browserPort, "port", 9222, "Remote debugging port")
}

type browserModel struct {
	status string
	ready  bool
	err    error
	ctrl   *browser.Controller
}

type browserReadyMsg struct{}
type browserErrorMsg struct{ err error }

func (m browserModel) Init() tea.Cmd {
	return launchBrowser(m.ctrl)
}

func (m browserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
	case browserReadyMsg:
		m.ready = true
		m.status = "Browser ready!"
	case browserErrorMsg:
		m.err = msg.err
		return m, tea.Quit
	}
	return m, nil
}

func (m browserModel) View() string {
	if m.err != nil {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(fmt.Sprintf("Error: %v\n", m.err))
	}

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")).Render("🌐 Browser Launcher")
	status := lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Render(m.status)

	if !m.ready {
		return fmt.Sprintf("%s\n\n%s", title, status)
	}

	instructions := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(
		"Instructions:\n" +
			"  1. Solve any CAPTCHA in the browser window\n" +
			"  2. Keep this terminal open\n" +
			"  3. Run scraper commands in another terminal\n\n" +
			"Press q or Ctrl+C to stop the browser",
	)

	success := lipgloss.NewStyle().Foreground(lipgloss.Color("34")).Render("✓ " + m.status)

	return fmt.Sprintf("%s\n\n%s\n\n%s", title, success, instructions)
}

func launchBrowser(ctrl *browser.Controller) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if err := ctrl.Launch(ctx); err != nil {
			return browserErrorMsg{err: err}
		}

		// Navigate to AliExpress
		if err := ctrl.NavigateToProduct(""); err == nil {
			// Construct URL manually since empty ID isn't valid
			// Just launch browser without navigation for now
		}

		return browserReadyMsg{}
	}
}

func runBrowser(cmd *cobra.Command, args []string) error {
	userDataDir, err := browser.GetDefaultUserDataDir()
	if err != nil {
		return fmt.Errorf("failed to get user data dir: %w", err)
	}

	ctrl := browser.NewController(browser.Options{
		DebugPort:   browserPort,
		Headless:    false,
		UserDataDir: userDataDir,
	})
	defer ctrl.Close()

	m := browserModel{
		status: fmt.Sprintf("Launching Chrome on port %d...", browserPort),
		ctrl:   ctrl,
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}
