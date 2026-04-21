package agentsession

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	itemStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))
	selectedItemStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	dimStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	dateStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("72"))
	idStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	providerTagStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("117")).Bold(true)
	yoloOnStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	mutedNameStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	footerBoxStyle    = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				PaddingLeft(1).PaddingRight(1).
				MarginLeft(2)
)

type pickerItem struct {
	entry  Entry
	active bool // true if session is currently open in pi
}

func (i pickerItem) FilterValue() string { return i.entry.Name }

func providerTag(provider string) string {
	provider = effectiveProvider(Entry{Provider: provider})
	return providerTagStyle.Render(provider)
}

func (i pickerItem) headline() string {
	sep := dimStyle.Render(" · ")
	parts := []string{}
	if effectiveProvider(i.entry) != "" {
		parts = append(parts, providerTag(i.entry.Provider))
	}
	if !i.entry.SavedAt.IsZero() {
		parts = append(parts, dateStyle.Render(i.entry.SavedAt.Format("2006-01-02 15:04")))
	}
	parts = append(parts, idStyle.Render(i.entry.ID))

	name := i.entry.Name
	if effectiveProvider(i.entry) == "pi" && !i.active {
		name = mutedNameStyle.Render(name)
	}
	return fmt.Sprintf("%s  %s", name, strings.Join(parts, sep))
}

type simpleDelegate struct{}

func (d simpleDelegate) Height() int                             { return 1 }
func (d simpleDelegate) Spacing() int                            { return 0 }
func (d simpleDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d simpleDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(pickerItem)
	if !ok {
		return
	}
	if index == m.Index() {
		fmt.Fprint(w, selectedItemStyle.Render("> "+i.headline()))
	} else {
		fmt.Fprint(w, itemStyle.Render("  "+i.headline()))
	}
}

type pickerModel struct {
	list      list.Model
	sessions  Sessions
	order     SortOrder
	chosen    *Entry
	yolo      bool
	allowYolo bool
	quit      bool
}

func (m pickerModel) Init() tea.Cmd { return nil }

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil
	case tea.KeyPressMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch msg.String() {
		case "q", "ctrl+c":
			m.quit = true
			return m, tea.Quit
		case "s":
			m.order = m.order.Next()
			entries := sortedEntries(m.sessions, m.order)
			items := make([]list.Item, 0, len(entries))
			for _, e := range entries {
				items = append(items, pickerItem{entry: e})
			}
			m.list.SetItems(items)
			return m, nil
		case "y":
			if m.allowYolo {
				m.yolo = !m.yolo
			}
			return m, nil
		case "enter":
			i, ok := m.list.SelectedItem().(pickerItem)
			if ok {
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

func (m pickerModel) View() tea.View {
	if m.chosen != nil || m.quit {
		return tea.View{}
	}

	m.list.SetShowHelp(false)
	m.list.SetShowPagination(false)

	footer := ""
	if i, ok := m.list.SelectedItem().(pickerItem); ok {
		e := i.entry
		content := dimStyle.Render("🤖 "+effectiveProvider(e)) + "\n" +
			dimStyle.Render("📁 "+shortenPath(e.Path)) + "\n" +
			dimStyle.Render("📝 "+e.Description)
		footer = footerBoxStyle.Render(content) + "\n"
	}

	helpText := m.list.Help.View(m.list) + dimStyle.Render("  • s: "+m.order.Label())
	if m.allowYolo {
		yoloStatus := dimStyle.Render("off")
		if m.yolo {
			yoloStatus = yoloOnStyle.Render("ON")
		}
		helpText += dimStyle.Render("  • y: yolo ") + yoloStatus
	}

	pagination := m.list.Styles.PaginationStyle.Render(m.list.Paginator.View()) + "\n"
	help := m.list.Styles.HelpStyle.Render(helpText)

	return tea.NewView(strings.TrimLeft(m.list.View(), "\n") + "\n" + footer + pagination + help)
}

func shortenPath(p string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	if strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}

type SortOrder int

const (
	SortDateDesc SortOrder = iota // default
	SortDateAsc
	SortNameAsc
	SortNameDesc
)

func (o SortOrder) Next() SortOrder { return (o + 1) % 4 }

func (o SortOrder) Label() string {
	switch o {
	case SortDateAsc:
		return "date ↑"
	case SortNameAsc:
		return "name ↑"
	case SortNameDesc:
		return "name ↓"
	default:
		return "date ↓"
	}
}

func sortedEntries(sessions Sessions, order SortOrder) []Entry {
	entries := make([]Entry, 0, len(sessions))
	for _, e := range sessions {
		entries = append(entries, e)
	}
	switch order {
	case SortDateAsc:
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].SavedAt.Before(entries[j].SavedAt)
		})
	case SortNameAsc:
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name < entries[j].Name
		})
	case SortNameDesc:
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name > entries[j].Name
		})
	default: // SortDateDesc
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].SavedAt.After(entries[j].SavedAt)
		})
	}
	return entries
}

func runPicker(sessions Sessions, allowYolo bool) (*Entry, bool, error) {
	activeIDs := activePiSessionIDs()
	entries := sortedEntries(sessions, SortDateDesc)
	items := make([]list.Item, 0, len(entries))
	for _, e := range entries {
		if e.Name == "" {
			continue
		}
		active := activeIDs[e.ID]
		items = append(items, pickerItem{entry: e, active: active})
	}

	const listHeight = 14
	l := list.New(items, simpleDelegate{}, 0, listHeight)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().MarginLeft(2)
	l.Styles.PaginationStyle = list.DefaultStyles(false).PaginationStyle.PaddingLeft(4)
	l.Styles.HelpStyle = list.DefaultStyles(false).HelpStyle.PaddingLeft(4).PaddingBottom(1)

	p := tea.NewProgram(pickerModel{list: l, sessions: sessions, order: SortDateDesc, allowYolo: allowYolo})
	m, err := p.Run()
	if err != nil {
		return nil, false, err
	}

	pm := m.(pickerModel)
	if pm.quit || pm.chosen == nil {
		return nil, false, nil
	}
	return pm.chosen, pm.yolo, nil
}

// activePiSessionIDs scans ~/.pi/agent/sessions/ for .jsonl files modified
// within the last 10 minutes and extracts their session UUIDs.
// Filename pattern: <timestamp>_<uuid>.jsonl
func activePiSessionIDs() map[string]bool {
	ids := make(map[string]bool)
	home, err := os.UserHomeDir()
	if err != nil {
		return ids
	}

	sessionsDir := filepath.Join(home, ".pi", "agent", "sessions")
	cutoff := time.Now().Add(-10 * time.Minute)

	// Walk all subdirectories
	_ = filepath.Walk(sessionsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		if info.ModTime().Before(cutoff) {
			return nil
		}
		// Extract UUID from filename: <timestamp>_<uuid>.jsonl
		base := filepath.Base(path)
		base = strings.TrimSuffix(base, ".jsonl")
		if idx := strings.LastIndex(base, "_"); idx >= 0 {
			ids[base[idx+1:]] = true
		}
		return nil
	})

	return ids
}
