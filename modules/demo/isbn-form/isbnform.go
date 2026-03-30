package isbnform

import (
	"fmt"
	"strings"
	"unicode"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/exp/charmtone"
	"github.com/spf13/cobra"
)

type errMsg error

var (
	inputStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(charmtone.Tang.Hex()))
	continueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(charmtone.Anchovy.Hex()))
	validStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(charmtone.Guac.Hex()))
	errStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color(charmtone.Cherry.Hex()))
)

type model struct {
	isbnInput    textinput.Model
	titleInput   textinput.Model
	focusedInput int
	err          error
}

func (m model) canFindBook() bool {
	correctIsbnGiven := m.isbnInput.Err == nil && len(m.isbnInput.Value()) != 0
	correctTitleGiven := m.titleInput.Err == nil && len(m.titleInput.Value()) != 0
	return correctIsbnGiven && correctTitleGiven
}

func isbn13Validator(s string) error {
	s = strings.ReplaceAll(s, "-", "")
	if len(s) != 13 {
		return fmt.Errorf("ISBN is of wrong length")
	}
	for _, c := range s {
		if !unicode.IsDigit(c) {
			return fmt.Errorf("ISBN contains invalid characters")
		}
	}
	gs1Prefix := s[:3]
	switch gs1Prefix {
	case "978", "979":
	default:
		return fmt.Errorf("ISBN has invalid GS1 prefix")
	}
	sum := 0
	for i, c := range s {
		n := int(c - '0')
		if i%2 != 0 {
			n *= 3
		}
		sum += n
	}
	if sum%10 != 0 {
		return fmt.Errorf("ISBN has invalid check digit")
	}
	return nil
}

var bannedTitleWords = []string{
	"very", "bad", "words", "that", "should", "not", "appear", "in", "book", "titles",
}

func bookTitleValidator(s string) error {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return fmt.Errorf("Book title is empty")
	}
	for _, bannedWord := range bannedTitleWords {
		if strings.Contains(s, bannedWord) {
			return fmt.Errorf("Book title contains banned word %q", bannedWord)
		}
	}
	return nil
}

func initialModel() model {
	isbnInput := textinput.New()
	isbnInput.Focus()
	isbnInput.Placeholder = "978-X-XXX-XXXXX-X"
	isbnInput.CharLimit = 17
	isbnInput.SetWidth(30)
	isbnInput.Prompt = ""
	isbnInput.Validate = isbn13Validator

	titleInput := textinput.New()
	titleInput.Blur()
	titleInput.Placeholder = "Title"
	titleInput.CharLimit = 100
	titleInput.SetWidth(100)
	titleInput.Prompt = ""
	titleInput.Validate = bookTitleValidator

	return model{
		isbnInput:    isbnInput,
		titleInput:   titleInput,
		focusedInput: 0,
		err:          nil,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "down":
			switch m.focusedInput {
			case 0:
				m.focusedInput = 1
				m.titleInput.Focus()
				m.isbnInput.Blur()
			case 1:
				m.focusedInput = 0
				m.isbnInput.Focus()
				m.titleInput.Blur()
			}
		case "enter":
			if m.canFindBook() {
				return m, tea.Quit
			}
		case "ctrl+c", "esc":
			return m, tea.Quit
		}
	case errMsg:
		m.err = msg
		return m, nil
	}
	var isbnCommand tea.Cmd
	m.isbnInput, isbnCommand = m.isbnInput.Update(msg)
	var titleCommand tea.Cmd
	m.titleInput, titleCommand = m.titleInput.Update(msg)
	return m, tea.Batch(isbnCommand, titleCommand)
}

func (m model) View() tea.View {
	var continueText string
	if m.canFindBook() {
		continueText = continueStyle.Render("Find ->")
	}
	var isbnErrorText string
	if m.isbnInput.Value() != "" {
		if m.isbnInput.Err != nil {
			isbnErrorText = errStyle.Render(m.isbnInput.Err.Error())
		} else {
			isbnErrorText = validStyle.Render("Valid ISBN")
		}
	}
	var titleErrorText string
	if m.titleInput.Value() != "" {
		if m.titleInput.Err != nil {
			titleErrorText = errStyle.Render(m.titleInput.Err.Error())
		} else {
			titleErrorText = validStyle.Render("Valid title")
		}
	}
	return tea.NewView(fmt.Sprintf(
		` Search book:
 %s
 %s
 %s

 %s
 %s
 %s

 %s
`,
		inputStyle.Width(30).Render("ISBN"),
		m.isbnInput.View(),
		isbnErrorText,
		inputStyle.Width(30).Render("Title"),
		m.titleInput.View(),
		titleErrorText,
		continueText,
	) + "\n")
}

func run(cmd *cobra.Command, args []string) error {
	p := tea.NewProgram(initialModel())
	_, err := p.Run()
	return err
}
