package agentsession

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type searchItem struct {
	cand candidate
}

func (i searchItem) FilterValue() string {
	return i.cand.Session.Name + " " + i.cand.Session.Description
}

type searchDelegate struct{ query string }

func (d searchDelegate) Height() int                             { return 3 }
func (d searchDelegate) Spacing() int                            { return 1 }
func (d searchDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d searchDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(searchItem)
	if !ok {
		return
	}
	s := it.cand.Session
	date := sessionDate(s).Format("2006-01-02 15:04")
	header := fmt.Sprintf("%s · %s · %s",
		date,
		s.Provider,
		shortPathSearch(s.Path),
	)
	name := s.Name
	if name == "" {
		name = synthName(s)
	}
	desc := s.Description
	if it.cand.Snippet != "" {
		desc = it.cand.Snippet
	} else if desc == "" {
		desc = bestSnippet(s, d.query, 140)
	}
	if it.cand.Why != "" {
		desc = it.cand.Why
	}

	hitLabel := ""
	switch {
	case it.cand.Source == "ai":
		hitLabel = fmt.Sprintf("ai %.2f", it.cand.Score)
	case it.cand.Hits > 0:
		word := "hit"
		if it.cand.Hits != 1 {
			word = "hits"
		}
		hitLabel = fmt.Sprintf("%d %s", it.cand.Hits, word)
	}
	hitStyled := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(hitLabel)

	cursor := "  "
	titleStyle := lipgloss.NewStyle()
	if index == m.Index() {
		cursor = "▶ "
		titleStyle = titleStyle.Bold(true).Foreground(lipgloss.Color("212"))
	}
	name = highlightMatches(name, d.query, titleStyle)
	desc = highlightMatches(truncateText(desc, 140), d.query,
		lipgloss.NewStyle().Foreground(lipgloss.Color("248")))
	header = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(header)

	fmt.Fprintf(w, "%s%s  %s\n", cursor, name, hitStyled)
	fmt.Fprintf(w, "    %s\n", header)
	fmt.Fprintf(w, "    %s", desc)
}

// highlightMatches wraps every case-insensitive occurrence of `query`
// (or any whitespace-separated token of it) in `s` with an emphasis style.
// Non-matching segments use `base`.
func highlightMatches(s, query string, base lipgloss.Style) string {
	query = strings.TrimSpace(query)
	if query == "" || s == "" {
		return base.Render(s)
	}
	tokens := strings.Fields(strings.ToLower(query))
	if len(tokens) == 0 {
		return base.Render(s)
	}
	lower := strings.ToLower(s)
	hit := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220")).Background(lipgloss.Color("237"))

	type span = struct{ start, end int }
	var spans []span
	for _, tok := range tokens {
		if tok == "" {
			continue
		}
		start := 0
		for {
			idx := strings.Index(lower[start:], tok)
			if idx < 0 {
				break
			}
			spans = append(spans, span{start: start + idx, end: start + idx + len(tok)})
			start += idx + len(tok)
		}
	}
	if len(spans) == 0 {
		return base.Render(s)
	}
	// Merge overlapping spans (e.g. tokens that touch).
	sortSpans(spans)
	merged := spans[:0:0]
	merged = append(merged, spans[0])
	for _, sp := range spans[1:] {
		last := &merged[len(merged)-1]
		if sp.start <= last.end {
			if sp.end > last.end {
				last.end = sp.end
			}
			continue
		}
		merged = append(merged, sp)
	}

	var b strings.Builder
	cursor := 0
	for _, sp := range merged {
		if sp.start > cursor {
			b.WriteString(base.Render(s[cursor:sp.start]))
		}
		b.WriteString(hit.Render(s[sp.start:sp.end]))
		cursor = sp.end
	}
	if cursor < len(s) {
		b.WriteString(base.Render(s[cursor:]))
	}
	return b.String()
}

func sortSpans(spans []struct{ start, end int }) {
	for i := 1; i < len(spans); i++ {
		j := i
		for j > 0 && spans[j-1].start > spans[j].start {
			spans[j-1], spans[j] = spans[j], spans[j-1]
			j--
		}
	}
}

type hlSpan = struct{ start, end int }

// bestSnippet returns a window of `s.SearchableText` (or first user msg)
// centered on the first occurrence of any query token, so the user sees
// CONTEXT around the match instead of always the start of the conversation.
func bestSnippet(s *indexedSession, query string, width int) string {
	source := s.SearchableText
	if source == "" {
		source = s.FirstUserMsg
	}
	if source == "" {
		return ""
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return truncateText(s.FirstUserMsg, width)
	}
	tokens := strings.Fields(strings.ToLower(query))
	lower := strings.ToLower(source)
	hit := -1
	for _, t := range tokens {
		if t == "" {
			continue
		}
		if idx := strings.Index(lower, t); idx >= 0 {
			if hit < 0 || idx < hit {
				hit = idx
			}
		}
	}
	if hit < 0 {
		return truncateText(source, width)
	}
	start := hit - width/3
	if start < 0 {
		start = 0
	}
	end := start + width
	if end > len(source) {
		end = len(source)
		start = end - width
		if start < 0 {
			start = 0
		}
	}
	prefix := ""
	if start > 0 {
		prefix = "…"
	}
	suffix := ""
	if end < len(source) {
		suffix = "…"
	}
	return prefix + strings.ReplaceAll(source[start:end], "\n", " ") + suffix
}

// synthName produces a human-meaningful title for sessions that were never
// `/checkpoint`-ed (so they have no catalog entry / no Name). Preference:
// first line of the first user message → short ID prefix.
func synthName(s *indexedSession) string {
	if s.FirstUserMsg != "" {
		line := strings.SplitN(s.FirstUserMsg, "\n", 2)[0]
		line = strings.TrimSpace(line)
		if line != "" {
			return truncateText(line, 80)
		}
	}
	if len(s.ID) >= 8 {
		return s.ID[:8]
	}
	return s.ID
}

func shortPathSearch(p string) string {
	if p == "" {
		return ""
	}
	parts := strings.Split(p, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2] + "/" + parts[len(parts)-1]
	}
	return p
}

type searchModel struct {
	idx       *searchIndex
	provider  string
	query     textinput.Model
	list      list.Model
	results   []candidate
	aiOn      bool
	aiStatus  string
	sort      SortMode
	live      *liveCache
	chosen    *indexedSession
	quit      bool
	width     int
	height    int
}

type rerankDoneMsg struct {
	results []candidate
	status  string
}

func newSearchModel(idx *searchIndex, providerFilter, initial string, aiOn bool) searchModel {
	ti := textinput.New()
	ti.Placeholder = "search sessions (try: backups, jsonc, dotfiles, fish completions)"
	ti.SetValue(initial)
	ti.Focus()

	delegate := searchDelegate{query: initial}
	l := list.New(nil, delegate, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowFilter(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)

	m := searchModel{
		idx:      idx,
		provider: providerFilter,
		query:    ti,
		list:     l,
		aiOn:     aiOn,
		aiStatus: aiStatusInitial(aiOn),
		sort:     SortByDate,
		live:     newLiveCache(),
	}
	m.results = m.runRank(initial)
	m.applyResults(m.results)
	return m
}

// runRank decides which ranker to use based on whether the query is empty.
// Empty query -> cheap metadata-only ranker. Non-empty query -> live grep
// across all .jsonl files (cached per query so retyping is instant).
func (m searchModel) runRank(query string) []candidate {
	if strings.TrimSpace(query) == "" {
		return rankLocal(m.idx, "", m.provider, m.sort)
	}
	return liveRank(m.idx, m.live, query, m.provider, m.sort, 16)
}

func aiStatusInitial(on bool) string {
	if !on {
		return "ai: off"
	}
	return "ai: idle"
}

func (m *searchModel) applyResults(cands []candidate) {
	items := make([]list.Item, 0, len(cands))
	for _, c := range cands {
		items = append(items, searchItem{cand: c})
	}
	m.list.SetItems(items)
}

func (m searchModel) Init() tea.Cmd { return textinput.Blink }

func (m searchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height-6)
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			m.quit = true
			return m, tea.Quit
		case "enter":
			if it, ok := m.list.SelectedItem().(searchItem); ok {
				m.chosen = it.cand.Session
				return m, tea.Quit
			}
		case "ctrl+a":
			m.aiOn = !m.aiOn
			m.aiStatus = aiStatusInitial(m.aiOn)
			return m, m.scheduleRerank()
		case "ctrl+s", "alt+s":
			m.sort = m.sort.Next()
			m.results = m.runRank(m.query.Value())
			m.applyResults(m.results)
			return m, nil
		case "down", "ctrl+n":
			m.list.CursorDown()
			return m, nil
		case "up", "ctrl+p":
			m.list.CursorUp()
			return m, nil
		}
	case rerankDoneMsg:
		if msg.status != "" {
			m.aiStatus = msg.status
		}
		if len(msg.results) > 0 {
			m.results = msg.results
			m.applyResults(m.results)
		}
	}

	prev := m.query.Value()
	var qcmd tea.Cmd
	m.query, qcmd = m.query.Update(msg)
	cmds := []tea.Cmd{qcmd}
	if m.query.Value() != prev {
		m.results = m.runRank(m.query.Value())
		m.list.SetDelegate(searchDelegate{query: m.query.Value()})
		m.applyResults(m.results)
		cmds = append(cmds, m.scheduleRerank())
	}
	return m, tea.Batch(cmds...)
}

func (m searchModel) scheduleRerank() tea.Cmd {
	if !m.aiOn || strings.TrimSpace(m.query.Value()) == "" || len(m.results) == 0 {
		return nil
	}
	query := m.query.Value()
	cands := m.results
	if len(cands) > 30 {
		cands = cands[:30]
	}
	return func() tea.Msg {
		time.Sleep(400 * time.Millisecond)
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		out, err := rerankAI(ctx, query, cands)
		if err != nil {
			return rerankDoneMsg{status: "ai: failed (" + truncateText(err.Error(), 60) + ")"}
		}
		return rerankDoneMsg{results: out, status: "ai: ready"}
	}
}

var (
	searchHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	searchHelpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	searchStatusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

func (m searchModel) View() tea.View {
	if m.quit {
		return tea.View{}
	}
	header := searchHeaderStyle.Render("cly as search") + "  " +
		searchStatusStyle.Render(fmt.Sprintf("%d results · sort: %s · %s",
			len(m.results), m.sort.Label(), m.aiStatus))
	q := m.query.View()
	help := searchHelpStyle.Render("↑↓ nav   enter resume   ctrl+s sort   ctrl+a toggle ai   esc quit")
	v := tea.NewView(strings.Join([]string{header, q, "", m.list.View(), help}, "\n"))
	v.AltScreen = true
	return v
}

func runSearchTUI(idx *searchIndex, providerFilter, query string, aiOn bool) (*indexedSession, error) {
	m := newSearchModel(idx, providerFilter, query, aiOn)
	p := tea.NewProgram(m)
	final, err := p.Run()
	if err != nil {
		return nil, err
	}
	sm, ok := final.(searchModel)
	if !ok {
		return nil, fmt.Errorf("unexpected model type")
	}
	if sm.quit || sm.chosen == nil {
		return nil, nil
	}
	return sm.chosen, nil
}
