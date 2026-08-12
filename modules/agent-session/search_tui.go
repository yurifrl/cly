package agentsession

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

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
	source := s.roleBody(roleAll)
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
	idx      *searchIndex
	provider string
	query    textinput.Model
	results  []candidate
	cursor   int // selected result index
	offset   int // first visible result index (scroll)
	aiOn     bool
	aiStatus string
	sort     SortMode
	role     string // "all" | "user" | "assistant"
	live     *liveCache
	chosen   *indexedSession
	quit     bool
	width    int
	height   int

	folder   string // active folder filter ("" = global)
	cwd      string // resolved cwd, target of the folder toggle
	gen      int    // query generation; stale async scans are discarded
	scanning bool
	detail   []string // expanded match snippets for the current item
	detailID string   // session ID whose occurrences are expanded
	showHelp bool     // syntax help overlay toggled with F1

	allResults []candidate // full ranked set before the no-path filter
	hidden     int         // count filtered out for having no path
	showHidden bool        // include no-path sessions (ctrl+h)
	scanStart  time.Time   // when the current scan began (for elapsed)
	spinner    int         // spinner frame while scanning
}

type rerankDoneMsg struct {
	results []candidate
	status  string
}

// searchDoneMsg carries the result of a debounced async scan. Applied only
// if gen still matches the model's current generation.
type searchDoneMsg struct {
	gen     int
	results []candidate
}

// debounceMsg fires after the typing pause; triggers the async scan.
type debounceMsg struct{ gen int }

// spinnerMsg animates the scanning indicator while a scan runs.
type spinnerMsg struct{ gen int }

func newSearchModel(idx *searchIndex, providerFilter, folder, cwd, initial string, aiOn bool, prefs searchPrefs) searchModel {
	ti := textinput.New()
	ti.Placeholder = "search (AND OR NOT \"phrase\" (group) field:val · F1 for help)"
	ti.SetValue(initial)
	ti.Focus()

	m := searchModel{
		idx:      idx,
		provider: providerFilter,
		query:    ti,
		aiOn:     aiOn,
		aiStatus: aiStatusInitial(aiOn),
		sort:     prefs.sort,
		role:     prefs.role,
		live:     newLiveCache(),
		folder:   folder,
		cwd:      cwd,
	}
	m.applyResults(m.runRank(initial))
	return m
}

// runRank decides which ranker to use based on whether the query is empty.
// Empty query -> cheap metadata-only ranker. Non-empty query -> live grep
// across all .jsonl files (cached per query so retyping is instant).
func (m searchModel) runRank(query string) []candidate {
	if strings.TrimSpace(query) == "" {
		return rankLocal(m.idx, "", m.provider, m.folder, m.sort)
	}
	pq := parseQuery(query)
	if pq.empty() {
		return rankLocal(m.idx, "", m.provider, m.folder, m.sort)
	}
	// Full-file scan (complete + consistent across roles), cached per query.
	return liveRank(m.idx, m.live, query, m.provider, m.folder, m.role, m.sort, 16)
}

func aiStatusInitial(on bool) string {
	if !on {
		return "ai: off"
	}
	return "ai: idle"
}

// applyResults swaps in a new result set (applying the no-path filter) and
// resets the cursor/scroll.
func (m *searchModel) applyResults(cands []candidate) {
	m.allResults = cands
	m.rebuildVisible()
	m.cursor = 0
	m.offset = 0
	m.detail = nil
	m.detailID = ""
}

// rebuildVisible derives m.results from m.allResults, hiding sessions with no
// path unless showHidden is set, and recording how many were hidden.
func (m *searchModel) rebuildVisible() {
	if m.showHidden {
		m.results = m.allResults
		m.hidden = 0
		return
	}
	vis := make([]candidate, 0, len(m.allResults))
	hidden := 0
	for _, c := range m.allResults {
		if strings.TrimSpace(c.Session.Path) == "" {
			hidden++
			continue
		}
		vis = append(vis, c)
	}
	m.results = vis
	m.hidden = hidden
}

func (m *searchModel) selected() (candidate, bool) {
	if m.cursor < 0 || m.cursor >= len(m.results) {
		return candidate{}, false
	}
	return m.results[m.cursor], true
}

func (m searchModel) Init() tea.Cmd { return textinput.Blink }

func (m searchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.clampScroll()
	case tea.KeyMsg:
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}
		switch msg.String() {
		case "esc", "ctrl+c":
			m.quit = true
			return m, tea.Quit
		case "f1":
			m.showHelp = true
			return m, nil
		case "enter":
			if c, ok := m.selected(); ok {
				m.chosen = c.Session
				return m, tea.Quit
			}
		case "tab":
			m.toggleExpand()
			return m, nil
		case "ctrl+f":
			if m.folder == "" {
				m.folder = m.cwd
			} else {
				m.folder = ""
			}
			saveFolderScope(m.folder != "")
			m.applyResults(m.runRank(m.query.Value()))
			return m, nil
		case "ctrl+r":
			m.role = nextRole(m.role)
			saveRole(m.role)
			m.applyResults(m.runRank(m.query.Value()))
			return m, nil
		case "ctrl+h":
			m.showHidden = !m.showHidden
			m.rebuildVisible()
			m.cursor = 0
			m.offset = 0
			m.detail = nil
			m.detailID = ""
			return m, nil
		case "ctrl+a":
			m.aiOn = !m.aiOn
			m.aiStatus = aiStatusInitial(m.aiOn)
			saveAI(m.aiOn)
			return m, m.scheduleRerank()
		case "ctrl+s", "alt+s":
			m.sort = m.sort.Next()
			saveSort(m.sort)
			m.applyResults(m.runRank(m.query.Value()))
			return m, nil
		case "down", "ctrl+n":
			m.moveCursor(1)
			return m, nil
		case "up", "ctrl+p":
			m.moveCursor(-1)
			return m, nil
		}
	case rerankDoneMsg:
		if msg.status != "" {
			m.aiStatus = msg.status
		}
		if len(msg.results) > 0 {
			m.applyResults(msg.results)
		}
	case debounceMsg:
		if msg.gen != m.gen {
			return m, nil // superseded by a newer keystroke
		}
		query := m.query.Value()
		gen := m.gen
		return m, func() tea.Msg {
			return searchDoneMsg{gen: gen, results: m.runRank(query)}
		}
	case searchDoneMsg:
		if msg.gen != m.gen {
			return m, nil // stale scan; a newer query is in flight
		}
		m.scanning = false
		m.applyResults(msg.results)
		return m, m.scheduleRerank()
	case spinnerMsg:
		if msg.gen != m.gen || !m.scanning {
			return m, nil
		}
		m.spinner++
		gen := m.gen
		return m, tea.Tick(90*time.Millisecond, func(time.Time) tea.Msg {
			return spinnerMsg{gen: gen}
		})
	}

	prev := m.query.Value()
	var qcmd tea.Cmd
	m.query, qcmd = m.query.Update(msg)
	cmds := []tea.Cmd{qcmd}
	if m.query.Value() != prev {
		// Debounce: bump generation and schedule the (async) scan after a
		// short pause. The scan never runs in Update, so typing never blocks.
		m.gen++
		m.scanning = true
		m.scanStart = time.Now()
		m.detail = nil
		m.detailID = ""
		gen := m.gen
		cmds = append(cmds,
			tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg {
				return debounceMsg{gen: gen}
			}),
			tea.Tick(90*time.Millisecond, func(time.Time) tea.Msg {
				return spinnerMsg{gen: gen}
			}),
		)
	}
	return m, tea.Batch(cmds...)
}

func (m *searchModel) moveCursor(delta int) {
	if len(m.results) == 0 {
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.results) {
		m.cursor = len(m.results) - 1
	}
	// Expansion is tied to the focused item; collapse when moving away.
	m.detail = nil
	m.detailID = ""
	m.clampScroll()
}

// toggleExpand loads (once) and toggles the extra match snippets for the
// currently selected session. They render inline directly under the selected
// item, in the same style as the result body, so the user sees other
// occurrences of the searched terms in place.
func (m *searchModel) toggleExpand() {
	c, ok := m.selected()
	if !ok {
		return
	}
	id := c.Session.ID
	if m.detailID == id {
		m.detail = nil
		m.detailID = ""
		return
	}
	pq := parseQuery(m.query.Value())
	snips := allMatchSnippets(c.Session.JsonlPath, pq, m.role, 8)
	if len(snips) == 0 {
		snips = []string{"(no other occurrences)"}
	}
	m.detail = snips
	m.detailID = id
	m.clampScroll()
}

// visibleRows is the number of terminal rows available for result lines
// (everything except header, filters, query, blank, and help).
func (m searchModel) visibleRows() int {
	r := m.height - 5
	if r < 3 {
		r = 3
	}
	return r
}

// itemRows is the rendered height of result i (3 content lines + 1 spacer,
// plus expanded occurrence lines when it is the focused/expanded item).
func (m searchModel) itemRows(i int) int {
	rows := 4
	if i == m.cursor && m.detailID != "" {
		rows += len(m.detail)
	}
	return rows
}

// clampScroll adjusts offset so the cursor item is fully visible.
func (m *searchModel) clampScroll() {
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	avail := m.visibleRows()
	for m.offset < m.cursor {
		h := 0
		for i := m.offset; i <= m.cursor; i++ {
			h += m.itemRows(i)
		}
		if h <= avail {
			break
		}
		m.offset++
	}
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
	searchDescStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))

	nameStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))  // cyan
	nameSelStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")) // pink
	hitsStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))            // orange
	srDateStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))            // gray
	pathStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))            // green
	providerPiSt  = lipgloss.NewStyle().Foreground(lipgloss.Color("75"))             // blue
	providerClSt  = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))            // purple
	aiScoreStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("219"))            // magenta
)

// providerBadge renders an emoji + colored provider label.
func providerBadge(p string) string {
	switch p {
	case "pi":
		return providerPiSt.Render("pi")
	case "claude":
		return providerClSt.Render("claude")
	default:
		return searchStatusStyle.Render(p)
	}
}

// renderCandidate returns the 3 styled lines for a single result.
func renderCandidate(c candidate, hlQuery string, selected bool) []string {
	s := c.Session
	name := s.Name
	if name == "" {
		name = synthName(s)
	}
	desc := s.Description
	if c.Snippet != "" {
		desc = c.Snippet
	} else if desc == "" {
		desc = bestSnippet(s, hlQuery, 140)
	}
	if c.Why != "" {
		desc = c.Why
	}

	hitLabel := ""
	switch {
	case c.Source == "ai":
		hitLabel = aiScoreStyle.Render(fmt.Sprintf("✨ ai %.2f", c.Score))
	case c.Hits > 0:
		word := "hit"
		if c.Hits != 1 {
			word = "hits"
		}
		hitLabel = hitsStyle.Render(fmt.Sprintf("🎯 %d %s", c.Hits, word))
	}

	cursor := "  "
	titleStyle := nameStyle
	if selected {
		cursor = "▶ "
		titleStyle = nameSelStyle
	}

	meta := srDateStyle.Render("🕘 "+sessionDate(s).Format("2006-01-02 15:04")) +
		"   " + providerBadge(s.Provider)
	if p := strings.TrimSpace(s.Path); p != "" {
		meta += "   " + pathStyle.Render("📁 "+shortPathSearch(p))
	}

	return []string{
		fmt.Sprintf("%s%s  %s", cursor, highlightMatches(name, hlQuery, titleStyle), hitLabel),
		"    " + meta,
		"    " + highlightMatches(truncateText(desc, 140), hlQuery, searchDescStyle),
	}
}

func (m searchModel) View() tea.View {
	if m.quit {
		return tea.View{}
	}
	if m.showHelp {
		v := tea.NewView(m.helpView())
		v.AltScreen = true
		return v
	}
	scan := ""
	if m.scanning {
		frames := []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}
		f := frames[m.spinner%len(frames)]
		scan = hitsStyle.Render(fmt.Sprintf(" · %c searching… %.1fs", f, time.Since(m.scanStart).Seconds()))
	}
	hidden := ""
	if m.hidden > 0 {
		hidden = searchHelpStyle.Render(fmt.Sprintf("  ·  %d hidden (ctrl+h)", m.hidden))
	}
	header := searchHeaderStyle.Render("cly as search") + "  " +
		searchStatusStyle.Render(fmt.Sprintf("%d results · sort: %s · %s",
			len(m.results), m.sort.Label(), m.aiStatus)) + scan + hidden
	folderLabel := "global"
	if m.folder != "" {
		folderLabel = shortPathSearch(m.folder)
	}
	filters := searchStatusStyle.Render(fmt.Sprintf("filters: folder:%s  role:%s  provider:%s",
		folderLabel, m.role, m.provider))
	hlQuery := strings.Join(parseQuery(m.query.Value()).positive, " ")

	// Render the visible window of results, inlining expanded occurrences
	// directly under the focused item.
	var body []string
	rows := 0
	avail := m.visibleRows()
	for i := m.offset; i < len(m.results); i++ {
		ih := m.itemRows(i)
		if rows > 0 && rows+ih > avail {
			break
		}
		body = append(body, renderCandidate(m.results[i], hlQuery, i == m.cursor)...)
		if i == m.cursor && m.detailID != "" {
			for _, snip := range m.detail {
				body = append(body, "    "+highlightMatches(truncateText(snip, 160), hlQuery, searchDescStyle))
			}
		}
		body = append(body, "") // spacer
		rows += ih
	}
	if len(m.results) == 0 {
		body = append(body, searchStatusStyle.Render("  no matches"))
	}

	help := searchHelpStyle.Render("↑↓ nav   enter resume   tab occurrences   ctrl+f folder   ctrl+r role   ctrl+s sort   ctrl+h hidden   ctrl+a ai   F1 syntax   esc quit")
	parts := []string{header, filters, m.query.View(), "", strings.Join(body, "\n"), help}
	v := tea.NewView(strings.Join(parts, "\n"))
	v.AltScreen = true
	return v
}

// helpView renders the search syntax reference (Microsoft Purview Unified
// Catalog syntax, plus `+`=AND and `|`=OR). Toggled with '?'.
func (m searchModel) helpView() string {
	title := searchHeaderStyle.Render("search syntax")
	rows := [][2]string{
		{"customer sales", "space = OR; more matches rank higher"},
		{"customer AND sales", "both terms present  (also: customer + sales)"},
		{"customer OR sales", "either term present  (also: customer | sales)"},
		{"customer NOT draft", "first present, second absent"},
		{`"sales report"`, "exact phrase, words in order"},
		{"(a OR b) AND c", "parentheses control precedence"},
		{"name:customer", "field-scoped: name, description, path, provider"},
		{"*  or empty", "match all sessions"},
	}
	var b []string
	b = append(b, title, "")
	for _, r := range rows {
		b = append(b, "  "+searchDescStyle.Render(fmt.Sprintf("%-22s", r[0]))+"  "+searchStatusStyle.Render(r[1]))
	}
	b = append(b, "", searchHelpStyle.Render("F1 toggles this · press any key to close"))
	return strings.Join(b, "\n")
}

func runSearchTUI(idx *searchIndex, providerFilter, folder, cwd, query string, aiOn bool, prefs searchPrefs) (*indexedSession, error) {
	m := newSearchModel(idx, providerFilter, folder, cwd, query, aiOn, prefs)
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
