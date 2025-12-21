package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

// DashboardModel is the Bubbletea model for scraping dashboard
type DashboardModel struct {
	// Product tracking
	products []ProductItem
	current  int
	paused   bool

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

	// Browser status
	browserStatus BrowserStatus

	// Logs
	logs    []LogEntry
	maxLogs int

	// Dimensions
	width  int
	height int

	// Control
	done        bool
	controlChan chan<- ControlMsg
}

type keymap struct {
	pause      key.Binding
	skip       key.Binding
	retry      key.Binding
	scrollUp   key.Binding
	scrollDown key.Binding
	quit       key.Binding
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
		scrollUp: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "scroll up"),
		),
		scrollDown: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "scroll down"),
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
		browserStatus: BrowserStatus{Connected: true, Message: "Connected"},
		controlChan:   controlChan,
	}
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

func (m DashboardModel) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
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
			if m.controlChan != nil {
				m.controlChan <- ControlMsg{Type: "stop"}
			}
			return m, tea.Quit

		case key.Matches(msg, m.keymap.pause):
			m.paused = !m.paused
			if m.controlChan != nil {
				if m.paused {
					m.controlChan <- ControlMsg{Type: "pause"}
					m.addLog("INFO", "Scraping paused")
				} else {
					m.controlChan <- ControlMsg{Type: "resume"}
					m.addLog("INFO", "Scraping resumed")
				}
			}

		case key.Matches(msg, m.keymap.skip):
			if m.current >= 0 && m.current < len(m.products) && !m.paused {
				if m.controlChan != nil {
					m.controlChan <- ControlMsg{Type: "skip"}
					m.addLog("INFO", "Skipped product: "+m.products[m.current].ID)
				}
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
		m.browserStatus = BrowserStatus{
			Connected: msg.Connected,
			Message:   msg.Message,
		}
		if !msg.Connected {
			m.addLog("WARN", "Browser: "+msg.Message)
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
		Render("🌐 Scraping AliExpress Products")

	statusIcon := "✓"
	statusColor := lipgloss.Color("34")
	if !m.browserStatus.Connected {
		statusIcon = "✗"
		statusColor = lipgloss.Color("196")
	}

	status := lipgloss.NewStyle().
		Foreground(statusColor).
		Render(fmt.Sprintf("[Browser: %s %s]", statusIcon, m.browserStatus.Message))

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

	pauseIndicator := ""
	if m.paused {
		pauseIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render(" [PAUSED]")
	}

	content := line1 + "\n" + line2 + pauseIndicator

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

	// Hotkey help
	keys := []string{
		m.keymap.pause.Help().Key + ": pause",
		m.keymap.skip.Help().Key + ": skip",
		m.keymap.retry.Help().Key + ": retry failed",
		"j/k: scroll",
		m.keymap.quit.Help().Key + ": quit",
	}

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
