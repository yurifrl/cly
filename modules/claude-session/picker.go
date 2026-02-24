package claudesession

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var appStyle = lipgloss.NewStyle().Padding(1, 2)

var titleStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FFFDF5")).
	Background(lipgloss.Color("#25A065")).
	Padding(0, 1)

type pickerItem struct{ entry Entry }

func (i pickerItem) Title() string {
	id := i.entry.ID
	if len(id) > 8 {
		id = id[:8]
	}
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	meta := []string{shortenPath(i.entry.Path)}
	if !i.entry.SavedAt.IsZero() {
		meta = append(meta, i.entry.SavedAt.Format("2006-01-02 15:04"))
	}
	meta = append(meta, id)
	return fmt.Sprintf("%s  %s", i.entry.Name, dim.Render(strings.Join(meta, " · ")))
}

func (i pickerItem) Description() string {
	if i.entry.Description != "" {
		return i.entry.Description
	}
	return ""
}

func (i pickerItem) FilterValue() string { return i.entry.Name }

type pickerModel struct {
	list     list.Model
	sessions Sessions
	order    SortOrder
	chosen   *Entry
	quit     bool
}

func (m pickerModel) Init() tea.Cmd { return nil }

func (m pickerModel) rebuildList() pickerModel {
	entries := sortedEntries(m.sessions, m.order)
	items := make([]list.Item, 0, len(entries))
	for _, e := range entries {
		items = append(items, pickerItem{entry: e})
	}
	m.list.SetItems(items)
	return m
}

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := appStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
		return m, nil
	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch msg.String() {
		case "q", "ctrl+c":
			m.quit = true
			return m, tea.Quit
		case "s":
			if m.order == SortDate {
				m.order = SortName
			} else {
				m.order = SortDate
			}
			return m.rebuildList(), nil
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

func (m pickerModel) View() string {
	if m.chosen != nil || m.quit {
		return ""
	}
	return appStyle.Render(m.list.View())
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

type SortOrder string

const (
	SortDate SortOrder = "date"
	SortName SortOrder = "name"
)

func sortedEntries(sessions Sessions, order SortOrder) []Entry {
	entries := make([]Entry, 0, len(sessions))
	for _, e := range sessions {
		entries = append(entries, e)
	}
	switch order {
	case SortName:
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name < entries[j].Name
		})
	default: // SortDate
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].SavedAt.After(entries[j].SavedAt)
		})
	}
	return entries
}

func runPicker(sessions Sessions, title string) (*Entry, error) {
	order := SortDate
	entries := sortedEntries(sessions, order)
	items := make([]list.Item, 0, len(entries))
	for _, e := range entries {
		if e.Name == "" {
			continue
		}
		items = append(items, pickerItem{entry: e})
	}

	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(2)
	l := list.New(items, delegate, 0, 0)
	l.Title = title
	l.Styles.Title = titleStyle

	p := tea.NewProgram(pickerModel{list: l, sessions: sessions, order: order}, tea.WithAltScreen())
	m, err := p.Run()
	if err != nil {
		return nil, err
	}

	pm := m.(pickerModel)
	if pm.quit || pm.chosen == nil {
		return nil, nil
	}
	return pm.chosen, nil
}
