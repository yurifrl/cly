package agentsession

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
)

var (
	editLabelStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("117")).Bold(true).Width(14)
	editActiveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))
	editDimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	editHelpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).MarginTop(1)
)

func editCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "edit [name]",
		Aliases: []string{"e"},
		Short:   "Edit a saved session's name and description",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			provider, err := providerFromCmd(cmd)
			if err != nil {
				return err
			}

			filePath := filePathFn()
			sessions, err := loadScopedSessions(cmd)
			if err != nil {
				return err
			}

			var entry *Entry
			if len(args) == 1 {
				entry = findEntry(sessions, provider.Name, args[0])
				if entry == nil {
					return fmt.Errorf("%s session %q not found", provider.Name, args[0])
				}
			} else {
				picked, _, err := runPicker(sessions, false)
				if err != nil {
					return err
				}
				if picked == nil {
					return nil
				}
				entry = picked
			}

			updated, err := runEditForm(entry)
			if err != nil {
				return err
			}
			if updated == nil {
				return nil
			}

			allSessions, err := Load(filePath)
			if err != nil {
				return err
			}

			oldName := entry.Name
			entry.Name = updated.Name
			entry.Description = updated.Description

			if oldName != entry.Name {
				allSessions = RemoveForProvider(allSessions, provider.Name, oldName)
			}
			upsertEntry(allSessions, *entry)

			if err := Save(filePath, allSessions); err != nil {
				return err
			}

			return jsonOut(cmd, entry)
			return nil
		},
	}
}

type editField int

const (
	editFieldName editField = iota
	editFieldDesc
	editFieldCount
)

type editModel struct {
	inputs [editFieldCount]textinput.Model
	focus  editField
	entry  *Entry
	done   bool
	cancel bool
}

type editResult struct {
	Name        string
	Description string
}

func newEditModel(entry *Entry) editModel {
	nameInput := textinput.New()
	nameInput.Placeholder = "session name"
	nameInput.SetValue(entry.Name)
	nameInput.Focus()
	nameInput.CharLimit = 64
	nameInput.SetWidth(50)

	descInput := textinput.New()
	descInput.Placeholder = "optional description"
	descInput.SetValue(entry.Description)
	descInput.CharLimit = 200
	descInput.SetWidth(50)

	return editModel{
		inputs: [editFieldCount]textinput.Model{nameInput, descInput},
		focus:  editFieldName,
		entry:  entry,
	}
}

func (m editModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m editModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancel = true
			return m, tea.Quit
		case "tab", "down":
			m.focus = (m.focus + 1) % editFieldCount
			return m, m.updateFocus()
		case "shift+tab", "up":
			m.focus = (m.focus - 1 + editFieldCount) % editFieldCount
			return m, m.updateFocus()
		case "enter":
			if m.focus == editFieldDesc {
				m.done = true
				return m, tea.Quit
			}
			m.focus++
			return m, m.updateFocus()
		}
	}

	cmd := m.updateInput(msg)
	return m, cmd
}

func (m *editModel) updateFocus() tea.Cmd {
	var cmds []tea.Cmd
	for i := range m.inputs {
		if editField(i) == m.focus {
			cmds = append(cmds, m.inputs[i].Focus())
		} else {
			m.inputs[i].Blur()
		}
	}
	return tea.Batch(cmds...)
}

func (m *editModel) updateInput(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)
	return cmd
}

func (m editModel) View() tea.View {
	var b strings.Builder

	b.WriteString("\n")
	labels := [editFieldCount]string{"  Name", "  Description"}
	for i, label := range labels {
		cursor := "  "
		style := editDimStyle
		if editField(i) == m.focus {
			cursor = editActiveStyle.Render("▸ ")
			style = lipgloss.NewStyle()
		}
		_ = style
		b.WriteString(fmt.Sprintf("%s%s %s\n", cursor, editLabelStyle.Render(label), m.inputs[i].View()))
	}

	b.WriteString(editHelpStyle.Render("  tab/↑↓ navigate • enter confirm • esc cancel"))
	b.WriteString("\n")

	return tea.NewView(b.String())
}

func runEditForm(entry *Entry) (*editResult, error) {
	m := newEditModel(entry)
	p := tea.NewProgram(m)
	final, err := p.Run()
	if err != nil {
		return nil, err
	}

	fm := final.(editModel)
	if fm.cancel {
		return nil, nil
	}
	if !fm.done {
		return nil, nil
	}

	name := strings.TrimSpace(fm.inputs[editFieldName].Value())
	if name == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}

	return &editResult{
		Name:        name,
		Description: strings.TrimSpace(fm.inputs[editFieldDesc].Value()),
	}, nil
}
