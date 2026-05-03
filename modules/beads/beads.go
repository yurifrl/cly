package beads

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// formWidth is the target inline width. Kept narrow so the form does not feel
// like it takes over the screen.
const formWidth = 64

// Curated type list. Edit to change the picker options.
var typeOptions = []string{
	"bug", "feature", "task", "epic", "chore", "decision",
}

var allowedTypes = func() map[string]bool {
	m := make(map[string]bool, len(typeOptions))
	for _, t := range typeOptions {
		m[t] = true
	}
	return m
}()

// bd accepts "0-4" or "P0-P4". We use the P-form in the UI and convert on
// submit to keep flag values canonical.
var priorityOptions = []string{"P0", "P1", "P2", "P3", "P4"}

const defaultPriority = "P2"

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

type typesLoadedMsg []string
type typesErrMsg struct{ err error }

// ---------------------------------------------------------------------------
// Styles
// ---------------------------------------------------------------------------

var (
	titleBar = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("63")).
			PaddingRight(1)

	badgeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("202")).
			Padding(0, 1).
			MarginLeft(1)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Width(9)

	activeLabelStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("63")).
				Width(9)

	sectionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Italic(true).
			MarginTop(1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			MarginTop(1)

	errStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	pillStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Padding(0, 1)

	pillActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("63")).
			Padding(0, 1)

	// Dim "current selection" used when the picker is not focused, so you
	// can still see the current value without it screaming for attention.
	pillSelectedDimStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("230")).
				Background(lipgloss.Color("237")).
				Padding(0, 1)
)

// ---------------------------------------------------------------------------
// Field identifiers
// ---------------------------------------------------------------------------

type fieldID int

const (
	fTitle fieldID = iota
	fDesc
	fType
	fPriority
	fLabels
	// Details (ctrl+d):
	fAcceptance
	fSkills
	fContext
	fDesign
	fNotes
)

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

type model struct {
	title      textinput.Model
	desc       textarea.Model
	typeP      picker
	priorityP  picker
	labels     textinput.Model
	acceptance textarea.Model
	skills     textinput.Model
	context    textinput.Model
	design     textarea.Model
	notes      textarea.Model

	dryRun bool

	detailsOpen bool
	focused     fieldID

	err       error
	submitted bool
	quitting  bool
}

type keymap struct {
	next, prev                  key.Binding
	pickerNext, pickerPrev      key.Binding
	toggleDryRun, toggleDetails key.Binding
	submit, quit                key.Binding
}

var keys = keymap{
	next:          key.NewBinding(key.WithKeys("tab")),
	prev:          key.NewBinding(key.WithKeys("shift+tab")),
	pickerNext:    key.NewBinding(key.WithKeys("right", "down", "ctrl+n")),
	pickerPrev:    key.NewBinding(key.WithKeys("left", "up", "ctrl+p")),
	toggleDryRun:  key.NewBinding(key.WithKeys("ctrl+r")),
	toggleDetails: key.NewBinding(key.WithKeys("ctrl+d")),
	// ctrl+enter needs Kitty keyboard protocol (iTerm2, WezTerm, Ghostty,
	// kitty, foot, Alacritty enhanced). Bubbletea v2 requests basic key
	// disambiguation by default.
	submit: key.NewBinding(key.WithKeys("ctrl+enter")),
	quit:   key.NewBinding(key.WithKeys("esc", "ctrl+c")),
}

func initialModel() model {
	st := loadState()

	title := textinput.New()
	title.Placeholder = "Write tests for X"
	title.Prompt = ""
	title.CharLimit = 200
	title.SetWidth(formWidth - 11)
	title.Focus()

	desc := textarea.New()
	desc.Placeholder = "Why this issue exists and what needs to be done…"
	desc.Prompt = "│ "
	desc.CharLimit = 4000
	desc.SetWidth(formWidth - 11)
	desc.SetHeight(4)
	desc.ShowLineNumbers = false

	typeP := picker{tag: "type", options: append([]string(nil), typeOptions...)}
	switch {
	case st.LastType != "" && allowedTypes[st.LastType]:
		typeP.setSelected(st.LastType)
	default:
		typeP.setSelected("task")
	}

	priorityP := picker{tag: "priority", options: append([]string(nil), priorityOptions...)}
	if st.LastPriority != "" {
		priorityP.setSelected(st.LastPriority)
	} else {
		priorityP.setSelected(defaultPriority)
	}

	labels := textinput.New()
	labels.Placeholder = "bug,backend,urgent"
	labels.Prompt = ""
	labels.CharLimit = 200
	labels.SetWidth(formWidth - 11)

	acceptance := textarea.New()
	acceptance.Placeholder = "What does done look like?"
	acceptance.Prompt = "│ "
	acceptance.CharLimit = 2000
	acceptance.SetWidth(formWidth - 11)
	acceptance.SetHeight(3)
	acceptance.ShowLineNumbers = false

	skills := textinput.New()
	skills.Placeholder = "go,postgres"
	skills.Prompt = ""
	skills.CharLimit = 200
	skills.SetWidth(formWidth - 11)

	contextIn := textinput.New()
	contextIn.Placeholder = "extra context one-liner"
	contextIn.Prompt = ""
	contextIn.CharLimit = 500
	contextIn.SetWidth(formWidth - 11)

	design := textarea.New()
	design.Placeholder = "Design notes / approach"
	design.Prompt = "│ "
	design.CharLimit = 4000
	design.SetWidth(formWidth - 11)
	design.SetHeight(3)
	design.ShowLineNumbers = false

	notes := textarea.New()
	notes.Placeholder = "Free-form notes"
	notes.Prompt = "│ "
	notes.CharLimit = 4000
	notes.SetWidth(formWidth - 11)
	notes.SetHeight(3)
	notes.ShowLineNumbers = false

	return model{
		title:      title,
		desc:       desc,
		typeP:      typeP,
		priorityP:  priorityP,
		labels:     labels,
		acceptance: acceptance,
		skills:     skills,
		context:    contextIn,
		design:     design,
		notes:      notes,
		focused:    fTitle,
	}
}

// ---------------------------------------------------------------------------
// Init / Update / View
// ---------------------------------------------------------------------------

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, loadTypesCmd())
}

// visibleFields returns the tab-traversable field order given the current
// details-expanded state.
func (m model) visibleFields() []fieldID {
	base := []fieldID{fTitle, fDesc, fType, fPriority, fLabels}
	if !m.detailsOpen {
		return base
	}
	return append(base, fAcceptance, fSkills, fContext, fDesign, fNotes)
}

func (m *model) advance(delta int) {
	fields := m.visibleFields()
	// Find current position.
	cur := 0
	for i, f := range fields {
		if f == m.focused {
			cur = i
			break
		}
	}
	next := (cur + delta + len(fields)) % len(fields)
	m.setFocus(fields[next])
}

func (m *model) setFocus(f fieldID) {
	m.focused = f
	m.title.Blur()
	m.desc.Blur()
	m.labels.Blur()
	m.acceptance.Blur()
	m.skills.Blur()
	m.context.Blur()
	m.design.Blur()
	m.notes.Blur()
	switch f {
	case fTitle:
		m.title.Focus()
	case fDesc:
		m.desc.Focus()
	case fLabels:
		m.labels.Focus()
	case fAcceptance:
		m.acceptance.Focus()
	case fSkills:
		m.skills.Focus()
	case fContext:
		m.context.Focus()
	case fDesign:
		m.design.Focus()
	case fNotes:
		m.notes.Focus()
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case typesLoadedMsg:
		if len(msg) > 0 {
			m.typeP.setOptions([]string(msg))
		}
		return m, nil

	case typesErrMsg:
		return m, nil

	case quickSelectExpireMsg:
		if m.typeP.onExpire(msg) {
			return m, nil
		}
		m.priorityP.onExpire(msg)
		return m, nil

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keys.quit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, keys.submit):
			if err := m.validate(); err != nil {
				m.err = err
				return m, nil
			}
			m.submitted = true
			return m, tea.Quit

		case key.Matches(msg, keys.toggleDryRun):
			m.dryRun = !m.dryRun
			return m, nil

		case key.Matches(msg, keys.toggleDetails):
			m.detailsOpen = !m.detailsOpen
			// If collapsing while focused on a hidden field, bounce to title.
			if !m.detailsOpen && m.focused >= fAcceptance {
				m.setFocus(fTitle)
			}
			return m, nil

		case key.Matches(msg, keys.next):
			m.advance(1)
			return m, nil

		case key.Matches(msg, keys.prev):
			m.advance(-1)
			return m, nil
		}

		// Picker-specific keys when focused on a picker field.
		if m.focused == fType || m.focused == fPriority {
			p := m.activePicker()
			switch {
			case key.Matches(msg, keys.pickerNext):
				p.next()
				return m, nil
			case key.Matches(msg, keys.pickerPrev):
				p.prev()
				return m, nil
			}
			if s := msg.String(); isQuickSelectKey(s) {
				return m, p.quickSelect(s)
			}
			if msg.String() == "backspace" {
				p.backspaceQS()
				return m, nil
			}
		}
	}

	// Route remaining messages to the focused text field.
	var cmd tea.Cmd
	switch m.focused {
	case fTitle:
		m.title, cmd = m.title.Update(msg)
	case fDesc:
		m.desc, cmd = m.desc.Update(msg)
	case fLabels:
		m.labels, cmd = m.labels.Update(msg)
	case fAcceptance:
		m.acceptance, cmd = m.acceptance.Update(msg)
	case fSkills:
		m.skills, cmd = m.skills.Update(msg)
	case fContext:
		m.context, cmd = m.context.Update(msg)
	case fDesign:
		m.design, cmd = m.design.Update(msg)
	case fNotes:
		m.notes, cmd = m.notes.Update(msg)
	}
	return m, cmd
}

// activePicker returns a pointer to whichever picker is focused; nil if
// neither picker is focused. Callers must check focus first.
func (m *model) activePicker() *picker {
	switch m.focused {
	case fType:
		return &m.typeP
	case fPriority:
		return &m.priorityP
	}
	return nil
}

func (m model) View() tea.View {
	if m.quitting || m.submitted {
		return tea.NewView("")
	}

	head := titleBar.Render("● beads · new issue")
	if m.dryRun {
		head += badgeStyle.Render("DRY-RUN")
	}

	rows := []string{
		head,
		m.row(fTitle, "title", m.title.View()),
		m.row(fDesc, "desc", m.desc.View()),
		m.row(fType, "type", m.typeP.view(m.focused == fType)),
		m.row(fPriority, "prio", m.priorityP.view(m.focused == fPriority)),
		m.row(fLabels, "labels", m.labels.View()),
	}

	if m.detailsOpen {
		rows = append(rows,
			sectionStyle.Render("▾ details"),
			m.row(fAcceptance, "accept", m.acceptance.View()),
			m.row(fSkills, "skills", m.skills.View()),
			m.row(fContext, "context", m.context.View()),
			m.row(fDesign, "design", m.design.View()),
			m.row(fNotes, "notes", m.notes.View()),
		)
	} else {
		rows = append(rows, sectionStyle.Render("▸ details (ctrl+d)"))
	}

	body := lipgloss.JoinVertical(lipgloss.Left, rows...)

	help := helpStyle.Render(
		"tab: move · letters/arrows on pickers · ctrl+r: dry-run · ctrl+d: details · ctrl+enter: submit · esc: cancel",
	)

	// Quick-select live buffer indicator.
	if p := m.activeBufPicker(); p != nil && p.qsBuf != "" {
		help = helpStyle.Render("→ matching: "+p.qsBuf) + "\n" + help
	}
	if m.err != nil {
		help = errStyle.Render("✖ "+m.err.Error()) + "\n" + help
	}

	return tea.NewView(body + "\n" + help + "\n")
}

// activeBufPicker returns the focused picker (read-only) if any.
func (m model) activeBufPicker() *picker {
	switch m.focused {
	case fType:
		return &m.typeP
	case fPriority:
		return &m.priorityP
	}
	return nil
}

func (m model) row(f fieldID, name, field string) string {
	lab := labelStyle
	if m.focused == f {
		lab = activeLabelStyle
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, lab.Render(name), field)
}

// ---------------------------------------------------------------------------
// Validation / submission
// ---------------------------------------------------------------------------

func (m *model) validate() error {
	if strings.TrimSpace(m.title.Value()) == "" {
		return fmt.Errorf("title is required")
	}
	if m.typeP.selected() == "" {
		return fmt.Errorf("type is required")
	}
	return nil
}

// runCreate shells out to `bd create` with collected values. Returns the
// trimmed stdout on success.
func (m model) runCreate() (string, error) {
	chosenType := m.typeP.selected()
	chosenPrio := m.priorityP.selected()

	args := []string{
		"create",
		"--title", strings.TrimSpace(m.title.Value()),
		"--description", m.desc.Value(),
		"--type", chosenType,
		"--priority", chosenPrio, // bd accepts "P0".."P4"
	}

	if v := strings.TrimSpace(m.labels.Value()); v != "" {
		args = append(args, "--labels", v)
	}
	if v := strings.TrimSpace(m.acceptance.Value()); v != "" {
		args = append(args, "--acceptance", v)
	}
	if v := strings.TrimSpace(m.skills.Value()); v != "" {
		args = append(args, "--skills", v)
	}
	if v := strings.TrimSpace(m.context.Value()); v != "" {
		args = append(args, "--context", v)
	}
	if v := strings.TrimSpace(m.design.Value()); v != "" {
		args = append(args, "--design", v)
	}
	if v := strings.TrimSpace(m.notes.Value()); v != "" {
		args = append(args, "--notes", v)
	}
	if m.dryRun {
		args = append(args, "--dry-run")
	}

	c := exec.Command("bd", args...)
	c.Stderr = os.Stderr
	out, err := c.Output()
	if err == nil && !m.dryRun {
		saveState(persistedState{
			LastType:     chosenType,
			LastPriority: chosenPrio,
		})
	}
	return strings.TrimRight(string(out), "\n"), err
}

// loadTypesCmd fetches types from `bd types --json`, filters through
// allowedTypes, and falls back silently to the hard-coded list.
func loadTypesCmd() tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("bd", "types", "--json").Output()
		if err != nil {
			return typesErrMsg{err}
		}
		var payload struct {
			CoreTypes []struct {
				Name string `json:"name"`
			} `json:"core_types"`
			CustomTypes []struct {
				Name string `json:"name"`
			} `json:"custom_types"`
		}
		if err := json.Unmarshal(out, &payload); err != nil {
			return typesErrMsg{err}
		}
		names := make([]string, 0, len(payload.CoreTypes)+len(payload.CustomTypes))
		for _, t := range payload.CoreTypes {
			if allowedTypes[t.Name] {
				names = append(names, t.Name)
			}
		}
		for _, t := range payload.CustomTypes {
			names = append(names, t.Name)
		}
		if len(names) == 0 {
			return typesErrMsg{fmt.Errorf("no types")}
		}
		return typesLoadedMsg(names)
	}
}
