package agentsession

import (
	"fmt"
	"io"
	"os"
	"strings"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
)

// tuiItem extends pickerItem with multi-select support.
type tuiItem struct {
	entry    Entry
	selected bool
	active   bool // true if session is currently open in pi
}

func (i tuiItem) FilterValue() string { return i.entry.Name }

func (i tuiItem) headline() string {
	return i.entry.Name
}

var (
	tuiSelectedMark   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	tuiUnselectedMark = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	tuiConfirmStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	tuiHelpBarStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).PaddingLeft(4).PaddingBottom(1)
)

type tuiDelegate struct{}

func (d tuiDelegate) Height() int                             { return 1 }
func (d tuiDelegate) Spacing() int                            { return 0 }
func (d tuiDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d tuiDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(tuiItem)
	if !ok {
		return
	}

	checkbox := tuiUnselectedMark.Render("○")
	if i.selected {
		checkbox = tuiSelectedMark.Render("●")
	}

	name := i.headline()
	if index == m.Index() {
		name = selectedItemStyle.Render(name)
		fmt.Fprint(w, fmt.Sprintf("> %s %s", checkbox, name))
	} else {
		name = itemStyle.Render(name)
		fmt.Fprint(w, fmt.Sprintf("  %s %s", checkbox, name))
	}
}

type tuiMode int

const (
	tuiModeNormal tuiMode = iota
	tuiModeConfirmDelete
	tuiModeRename
)

type tuiModel struct {
	list      list.Model
	sessions  Sessions
	order     SortOrder
	yolo      bool
	allowYolo bool
	mode      tuiMode
	chosen    *Entry // set when user picks one to resume
	quit      bool
	filePath  string
	provider  Provider
	message   string // status message shown briefly
	activeIDs map[string]bool // active pi session IDs
	pathScope string // directory scope (empty = all)
	renameInput textinput.Model
	renameEntry *Entry // entry being renamed
}

func (m tuiModel) Init() tea.Cmd { return nil }

func (m tuiModel) selectedCount() int {
	count := 0
	for _, item := range m.list.Items() {
		if ti, ok := item.(tuiItem); ok && ti.selected {
			count++
		}
	}
	return count
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil
	case tea.KeyPressMsg:
		// In confirm-delete mode, only y/n/esc
		if m.mode == tuiModeConfirmDelete {
			switch msg.String() {
			case "y", "Y":
				m.performDelete()
				m.mode = tuiModeNormal
				return m, nil
			case "n", "N", "esc":
				m.mode = tuiModeNormal
				m.message = ""
				return m, nil
			}
			return m, nil
		}

		// In rename mode, handle text input
		if m.mode == tuiModeRename {
			switch msg.String() {
			case "enter":
				newName := strings.TrimSpace(m.renameInput.Value())
				if newName != "" && m.renameEntry != nil {
					m.performRename(*m.renameEntry, newName)
				}
				m.mode = tuiModeNormal
				m.renameEntry = nil
				return m, nil
			case "esc":
				m.mode = tuiModeNormal
				m.message = ""
				m.renameEntry = nil
				return m, nil
			default:
				var cmd tea.Cmd
				m.renameInput, cmd = m.renameInput.Update(msg)
				return m, cmd
			}
		}

		// In filtering mode, let the list handle it
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch msg.String() {
		case "q", "ctrl+c":
			m.quit = true
			return m, tea.Quit

		case " ", "space": // toggle select
			if i, ok := m.list.SelectedItem().(tuiItem); ok {
				i.selected = !i.selected
				m.list.SetItem(m.list.Index(), i)
			}
			return m, nil

		case "a": // select all / deselect all
			allSelected := true
			for _, item := range m.list.Items() {
				if ti, ok := item.(tuiItem); ok && !ti.selected {
					allSelected = false
					break
				}
			}
			items := m.list.Items()
			for idx, item := range items {
				if ti, ok := item.(tuiItem); ok {
					ti.selected = !allSelected
					items[idx] = ti
				}
			}
			m.list.SetItems(items)
			return m, nil

		case "d", "D": // delete selected or current
			if m.selectedCount() == 0 {
				// No selection: soft-delete just the highlighted item
				if i, ok := m.list.SelectedItem().(tuiItem); ok {
					m.performDeleteEntries([]Entry{i.entry})
				}
			} else {
				m.performDelete()
			}
			return m, nil

		case "e": // edit selected item
			if i, ok := m.list.SelectedItem().(tuiItem); ok {
				entry := i.entry
				updated, err := runEditForm(&entry)
				if err != nil || updated == nil {
					return m, nil
				}

				allSessions, err := Load(m.filePath)
				if err != nil {
					m.message = "Error: " + err.Error()
					return m, nil
				}

				oldName := entry.Name
				entry.Name = updated.Name
				entry.Description = updated.Description

				if oldName != entry.Name {
					allSessions = RemoveForProvider(allSessions, m.provider.Name, oldName)
				}
				upsertEntry(allSessions, entry)
				if err := Save(m.filePath, allSessions); err != nil {
					m.message = "Error saving: " + err.Error()
					return m, nil
				}

				m.sessions = allSessions
				m.refreshList()
				m.message = fmt.Sprintf("Updated %q", entry.Name)
			}
			return m, nil

		case "r": // rename current item inline
			if i, ok := m.list.SelectedItem().(tuiItem); ok {
				entry := i.entry
				m.renameInput = textinput.New()
				m.renameInput.SetValue(entry.Name)
				m.renameInput.Focus()
				m.renameEntry = &entry
				m.mode = tuiModeRename
				m.message = "Rename (enter to confirm, esc to cancel)"
			}
			return m, nil

		case "s": // sort
			m.order = m.order.Next()
			m.refreshList()
			return m, nil

		case "y": // copy ID to clipboard
			if i, ok := m.list.SelectedItem().(tuiItem); ok && i.entry.ID != "" {
				if err := copyToClipboard(i.entry.ID); err != nil {
					m.message = "Error copying: " + err.Error()
				} else {
					m.message = "Copied ID: " + i.entry.ID
				}
			}
			return m, nil

		case "Y": // yolo toggle
			if m.allowYolo {
				m.yolo = !m.yolo
			}
			return m, nil

		case "enter": // resume
			if i, ok := m.list.SelectedItem().(tuiItem); ok {
				e := i.entry
				m.chosen = &e
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *tuiModel) performDelete() {
	var entries []Entry
	for _, item := range m.list.Items() {
		if ti, ok := item.(tuiItem); ok && ti.selected {
			entries = append(entries, ti.entry)
		}
	}
	m.performDeleteEntries(entries)
}

func (m *tuiModel) performDeleteEntries(entries []Entry) {
	allSessions, err := Load(m.filePath)
	if err != nil {
		m.message = "Error: " + err.Error()
		return
	}

	for _, e := range entries {
		allSessions = SoftDeleteForProvider(allSessions, effectiveProvider(e), e.Name)
	}

	if err := Save(m.filePath, allSessions); err != nil {
		m.message = "Error saving: " + err.Error()
		return
	}

	m.sessions = allSessions
	m.refreshList()
	m.message = fmt.Sprintf("Deleted %d session(s)", len(entries))
}

func (m *tuiModel) performRename(entry Entry, newName string) {
	allSessions, err := Load(m.filePath)
	if err != nil {
		m.message = "Error: " + err.Error()
		return
	}

	oldName := entry.Name
	if oldName != newName {
		allSessions = RemoveForProvider(allSessions, effectiveProvider(entry), oldName)
	}
	entry.Name = newName
	upsertEntry(allSessions, entry)

	if err := Save(m.filePath, allSessions); err != nil {
		m.message = "Error saving: " + err.Error()
		return
	}

	m.sessions = allSessions
	m.refreshList()
	m.message = fmt.Sprintf("Renamed %q → %q", oldName, newName)
}

func (m *tuiModel) refreshList() {
	sessions := m.sessions
	if m.pathScope != "" {
		sessions = filterByPath(sessions, m.pathScope)
	}
	providerFilter := m.provider.Name
	if providerFilter != "all" {
		sessions = filterByProvider(sessions, providerFilter)
	}

	entries := sortedEntries(filterDeleted(sessions, false), m.order)
	items := make([]list.Item, 0, len(entries))
	for _, e := range entries {
		if e.Name == "" {
			continue
		}
		items = append(items, tuiItem{entry: e, active: m.activeIDs[e.ID]})
	}
	m.list.SetItems(items)
}

func (m tuiModel) View() tea.View {
	if m.chosen != nil || m.quit {
		return tea.View{}
	}

	m.list.SetShowHelp(false)
	m.list.SetShowPagination(false)

	var b strings.Builder

	b.WriteString(strings.TrimLeft(m.list.View(), "\n"))
	b.WriteString("\n")

	// Detail box for selected item
	if i, ok := m.list.SelectedItem().(tuiItem); ok {
		e := i.entry
		desc := e.Description
		if desc == "" {
			desc = "-"
		}
		lines := []string{
			dimStyle.Render("🤖 " + effectiveProvider(e)),
			dimStyle.Render("🆔 " + e.ID),
			dimStyle.Render("📁 " + shortenPath(e.Path)),
			dimStyle.Render("📅 " + formatSavedAt(e.SavedAt)),
			dimStyle.Render("📝 " + desc),
		}
		if len(e.Meta) == 0 {
			lines = append(lines, dimStyle.Render("🏷️  -"))
		} else {
			first := true
			for k, v := range e.Meta {
				if first {
					lines = append(lines, dimStyle.Render(fmt.Sprintf("🏷️  %s=%s", k, v)))
					first = false
				} else {
					lines = append(lines, dimStyle.Render(fmt.Sprintf("   %s=%s", k, v)))
				}
			}
		}
		b.WriteString(footerBoxStyle.Render(strings.Join(lines, "\n")))
		b.WriteString("\n")
	}

	// Status message or confirm prompt
	if m.mode == tuiModeRename {
		b.WriteString("  " + dimStyle.Render(m.message) + "\n")
		b.WriteString("  " + m.renameInput.View() + "\n")
	} else if m.message != "" {
		if m.mode == tuiModeConfirmDelete {
			b.WriteString("  " + tuiConfirmStyle.Render(m.message))
		} else {
			b.WriteString("  " + dimStyle.Render(m.message))
		}
		b.WriteString("\n")
	}

	// Pagination
	b.WriteString(m.list.Styles.PaginationStyle.Render(m.list.Paginator.View()))
	b.WriteString("\n")

	// Help bar
	count := m.selectedCount()
	helpParts := []string{
		"enter: resume",
		"space: select",
		"a: all",
		fmt.Sprintf("d: delete (%d)", count),
		"e: edit",
		"r: rename",
		"s: " + m.order.Label(),
		"/: filter",
	}
	helpParts = append(helpParts, "y: copy id")
	if m.allowYolo {
		yoloStatus := "off"
		if m.yolo {
			yoloStatus = yoloOnStyle.Render("ON")
		}
		helpParts = append(helpParts, "Y: yolo "+yoloStatus)
	}
	helpParts = append(helpParts, "q: quit")
	b.WriteString(tuiHelpBarStyle.Render(strings.Join(helpParts, "  •  ")))

	return tea.NewView(b.String())
}

func tuiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Interactive TUI for managing sessions (list, resume, delete, edit)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI(cmd)
		},
	}
}

func runTUI(cmd *cobra.Command) error {
	sessions, err := loadScopedSessions(cmd)
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		cmd.PrintErrln("No sessions found.")
		return nil
	}

	// Determine path scope for refreshList
	var pathScope string
	all, _ := cmd.Flags().GetBool("all")
	directory, _ := cmd.Flags().GetString("directory")
	if directory != "" {
		pathScope = directory
	} else if !all {
		pathScope, _ = os.Getwd()
	}

	filePath := filePathFn()
	providerFilter := providerFilterFromCmd(cmd)
	providerName := providerFilter
	if providerName == "" || providerName == "all" {
		providerName = defaultProvider()
	}
	provider, err := providerByName(providerName)
	if err != nil {
		return err
	}

	allowYolo := providerSupportsYolo(provider)
	activeIDs := activePiSessionIDs()

	entries := sortedEntries(sessions, SortDateDesc)
	items := make([]list.Item, 0, len(entries))
	for _, e := range entries {
		if e.Name == "" {
			continue
		}
		items = append(items, tuiItem{entry: e, active: activeIDs[e.ID]})
	}

	const listHeight = 14
	l := list.New(items, tuiDelegate{}, 0, listHeight)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().MarginLeft(2)
	l.Styles.PaginationStyle = list.DefaultStyles(false).PaginationStyle.PaddingLeft(4)
	l.Styles.HelpStyle = list.DefaultStyles(false).HelpStyle.PaddingLeft(4).PaddingBottom(1)

	model := tuiModel{
		list:      l,
		sessions:  sessions,
		order:     SortDateDesc,
		allowYolo: allowYolo,
		filePath:  filePath,
		provider:  provider,
		activeIDs: activeIDs,
		pathScope: pathScope,
	}

	p := tea.NewProgram(model)
	final, err := p.Run()
	if err != nil {
		return err
	}

	fm := final.(tuiModel)
	if fm.quit || fm.chosen == nil {
		return nil
	}
	return resumeEntry(fm.chosen, provider, fm.yolo)
}
