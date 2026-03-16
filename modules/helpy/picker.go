package helpy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yurifrl/cly/pkg/llm"
)

// docEntry represents a discovered markdown doc file.
type docEntry struct {
	name        string // display name (relative path without .md)
	path        string // absolute path
	relPath     string // relative path from docs root
	description string // from frontmatter, if available
}

// pickerModel is the fuzzy doc picker TUI.
type pickerModel struct {
	docs         []docEntry
	filtered     []docEntry
	filterInput  textinput.Model
	selectedIdx  int
	width        int
	height       int
	viewing      bool       // true when viewing a doc
	viewer       *model     // the helpy viewer model
	viewerDoc    string     // name of currently viewed doc
	err          error
}

var (
	pickerTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205")).
				MarginBottom(1)
	pickerSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true)
	pickerNormalStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))
	pickerDimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
	pickerFooterStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241"))
)

// discoverDocs recursively finds all *.md files under docsDir.
func discoverDocs(docsDir string) ([]docEntry, error) {
	docsDir = expandPath(docsDir)

	var docs []docEntry
	err := filepath.Walk(docsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible paths
		}
		if info.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(path)) != ".md" {
			return nil
		}

		rel, _ := filepath.Rel(docsDir, path)
		name := strings.TrimSuffix(rel, ".md")

		// Try to read frontmatter for description
		var description string
		if content, readErr := os.ReadFile(path); readErr == nil {
			meta, _ := parseFrontmatter(string(content))
			description = meta.Description
			if meta.Name != "" {
				name = meta.Name
			}
		}

		docs = append(docs, docEntry{
			name:        name,
			path:        path,
			relPath:     rel,
			description: description,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking docs dir %s: %w", docsDir, err)
	}
	if len(docs) == 0 {
		return nil, fmt.Errorf("no .md files found in %s", docsDir)
	}
	return docs, nil
}

func newPickerModel(docs []docEntry) pickerModel {
	ti := textinput.New()
	ti.Placeholder = "type to filter docs..."
	ti.CharLimit = 100
	ti.Width = 50
	ti.Focus()

	return pickerModel{
		docs:     docs,
		filtered: docs,
		filterInput: ti,
	}
}

func (m pickerModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// If viewing a doc, delegate to viewer
	if m.viewing {
		return m.updateViewer(msg)
	}
	return m.updatePicker(msg)
}

func (m pickerModel) updateViewer(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// Back to picker
			m.viewing = false
			m.viewer = nil
			m.filterInput.Focus()
			return m, textinput.Blink
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	if m.viewer != nil {
		updated, cmd := m.viewer.Update(msg)
		m.viewer = updated.(*model)
		return m, cmd
	}
	return m, nil
}

func (m pickerModel) updatePicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.filterInput.Value() != "" {
				m.filterInput.Reset()
				m.filtered = m.docs
				m.selectedIdx = 0
				return m, nil
			}
			return m, tea.Quit
		case "up", "ctrl+k":
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}
			return m, nil
		case "down", "ctrl+j":
			if m.selectedIdx < len(m.filtered)-1 {
				m.selectedIdx++
			}
			return m, nil
		case "enter":
			if len(m.filtered) > 0 && m.selectedIdx < len(m.filtered) {
				return m.openDoc(m.filtered[m.selectedIdx])
			}
			return m, nil
		default:
			var cmd tea.Cmd
			m.filterInput, cmd = m.filterInput.Update(msg)
			m.filterDocs()
			return m, cmd
		}
	}

	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	return m, cmd
}

func (m *pickerModel) filterDocs() {
	query := strings.ToLower(m.filterInput.Value())
	if query == "" {
		m.filtered = m.docs
		m.selectedIdx = 0
		return
	}

	var filtered []docEntry
	for _, d := range m.docs {
		if fuzzyMatch(strings.ToLower(d.name), query) {
			filtered = append(filtered, d)
		}
	}
	m.filtered = filtered
	if m.selectedIdx >= len(m.filtered) {
		m.selectedIdx = max(0, len(m.filtered)-1)
	}
}

// fuzzyMatch checks if all chars of query appear in s in order.
func fuzzyMatch(s, query string) bool {
	qi := 0
	for i := 0; i < len(s) && qi < len(query); i++ {
		if s[i] == query[qi] {
			qi++
		}
	}
	return qi == len(query)
}

func (m pickerModel) openDoc(doc docEntry) (tea.Model, tea.Cmd) {
	rawContent, err := readFileContent(doc.path)
	if err != nil {
		m.err = fmt.Errorf("reading %s: %w", doc.path, err)
		return m, nil
	}

	// Parse frontmatter
	meta, content := parseFrontmatter(rawContent)

	viewer, err := initialModel(content)
	if err != nil {
		m.err = fmt.Errorf("creating viewer: %w", err)
		return m, nil
	}

	// Set up AI chat if configured
	aiCfg := loadAIConfig()
	if aiCfg != nil {
		client, clientErr := llm.NewClient(*aiCfg)
		if clientErr == nil {
			viewer.chat = newChatModel(client, aiCfg.SystemPrompt, content, meta)
			viewer.chatEnabled = true
		}
	}

	// Resize viewer to current dimensions
	if m.width > 0 && m.height > 0 {
		viewer.width = m.width
		viewer.height = m.height
		viewer.viewport.Width = m.width
		viewer.viewport.Height = m.height - 2
		viewer.setContent(content, m.width)
	}

	// Use frontmatter name if available, otherwise use file name
	displayName := doc.name
	if meta.Name != "" {
		displayName = meta.Name
	}

	m.viewing = true
	m.viewer = viewer
	m.viewerDoc = displayName
	m.filterInput.Blur()
	return m, nil
}

func (m pickerModel) View() string {
	if m.viewing && m.viewer != nil {
		docInfo := pickerDimStyle.Render(fmt.Sprintf("  📄 %s  (esc: back to picker)", m.viewerDoc))
		return docInfo + "\n" + m.viewer.View()
	}

	var b strings.Builder

	b.WriteString(pickerTitleStyle.Render("📚 Docs Browser"))
	b.WriteString("\n")
	b.WriteString("  " + m.filterInput.View())
	b.WriteString("\n")
	b.WriteString(pickerDimStyle.Render("  " + strings.Repeat("─", 50)))
	b.WriteString("\n")

	if m.err != nil {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(
			fmt.Sprintf("  Error: %v", m.err)))
		b.WriteString("\n")
	}

	// Calculate visible items based on terminal height
	maxVisible := m.height - 6 // header + filter + separator + footer
	if maxVisible < 3 {
		maxVisible = 10
	}

	// Scroll window around selected item
	start := 0
	if m.selectedIdx >= maxVisible {
		start = m.selectedIdx - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	for i := start; i < end; i++ {
		d := m.filtered[i]
		// Show folder prefix dimmed if in subdirectory
		parts := strings.Split(d.name, string(filepath.Separator))
		var display string
		if len(parts) > 1 {
			folder := strings.Join(parts[:len(parts)-1], "/") + "/"
			file := parts[len(parts)-1]
			if i == m.selectedIdx {
				display = pickerSelectedStyle.Render("> " + folder + file)
			} else {
				display = pickerNormalStyle.Render("  ") +
					pickerDimStyle.Render(folder) +
					pickerNormalStyle.Render(file)
			}
		} else {
			if i == m.selectedIdx {
				display = pickerSelectedStyle.Render("> " + d.name)
			} else {
				display = pickerNormalStyle.Render("  " + d.name)
			}
		}
		b.WriteString(display)
		if d.description != "" {
			b.WriteString("\n")
			b.WriteString(pickerDimStyle.Render("    " + d.description))
		}
		b.WriteString("\n")
	}

	if len(m.filtered) == 0 {
		b.WriteString(pickerDimStyle.Render("  no matching docs"))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	count := fmt.Sprintf("%d/%d docs", len(m.filtered), len(m.docs))
	b.WriteString(pickerFooterStyle.Render(
		fmt.Sprintf("  ↑/↓: Navigate • enter: Open • esc: %s • %s",
			func() string {
				if m.filterInput.Value() != "" {
					return "Clear"
				}
				return "Quit"
			}(), count)))

	return b.String()
}
