package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"

	"github.com/NSXBet/nsx-cli/shared/interact"
	"github.com/NSXBet/nsx-cli/shared/style"
	"github.com/NSXBet/nsx-cli/shared/ui"
)

// Column keys for the database process list table
const (
	columnKeyID           = "id"
	columnKeyUser         = "user"
	columnKeyDatabase     = "database"
	columnKeyDuration     = "duration"
	columnKeyCommand      = "command"
	columnKeyState        = "state"
	columnKeyQueryPreview = "query_preview"
)

// List of users to ignore for long-running queries (system users)
var ignoredUsers = []string{
	// MySQL system users
	"event_scheduler",
	"rdsadmin",
	"mysql.session",
	"mysql.sys",
	"root",
	"system user",
	// PostgreSQL system users
	"postgres", // Default superuser
	"rds_superuser",
	"rds_replication",
	"rdsdb",
	"aurora_superuser", // Aurora PostgreSQL
}

// QueryResult represents a database process list query result
type QueryResult struct {
	ID           int
	User         string
	Host         string
	Database     string
	Command      string
	Duration     int
	DurationStr  string
	State        string
	QueryPreview string
}

// DBProcessListView provides functionality to display database process lists
type DBProcessListView struct {
	title       string
	results     []QueryResult
	refreshFunc func() ([]QueryResult, error)
	killFunc    func(int) error
}

// NewDBProcessListView creates a new database process list view
func NewDBProcessListView(
	title string,
	results []QueryResult,
	refreshFunc func() ([]QueryResult, error),
	killFunc func(int) error,
) *DBProcessListView {
	return &DBProcessListView{
		title:       title,
		results:     results,
		refreshFunc: refreshFunc,
		killFunc:    killFunc,
	}
}

// Display shows the database process list in an interactive table
func (v *DBProcessListView) Display() error {
	// Try to create interactive table first
	if err := v.displayInteractiveTable(); err != nil {
		// Fall back to simple text table if TTY is not available
		interact.Info("Interactive table not available, showing simple table:")
		v.displaySimpleTable()
	}

	return nil
}

// displayInteractiveTable creates and runs an interactive table with refresh capability
func (v *DBProcessListView) displayInteractiveTable() error {
	// Define columns with flexible sizing
	columns := []table.Column{
		table.NewColumn(columnKeyID, "ID", 10),
		table.NewFlexColumn(columnKeyUser, "User", 15),
		table.NewColumn(columnKeyDatabase, "Database", 16),
		table.NewFlexColumn(columnKeyDuration, "Duration", 12),
		table.NewColumn(columnKeyCommand, "Command", 10),
		table.NewFlexColumn(columnKeyState, "State", 20),
		table.NewFlexColumn(columnKeyQueryPreview, "Query Preview", 64),
	}

	// Create a refreshable table model
	refreshableTable := &RefreshableTable{
		title:       v.title,
		columns:     columns,
		data:        v.results,
		rowFunc:     v.createQueryRowFunc,
		refreshFunc: v.refreshFunc,
		killFunc:    v.killFunc,
	}

	// Initialize the table
	refreshableTable.initTable()

	// Run the table
	program := tea.NewProgram(refreshableTable, tea.WithAltScreen())
	_, err := program.Run()
	return err
}

// createQueryRowFunc creates a table row from a QueryResult
func (v *DBProcessListView) createQueryRowFunc(result QueryResult) table.Row {
	return table.NewRow(table.RowData{
		columnKeyID:           fmt.Sprintf("%d", result.ID),
		columnKeyUser:         result.User,
		columnKeyDatabase:     result.Database,
		columnKeyDuration:     result.DurationStr,
		columnKeyCommand:      result.Command,
		columnKeyState:        result.State,
		columnKeyQueryPreview: result.QueryPreview,
	})
}

// displaySimpleTable displays a simple ASCII table as fallback
func (v *DBProcessListView) displaySimpleTable() {
	// Print title
	if v.title != "" {
		fmt.Println(style.Title.Render(v.title))
		fmt.Println()
	}

	// Print header with styling
	fmt.Println(
		style.TableBorder.Render(
			"┌──────┬───────────────┬────────────┬────────────┬──────────┬────────────────────┬────────────────────────────────────────┐",
		),
	)
	fmt.Println(
		style.TableBorder.Render(
			"│ ",
		) + style.FormatTableHeader(
			fmt.Sprintf("%-4s", "ID"),
		) + style.TableBorder.Render(
			" │ ",
		) + style.FormatTableHeader(
			fmt.Sprintf("%-13s", "User"),
		) + style.TableBorder.Render(
			" │ ",
		) + style.FormatTableHeader(
			fmt.Sprintf("%-10s", "Database"),
		) + style.TableBorder.Render(
			" │ ",
		) + style.FormatTableHeader(
			fmt.Sprintf("%-10s", "Duration"),
		) + style.TableBorder.Render(
			" │ ",
		) + style.FormatTableHeader(
			fmt.Sprintf("%-8s", "Command"),
		) + style.TableBorder.Render(
			" │ ",
		) + style.FormatTableHeader(
			fmt.Sprintf("%-18s", "State"),
		) + style.TableBorder.Render(
			" │ ",
		) + style.FormatTableHeader(
			fmt.Sprintf("%-38s", "Query Preview"),
		) + style.TableBorder.Render(
			" │",
		),
	)
	fmt.Println(
		style.TableBorder.Render(
			"├──────┼───────────────┼────────────┼────────────┼──────────┼────────────────────┼────────────────────────────────────────┤",
		),
	)

	// Print rows with alternating styling
	for i, result := range v.results {
		idStr := fmt.Sprintf("%-4d", result.ID)
		userStr := fmt.Sprintf("%-13s", truncateString(result.User, 13))
		dbStr := fmt.Sprintf("%-10s", truncateString(result.Database, 10))
		durationStr := fmt.Sprintf("%-10s", result.DurationStr)
		commandStr := fmt.Sprintf("%-8s", truncateString(result.Command, 8))
		stateStr := fmt.Sprintf("%-18s", truncateString(result.State, 18))
		queryStr := fmt.Sprintf("%-38s", truncateString(result.QueryPreview, 38))

		fmt.Println(
			style.TableBorder.Render(
				"│ ",
			) + style.FormatTableRow(
				idStr,
				i%2 == 0,
			) + style.TableBorder.Render(
				" │ ",
			) + style.FormatTableRow(
				userStr,
				i%2 == 0,
			) + style.TableBorder.Render(
				" │ ",
			) + style.FormatTableRow(
				dbStr,
				i%2 == 0,
			) + style.TableBorder.Render(
				" │ ",
			) + style.FormatDuration(
				durationStr,
			) + style.TableBorder.Render(
				" │ ",
			) + style.FormatTableRow(
				commandStr,
				i%2 == 0,
			) + style.TableBorder.Render(
				" │ ",
			) + style.FormatTableRow(
				stateStr,
				i%2 == 0,
			) + style.TableBorder.Render(
				" │ ",
			) + style.FormatTableRow(
				queryStr,
				i%2 == 0,
			) + style.TableBorder.Render(
				" │",
			),
		)
	}

	fmt.Println(
		style.TableBorder.Render(
			"└──────┴───────────────┴────────────┴────────────┴──────────┴────────────────────┴────────────────────────────────────────┘",
		),
	)
}

// truncateString truncates a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// FormatDuration formats seconds into a human-readable duration string
func FormatDuration(seconds int) string {
	duration := time.Duration(seconds) * time.Second

	var durationStr string
	if duration < time.Minute {
		durationStr = fmt.Sprintf("%ds", seconds)
	} else if duration < time.Hour {
		durationStr = fmt.Sprintf("%dm %ds", int(duration.Minutes()), seconds%60)
	} else {
		hours := int(duration.Hours())
		minutes := int(duration.Minutes()) % 60
		durationStr = fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds%60)
	}

	return style.FormatDuration(durationStr)
}

// ShouldIgnoreUser checks if a user should be ignored for long-running queries
func ShouldIgnoreUser(user string) bool {
	userLower := strings.ToLower(user)
	for _, ignoredUser := range ignoredUsers {
		if strings.ToLower(ignoredUser) == userLower {
			return true
		}
	}
	return false
}

// UpdateResults updates the query results and refreshes the display
func (v *DBProcessListView) UpdateResults(newResults []QueryResult) {
	v.results = newResults
}

// GetResults returns the current query results
func (v *DBProcessListView) GetResults() []QueryResult {
	return v.results
}

// SetTitle updates the view title
func (v *DBProcessListView) SetTitle(title string) {
	v.title = title
}

// RefreshableTable is a custom table that can refresh its data
type RefreshableTable struct {
	title       string
	columns     []table.Column
	data        []QueryResult
	rowFunc     func(QueryResult) table.Row
	refreshFunc func() ([]QueryResult, error)
	killFunc    func(int) error // Function to kill a process by ID
	tableModel  *ui.Table[QueryResult]
	refreshing  bool
	pendingKill *QueryResult // Process pending kill confirmation
	statusMsg   string       // Current status message for user feedback
	showStatus  bool         // Whether to show the status message
}

// initTable initializes the refreshable table
func (rt *RefreshableTable) initTable() {
	customOptions := []ui.CustomOption{
		{
			Key:   "k",
			Title: "kill",
			Function: func(selectedRow interface{}) tea.Cmd {
				if result, ok := selectedRow.(QueryResult); ok {
					return rt.handleKillCommand(result)
				}
				return nil
			},
		},
		{
			Key:   "r",
			Title: "refresh",
			Function: func(selectedRow interface{}) tea.Cmd {
				// Don't allow refresh if already refreshing
				if rt.refreshing {
					return nil
				}
				return rt.refreshData()
			},
		},
		{
			Key:   "c",
			Title: "copy",
			Function: func(selectedRow interface{}) tea.Cmd {
				if result, ok := selectedRow.(QueryResult); ok {
					return rt.copyQueryPreview(result)
				}
				return nil
			},
		},
	}

	options := ui.TableOption[QueryResult]{
		Title:         rt.title,
		Columns:       rt.columns,
		Data:          rt.data,
		RowFunc:       rt.rowFunc,
		CustomOptions: customOptions,
		Width:         0,
	}

	rt.tableModel = ui.NewTable(options)

	// Sort by duration in descending order (longest queries first)
	rt.tableModel.SortByDesc(columnKeyDuration)
}

// handleKillCommand handles the kill command with confirmation
func (rt *RefreshableTable) handleKillCommand(result QueryResult) tea.Cmd {
	return func() tea.Msg {
		if rt.pendingKill != nil && rt.pendingKill.ID == result.ID {
			// Second press - confirm kill
			return KillConfirmedMsg{process: result}
		} else {
			// First press - request confirmation
			return KillRequestMsg{process: result}
		}
	}
}

// refreshData handles the refresh command
func (rt *RefreshableTable) refreshData() tea.Cmd {
	return tea.Batch(
		// First command: Update UI to show refreshing state
		func() tea.Msg {
			return RefreshStartMsg{}
		},
		// Second command: Perform the actual refresh
		func() tea.Msg {
			if rt.refreshFunc == nil {
				return RefreshErrorMsg{error: fmt.Errorf("refresh function not available")}
			}

			newResults, err := rt.refreshFunc()
			if err != nil {
				return RefreshErrorMsg{error: err}
			}

			return RefreshSuccessMsg{results: newResults}
		},
	)
}

// Init implements tea.Model
func (rt *RefreshableTable) Init() tea.Cmd {
	return rt.tableModel.Init()
}

// Update implements tea.Model
func (rt *RefreshableTable) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case RefreshStartMsg:
		// Set refreshing state
		rt.refreshing = true
		return rt, rt.setStatusMessageWithTimer("Refreshing data...")

	case RefreshSuccessMsg:
		// Update the data and refresh the table
		rt.data = msg.results
		rt.tableModel.UpdateData(msg.results)
		// Re-sort by duration in descending order after refresh
		rt.tableModel.SortByDesc(columnKeyDuration)
		rt.refreshing = false
		return rt, rt.setStatusMessageWithTimer(fmt.Sprintf("Refreshed successfully (%d results)", len(msg.results)))

	case RefreshErrorMsg:
		rt.refreshing = false
		return rt, rt.setStatusMessageWithTimer(fmt.Sprintf("Refresh failed: %v", msg.error))

	case KillRequestMsg:
		// First press - request confirmation
		rt.pendingKill = &msg.process
		rt.updateFooterForKill()
		interact.Warn("Press 'k' again to confirm killing process ID %d (User: %s)", msg.process.ID, msg.process.User)
		return rt, nil

	case KillConfirmedMsg:
		// Second press - perform kill
		rt.pendingKill = nil
		rt.resetFooter()
		return rt, rt.performKill(msg.process)

	case KillSuccessMsg:
		// Refresh the table after successful kill
		return rt, rt.refreshData()

	case KillErrorMsg:
		return rt, rt.setStatusMessageWithTimer(fmt.Sprintf("Failed to kill process ID %d: %v", msg.processID, msg.error))

	case CopySuccessMsg:
		return rt, rt.setStatusMessageWithTimer(fmt.Sprintf("Query copied to clipboard for process ID %d", msg.processID))

	case CopyErrorMsg:
		return rt, rt.setStatusMessageWithTimer(fmt.Sprintf("Failed to copy query for process ID %d: %v", msg.processID, msg.error))

	case ClearStatusMsg:
		rt.clearStatusMessage()
		return rt, nil
	}

	// Handle key presses when there's a pending kill
	if keyMsg, ok := msg.(tea.KeyMsg); ok && rt.pendingKill != nil {
		// Any key other than 'k' cancels the kill confirmation
		if keyMsg.String() != "k" {
			rt.pendingKill = nil
			rt.resetFooter()
			interact.Info("Kill operation cancelled")
		}
	}

	// Delegate to the table model
	updatedModel, cmd := rt.tableModel.Update(msg)
	if updatedTable, ok := updatedModel.(*ui.Table[QueryResult]); ok {
		rt.tableModel = updatedTable
	}

	// Handle copy messages coming from the table model
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "c" {
		// The copy operation is handled by the custom option function
		// The status message will be set by the CopySuccessMsg or CopyErrorMsg
		return rt, cmd
	}

	return rt, cmd
}

// updateFooterForKill updates the footer to show kill confirmation
func (rt *RefreshableTable) updateFooterForKill() {
	if rt.pendingKill != nil {
		newFooter := fmt.Sprintf(
			"KILL CONFIRMATION: Press 'k' again to kill process ID %d (User: %s) • Any other key to cancel",
			rt.pendingKill.ID,
			rt.pendingKill.User,
		)
		rt.tableModel.SetFooter(newFooter)
	}
}

// resetFooter resets the footer to the default state
func (rt *RefreshableTable) resetFooter() {
	rt.tableModel.View()
}

// setStatusMessageWithTimer sets a status message and returns a timer command to clear it
func (rt *RefreshableTable) setStatusMessageWithTimer(msg string) tea.Cmd {
	rt.statusMsg = msg
	rt.showStatus = true
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return ClearStatusMsg{}
	})
}

// clearStatusMessage clears the current status message
func (rt *RefreshableTable) clearStatusMessage() {
	rt.statusMsg = ""
	rt.showStatus = false
}

// performKill performs the actual kill operation
func (rt *RefreshableTable) performKill(process QueryResult) tea.Cmd {
	return func() tea.Msg {
		interact.Info("Killing process ID %d...", process.ID)

		// Use the actual kill function if available
		if rt.killFunc != nil {
			if err := rt.killFunc(process.ID); err != nil {
				return KillErrorMsg{processID: process.ID, error: err}
			}
			return KillSuccessMsg{processID: process.ID}
		}

		// Fallback if no kill function is provided
		return KillErrorMsg{processID: process.ID, error: fmt.Errorf("kill function not available")}
	}
}

// copyQueryPreview copies the query preview to the clipboard
func (rt *RefreshableTable) copyQueryPreview(process QueryResult) tea.Cmd {
	return func() tea.Msg {
		if err := clipboard.WriteAll(process.QueryPreview); err != nil {
			return CopyErrorMsg{processID: process.ID, error: err}
		}

		return CopySuccessMsg{processID: process.ID}
	}
}

// View implements tea.Model
func (rt *RefreshableTable) View() string {
	// Get the normal table view
	tableView := rt.tableModel.View()

	// Add status message if one is set (after the table footer)
	if rt.showStatus && rt.statusMsg != "" {
		statusStyle := style.Dimmed.
			Align(lipgloss.Center).
			MarginTop(1)
		statusLine := statusStyle.Render(rt.statusMsg)

		return lipgloss.JoinVertical(lipgloss.Left, tableView, statusLine)
	}

	return tableView
}

// RefreshStartMsg represents the start of a refresh operation
type RefreshStartMsg struct{}

// RefreshSuccessMsg represents a successful refresh
type RefreshSuccessMsg struct {
	results []QueryResult
}

// RefreshErrorMsg represents a failed refresh
type RefreshErrorMsg struct {
	error error
}

// KillRequestMsg represents a kill request (first press)
type KillRequestMsg struct {
	process QueryResult
}

// KillConfirmedMsg represents a confirmed kill (second press)
type KillConfirmedMsg struct {
	process QueryResult
}

// KillSuccessMsg represents a successful kill
type KillSuccessMsg struct {
	processID int
}

// KillErrorMsg represents a failed kill
type KillErrorMsg struct {
	processID int
	error     error
}

// CopySuccessMsg represents a successful copy operation
type CopySuccessMsg struct {
	processID int
}

// CopyErrorMsg represents a failed copy operation
type CopyErrorMsg struct {
	processID int
	error     error
}

// ClearStatusMsg represents a request to clear the status message
type ClearStatusMsg struct{}
