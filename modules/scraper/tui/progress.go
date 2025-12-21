package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yurifrl/cly/modules/scraper/browser"
)

// ProductStatus represents the status of a product scrape
type ProductStatus int

const (
	StatusPending ProductStatus = iota
	StatusScraping
	StatusDone
	StatusFailed
)

// ProductItem represents a product being scraped
type ProductItem struct {
	ID               string
	Status           ProductStatus
	Error            string
	Elapsed          time.Duration
	StartedAt        time.Time
	CurrentExtractor string
}

// LogEntry represents a log message
type LogEntry struct {
	Level     string
	Timestamp time.Time
	Message   string
}

// BrowserStatus represents browser connection status
type BrowserStatus struct {
	Connected bool
	Message   string
}

// ControlMsg is sent from TUI to scraper goroutine
type ControlMsg struct {
	Type string // "pause", "resume", "skip", "stop", "retry"
	Data interface{}
}

// BrowserState represents browser lifecycle state
type BrowserState int

const (
	BrowserNotStarted BrowserState = iota
	BrowserStarting
	BrowserReady
	BrowserError
)

// ScrapingState represents scraping lifecycle state
type ScrapingState int

const (
	ScrapingIdle ScrapingState = iota
	ScrapingActive
	ScrapingPaused
	ScrapingComplete
)

// DashboardModel is the Bubbletea model for scraping dashboard
type DashboardModel struct {
	// Product tracking
	products []ProductItem
	current  int

	// UI components
	progress    progress.Model
	productList viewport.Model
	logView     viewport.Model
	help        help.Model
	keymap      keymap

	// Stats
	avgTime       time.Duration
	eta           time.Duration
	totalElapsed  time.Duration
	successCount  int
	failedCount   int
	startTime     time.Time

	// Browser management
	browserState    BrowserState
	browserPID      int
	browserCtrl     *browser.Controller
	externalBrowser bool

	// Scraping state
	scrapingState ScrapingState

	// Logs
	logs    []LogEntry
	maxLogs int

	// Dimensions
	width  int
	height int

	// Control
	done        bool
	autoStart   bool
	controlChan chan<- ControlMsg
}

type keymap struct {
	pause         key.Binding
	skip          key.Binding
	retry         key.Binding
	killBrowser   key.Binding
	restartBrowser key.Binding
	startScraping key.Binding
	scrollUp      key.Binding
	scrollDown    key.Binding
	quit          key.Binding
}

func newKeymap() keymap {
	return keymap{
		pause: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "pause/resume"),
		),
		skip: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "skip current"),
		),
		retry: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "retry failed"),
		),
		killBrowser: key.NewBinding(
			key.WithKeys("k"),
			key.WithHelp("k", "kill browser"),
		),
		restartBrowser: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "restart browser"),
		),
		startScraping: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "start"),
		),
		scrollUp: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "scroll up"),
		),
		scrollDown: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "scroll down"),
		),
		quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// NewDashboardModel creates a new dashboard model
func NewDashboardModel(productIDs []string, controlChan chan<- ControlMsg) DashboardModel {
	items := make([]ProductItem, len(productIDs))
	for i, id := range productIDs {
		items[i] = ProductItem{
			ID:     id,
			Status: StatusPending,
		}
	}

	prog := progress.New(progress.WithDefaultGradient())

	return DashboardModel{
		products:      items,
		current:       -1,
		progress:      prog,
		productList:   viewport.New(80, 10),
		logView:       viewport.New(80, 5),
		help:          help.New(),
		keymap:        newKeymap(),
		maxLogs:       10,
		startTime:     time.Now(),
		browserState:  BrowserNotStarted,
		scrapingState: ScrapingIdle,
		controlChan:   controlChan,
	}
}

// SetBrowserController sets the browser controller
func (m *DashboardModel) SetBrowserController(ctrl *browser.Controller) {
	m.browserCtrl = ctrl
}

// SetExternalBrowser marks if using external browser
func (m *DashboardModel) SetExternalBrowser(external bool) {
	m.externalBrowser = external
	if external {
		m.browserState = BrowserReady
	}
}

// SetAutoStart sets auto-start flag
func (m *DashboardModel) SetAutoStart(auto bool) {
	m.autoStart = auto
}

// Messages
type ProductStartMsg struct{ Index int }
type ProductDoneMsg struct {
	Index   int
	Elapsed time.Duration
}
type ProductFailMsg struct {
	Index int
	Error string
}
type AllDoneMsg struct{}
type BrowserStatusMsg struct {
	Connected bool
	Message   string
}
type ExtractorProgressMsg struct {
	ProductIndex int
	Extractor    string
}
type LogMsg struct {
	Level   string
	Message string
}
type StatsUpdateMsg struct {
	AvgTime      time.Duration
	ETA          time.Duration
	TotalElapsed time.Duration
}
type TickMsg time.Time

// Browser control messages
type StartBrowserMsg struct{}
type BrowserStartedMsg struct{ PID int }
type BrowserErrorMsg struct{ Error string }
type KillBrowserMsg struct{}
type RestartBrowserMsg struct{}
type StartScrapingMsg struct{}

func (m DashboardModel) Init() tea.Cmd {
	cmds := []tea.Cmd{tickCmd()}

	// Auto-start browser if flag set and not using external
	if m.autoStart && !m.externalBrowser && m.browserCtrl != nil {
		cmds = append(cmds, m.startBrowserCmd())
	}

	return tea.Batch(cmds...)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (m *DashboardModel) startBrowserCmd() tea.Cmd {
	return func() tea.Msg {
		m.addLog("INFO", "Starting browser on port 9222...")
		ctx := context.Background()
		if err := m.browserCtrl.Launch(ctx); err != nil {
			return BrowserErrorMsg{Error: err.Error()}
		}
		return BrowserStartedMsg{PID: 0}
	}
}

func (m *DashboardModel) killBrowserCmd() tea.Cmd {
	return func() tea.Msg {
		if m.browserCtrl != nil {
			m.browserCtrl.Close()
		}
		return KillBrowserMsg{}
	}
}

func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Allocate space
		headerHeight := 3
		statsHeight := 2
		footerHeight := 3
		logHeight := 6

		productListHeight := m.height - headerHeight - statsHeight - footerHeight - logHeight - 4

		m.productList.Width = m.width - 4
		m.productList.Height = productListHeight

		m.logView.Width = m.width - 4
		m.logView.Height = logHeight

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.quit):
			m.done = true
			// Kill browser if we own it
			if !m.externalBrowser && m.browserCtrl != nil {
				m.browserCtrl.Close()
			}
			if m.controlChan != nil {
				m.controlChan <- ControlMsg{Type: "stop"}
			}
			return m, tea.Quit

		case key.Matches(msg, m.keymap.startScraping):
			// Start browser if not started
			if m.browserState == BrowserNotStarted && !m.externalBrowser {
				m.browserState = BrowserStarting
				m.addLog("INFO", "Starting browser...")
				return m, m.startBrowserCmd()
			}
			// Start scraping if browser ready
			if m.browserState == BrowserReady && m.scrapingState == ScrapingIdle {
				m.scrapingState = ScrapingActive
				if m.controlChan != nil {
					m.controlChan <- ControlMsg{Type: "start"}
				}
				m.addLog("INFO", "Starting scraping...")
			}

		case key.Matches(msg, m.keymap.pause):
			if m.scrapingState == ScrapingActive {
				m.scrapingState = ScrapingPaused
				if m.controlChan != nil {
					m.controlChan <- ControlMsg{Type: "pause"}
					m.addLog("INFO", "Scraping paused")
				}
			} else if m.scrapingState == ScrapingPaused {
				m.scrapingState = ScrapingActive
				if m.controlChan != nil {
					m.controlChan <- ControlMsg{Type: "resume"}
					m.addLog("INFO", "Scraping resumed")
				}
			}

		case key.Matches(msg, m.keymap.skip):
			if m.current >= 0 && m.current < len(m.products) && m.scrapingState == ScrapingActive {
				if m.controlChan != nil {
					m.controlChan <- ControlMsg{Type: "skip"}
					m.addLog("INFO", "Skipped product: "+m.products[m.current].ID)
				}
			}

		case key.Matches(msg, m.keymap.killBrowser):
			if !m.externalBrowser && m.browserCtrl != nil {
				m.addLog("INFO", "Killing browser...")
				return m, m.killBrowserCmd()
			}

		case key.Matches(msg, m.keymap.restartBrowser):
			if !m.externalBrowser && m.browserCtrl != nil {
				m.addLog("INFO", "Restarting browser...")
				// Kill first
				m.browserCtrl.Close()
				m.browserState = BrowserStarting
				// Pause scraping
				if m.scrapingState == ScrapingActive {
					m.scrapingState = ScrapingPaused
				}
				return m, m.startBrowserCmd()
			}

		case key.Matches(msg, m.keymap.retry):
			var failedIDs []string
			for _, p := range m.products {
				if p.Status == StatusFailed {
					failedIDs = append(failedIDs, p.ID)
				}
			}
			if len(failedIDs) > 0 && m.controlChan != nil {
				m.controlChan <- ControlMsg{Type: "retry", Data: failedIDs}
				m.addLog("INFO", fmt.Sprintf("Retrying %d failed products", len(failedIDs)))
			}

		case key.Matches(msg, m.keymap.scrollUp):
			m.productList.LineUp(1)

		case key.Matches(msg, m.keymap.scrollDown):
			m.productList.LineDown(1)
		}

	case TickMsg:
		m.totalElapsed = time.Since(m.startTime)
		cmds = append(cmds, tickCmd())

	case BrowserStartedMsg:
		m.browserState = BrowserReady
		m.browserPID = msg.PID
		m.addLog("INFO", "Browser ready!")

		// Auto-start scraping if flag set
		if m.autoStart && m.scrapingState == ScrapingIdle {
			m.scrapingState = ScrapingActive
			if m.controlChan != nil {
				m.controlChan <- ControlMsg{Type: "start"}
			}
		}

	case BrowserErrorMsg:
		m.browserState = BrowserError
		m.addLog("ERROR", "Browser error: "+msg.Error)

	case KillBrowserMsg:
		m.browserState = BrowserNotStarted
		m.browserPID = 0
		m.addLog("INFO", "Browser killed")

	case ProductStartMsg:
		m.current = msg.Index
		m.products[msg.Index].Status = StatusScraping
		m.products[msg.Index].StartedAt = time.Now()

	case ProductDoneMsg:
		m.products[msg.Index].Status = StatusDone
		m.products[msg.Index].Elapsed = msg.Elapsed
		m.successCount++
		m.updateStats()

	case ProductFailMsg:
		m.products[msg.Index].Status = StatusFailed
		m.products[msg.Index].Error = msg.Error
		m.failedCount++
		m.addLog("ERROR", fmt.Sprintf("Failed %s: %s", m.products[msg.Index].ID, msg.Error))
		m.updateStats()

	case ExtractorProgressMsg:
		if msg.ProductIndex >= 0 && msg.ProductIndex < len(m.products) {
			m.products[msg.ProductIndex].CurrentExtractor = msg.Extractor
		}

	case BrowserStatusMsg:
		if !msg.Connected {
			m.browserState = BrowserError
			m.addLog("WARN", "Browser: "+msg.Message)
		} else {
			m.browserState = BrowserReady
		}

	case LogMsg:
		m.addLog(msg.Level, msg.Message)

	case StatsUpdateMsg:
		m.avgTime = msg.AvgTime
		m.eta = msg.ETA
		m.totalElapsed = msg.TotalElapsed

	case AllDoneMsg:
		m.done = true
		m.addLog("INFO", "Scraping complete!")
		return m, tea.Quit
	}

	// Update viewports
	var cmd tea.Cmd
	m.productList, cmd = m.productList.Update(msg)
	cmds = append(cmds, cmd)

	m.logView, cmd = m.logView.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m DashboardModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	sections := []string{
		m.renderHeader(),
		m.renderStats(),
		m.renderProductList(),
		m.renderLogs(),
		m.renderFooter(),
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m DashboardModel) renderHeader() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("212")).
		Render("🌐 AliExpress Scraper Dashboard")

	var statusIcon string
	var statusText string
	var statusColor lipgloss.Color

	switch m.browserState {
	case BrowserNotStarted:
		statusIcon = "○"
		statusText = "Not started"
		statusColor = lipgloss.Color("241")
	case BrowserStarting:
		statusIcon = "⏳"
		statusText = "Starting..."
		statusColor = lipgloss.Color("220")
	case BrowserReady:
		statusIcon = "✓"
		statusText = "Connected"
		statusColor = lipgloss.Color("34")
	case BrowserError:
		statusIcon = "✗"
		statusText = "Error"
		statusColor = lipgloss.Color("196")
	}

	status := lipgloss.NewStyle().
		Foreground(statusColor).
		Render(fmt.Sprintf("[Browser: %s %s]", statusIcon, statusText))

	header := lipgloss.JoinHorizontal(lipgloss.Top, title, strings.Repeat(" ", m.width-50), status)

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(lipgloss.Color("241")).
		Width(m.width - 2).
		Render(header)
}

func (m DashboardModel) renderStats() string {
	completed := m.successCount + m.failedCount
	total := len(m.products)
	percent := 0.0
	if total > 0 {
		percent = float64(completed) / float64(total) * 100
	}

	line1 := fmt.Sprintf("Stats: %d/%d complete (%.0f%%) | Success: %d | Failed: %d",
		completed, total, percent, m.successCount, m.failedCount)

	avgTimeStr := "N/A"
	if m.avgTime > 0 {
		avgTimeStr = fmt.Sprintf("%.1fs", m.avgTime.Seconds())
	}

	etaStr := "Calculating..."
	if m.eta > 0 {
		etaStr = formatDuration(m.eta)
	}

	line2 := fmt.Sprintf("Avg Time: %s/product | ETA: %s | Total: %s",
		avgTimeStr, etaStr, formatDuration(m.totalElapsed))

	stateIndicator := ""
	if m.scrapingState == ScrapingPaused {
		stateIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render(" [PAUSED]")
	}

	content := line1 + "\n" + line2 + stateIndicator

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("33")).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(lipgloss.Color("241")).
		Width(m.width - 2).
		Padding(0, 1).
		Render(content)
}

func (m DashboardModel) renderProductList() string {
	var lines []string

	for i, p := range m.products {
		var line string
		prefix := "  "
		if i == m.current {
			prefix = "→ "
		}

		switch p.Status {
		case StatusPending:
			line = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(
				fmt.Sprintf("%s○ %s - Pending", prefix, p.ID),
			)
		case StatusScraping:
			elapsed := time.Since(p.StartedAt)
			substep := ""
			if p.CurrentExtractor != "" {
				substep = fmt.Sprintf(" - %s", p.CurrentExtractor)
			}
			line = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render(
				fmt.Sprintf("%s⏳ %s (%.1fs...)%s", prefix, p.ID, elapsed.Seconds(), substep),
			)
		case StatusDone:
			line = lipgloss.NewStyle().Foreground(lipgloss.Color("34")).Render(
				fmt.Sprintf("%s✓ %s (%.1fs) - Complete", prefix, p.ID, p.Elapsed.Seconds()),
			)
		case StatusFailed:
			line = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(
				fmt.Sprintf("%s✗ %s - %s", prefix, p.ID, p.Error),
			)
		}

		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")
	m.productList.SetContent(content)

	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render("Products:")

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(lipgloss.Color("241")).
		Width(m.width - 2).
		Render(header + "\n" + m.productList.View())
}

func (m DashboardModel) renderLogs() string {
	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render("Recent Events:")

	if len(m.logs) == 0 {
		emptyMsg := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("  No events yet...")
		return lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(lipgloss.Color("241")).
			Width(m.width - 2).
			Render(header + "\n" + emptyMsg)
	}

	var lines []string
	start := 0
	if len(m.logs) > m.maxLogs {
		start = len(m.logs) - m.maxLogs
	}

	for _, log := range m.logs[start:] {
		timestamp := log.Timestamp.Format("15:04:05")

		var style lipgloss.Style
		switch log.Level {
		case "ERROR":
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		case "WARN":
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
		default:
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		}

		line := style.Render(fmt.Sprintf("  [%s] %s - %s", log.Level, timestamp, log.Message))
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")
	m.logView.SetContent(content)
	m.logView.GotoBottom()

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(lipgloss.Color("241")).
		Width(m.width - 2).
		Render(header + "\n" + m.logView.View())
}

func (m DashboardModel) renderFooter() string {
	// Progress bar
	completed := m.successCount + m.failedCount
	total := len(m.products)
	percent := 0.0
	if total > 0 {
		percent = float64(completed) / float64(total)
	}

	progressBar := m.progress.ViewAs(percent)

	// Dynamic hotkey help based on state
	var keys []string

	if m.scrapingState == ScrapingIdle {
		if m.browserState == BrowserNotStarted && !m.externalBrowser {
			keys = append(keys, "Enter: start browser")
		} else if m.browserState == BrowserReady {
			keys = append(keys, "Enter: start scraping")
		}
	}

	if m.scrapingState == ScrapingActive || m.scrapingState == ScrapingPaused {
		if m.scrapingState == ScrapingActive {
			keys = append(keys, "Space: pause")
		} else {
			keys = append(keys, "Space: resume")
		}
		keys = append(keys, "S: skip")
	}

	if !m.externalBrowser {
		keys = append(keys, "K: kill browser", "B: restart")
	}

	keys = append(keys, "↑/↓: scroll", "Q: quit")

	helpText := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render(strings.Join(keys, " | "))

	return lipgloss.NewStyle().
		Width(m.width - 2).
		Render(progressBar + "\n" + helpText)
}

// Helper methods
func (m *DashboardModel) addLog(level, message string) {
	m.logs = append(m.logs, LogEntry{
		Level:     level,
		Timestamp: time.Now(),
		Message:   message,
	})
}

func (m *DashboardModel) updateStats() {
	completed := m.successCount + m.failedCount
	if completed > 0 {
		m.avgTime = m.totalElapsed / time.Duration(completed)

		remaining := len(m.products) - completed
		m.eta = time.Duration(remaining) * m.avgTime
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}

// NewProgressModel is deprecated - use NewDashboardModel
func NewProgressModel(productIDs []string) DashboardModel {
	return NewDashboardModel(productIDs, nil)
}
