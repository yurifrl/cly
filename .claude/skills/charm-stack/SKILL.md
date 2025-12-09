---
name: charm-stack
description: Build terminal UIs with Charm ecosystem (Bubbletea, Bubbles, Lipgloss, Huh). Use when building TUIs, handling terminal UI components, styling terminal output, or when user mentions Bubbletea, Elm architecture, terminal forms, or Charm libraries.
---

# Charm Stack TUI Development

Build elegant terminal user interfaces using the Charm ecosystem: Bubbletea (framework), Bubbles (components), Lipgloss (styling), and Huh (forms).

## Your Role: TUI Architect

You help build terminal UIs following the Elm architecture pattern. You:

✅ **Guide Bubbletea implementation** - Model-Update-View pattern
✅ **Recommend Bubbles components** - Spinner, List, Table, TextInput, etc.
✅ **Apply Lipgloss styling** - Colors, borders, layouts
✅ **Use Huh for forms** - Multi-field input collection
✅ **Follow Elm patterns** - Pure functions, immutable updates
✅ **Reference working examples** - Use references/bubbletea/examples/

❌ **Do NOT fight the framework** - Follow Elm architecture
❌ **Do NOT mutate state directly** - Return new state from Update
❌ **Do NOT block in Update** - Use Cmd for async operations

## Core Principles

### 1. Elm Architecture (Model-Update-View)

**Model**: Application state
**Update**: State transitions based on messages
**View**: Pure rendering function
**Cmd**: Side effects (async operations, timers)

**✅ GOOD:**
```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.String() == "q" {
            return m, tea.Quit  // Return new state + command
        }
    }
    return m, nil
}
```

**❌ BAD:**
```go
func (m *model) Update(msg tea.Msg) {
    // Mutating state directly
    m.count++
}
```

### 2. Component Lifecycle

**Init()**: Initial commands (start spinner, load data)
**Update()**: Process messages, return new state
**View()**: Render current state to string

```go
type model struct {
    spinner spinner.Model
    loading bool
}

func (m model) Init() tea.Cmd {
    return m.spinner.Tick  // Return initial command
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case spinner.TickMsg:
        var cmd tea.Cmd
        m.spinner, cmd = m.spinner.Update(msg)
        return m, cmd
    case tea.KeyMsg:
        if msg.String() == "q" {
            return m, tea.Quit
        }
    }
    return m, nil
}

func (m model) View() string {
    if m.loading {
        return m.spinner.View() + " Loading...\n"
    }
    return "Done!\n"
}
```

## Bubbletea Framework

### Program Initialization

**Basic:**
```go
p := tea.NewProgram(initialModel())
if _, err := p.Run(); err != nil {
    return err
}
```

**With options:**
```go
p := tea.NewProgram(
    initialModel(),
    tea.WithAltScreen(),      // Fullscreen mode
    tea.WithMouseAllMotion(), // Mouse tracking
    tea.WithReportFocus(),    // Focus/blur events
)
```

### Message Types

**Builtin:**
- `tea.KeyMsg` - Keyboard input
- `tea.MouseMsg` - Mouse events
- `tea.WindowSizeMsg` - Terminal resize
- `tea.BlurMsg`/`tea.FocusMsg` - Focus events

**Custom:**
```go
type dataLoadedMsg struct{ data []string }

func loadData() tea.Msg {
    // Simulate loading
    time.Sleep(1 * time.Second)
    return dataLoadedMsg{data: []string{"item1", "item2"}}
}

// In Update():
case dataLoadedMsg:
    m.items = msg.data
    m.loading = false
    return m, nil
```

### Commands (Cmd)

Commands are functions that return messages:

```go
// Simple command
func doSomething() tea.Msg {
    return someMsg{}
}

// Command with cleanup
func tick() tea.Cmd {
    return tea.Tick(time.Second, func(t time.Time) tea.Msg {
        return tickMsg(t)
    })
}

// Batch commands
return m, tea.Batch(cmd1, cmd2, cmd3)
```

## Bubbles Components

### Spinner

```go
import "github.com/charmbracelet/bubbles/spinner"

s := spinner.New()
s.Spinner = spinner.Dot
s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

// In Update():
case spinner.TickMsg:
    m.spinner, cmd = m.spinner.Update(msg)
    return m, cmd

// In View():
return m.spinner.View() + " Loading..."
```

### List

```go
import "github.com/charmbracelet/bubbles/list"

type item struct{ title, desc string }
func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

items := []list.Item{
    item{title: "Item 1", desc: "Description"},
}

l := list.New(items, list.NewDefaultDelegate(), 0, 0)
l.Title = "My List"

// In Update():
case tea.KeyMsg:
    m.list, cmd = m.list.Update(msg)
    return m, cmd

// Get selected:
selected := m.list.SelectedItem().(item)
```

### Table

```go
import "github.com/charmbracelet/bubbles/table"

columns := []table.Column{
    {Title: "Name", Width: 20},
    {Title: "Age", Width: 10},
}

rows := []table.Row{
    {"Alice", "30"},
    {"Bob", "25"},
}

t := table.New(
    table.WithColumns(columns),
    table.WithRows(rows),
    table.WithFocused(true),
    table.WithHeight(7),
)

// In Update():
m.table, cmd = m.table.Update(msg)
```

### TextInput

```go
import "github.com/charmbracelet/bubbles/textinput"

ti := textinput.New()
ti.Placeholder = "Enter name..."
ti.Focus()

// In Update():
m.textinput, cmd = m.textinput.Update(msg)

// Get value:
value := m.textinput.Value()
```

### Textarea

```go
import "github.com/charmbracelet/bubbles/textarea"

ta := textarea.New()
ta.Placeholder = "Type something..."
ta.Focus()

// In Update():
m.textarea, cmd = m.textarea.Update(msg)
```

### Progress

```go
import "github.com/charmbracelet/bubbles/progress"

prog := progress.New(progress.WithDefaultGradient())

// In View():
return prog.ViewAs(0.45) // 45% progress
```

## Lipgloss Styling

### Basic Styles

```go
import "github.com/charmbracelet/lipgloss"

style := lipgloss.NewStyle().
    Foreground(lipgloss.Color("205")).
    Background(lipgloss.Color("235")).
    Bold(true).
    Padding(1, 2)

text := style.Render("Hello!")
```

### Colors

```go
// ANSI colors (0-255)
lipgloss.Color("205")

// Hex colors
lipgloss.Color("#FF5F87")

// Adaptive (light/dark terminal)
lipgloss.AdaptiveColor{Light: "#000", Dark: "#FFF"}
```

### Layout

**Join vertically:**
```go
lipgloss.JoinVertical(
    lipgloss.Left,
    "Line 1",
    "Line 2",
)
```

**Join horizontally:**
```go
lipgloss.JoinHorizontal(
    lipgloss.Top,
    "Column 1",
    "Column 2",
)
```

**Borders:**
```go
style := lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    BorderForeground(lipgloss.Color("62")).
    Padding(1)
```

### Common Patterns

**Title bar:**
```go
titleStyle := lipgloss.NewStyle().
    Bold(true).
    Background(lipgloss.Color("62")).
    Foreground(lipgloss.Color("230")).
    Padding(0, 1)

title := titleStyle.Render("My App")
```

**Box with content:**
```go
boxStyle := lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    Padding(1, 2).
    Width(50)

box := boxStyle.Render(content)
```

## Huh Forms

### Simple Form

```go
import "github.com/charmbracelet/huh"

var name, email string

form := huh.NewForm(
    huh.NewGroup(
        huh.NewInput().
            Title("Name").
            Value(&name),

        huh.NewInput().
            Title("Email").
            Value(&email).
            Validate(validateEmail),
    ),
)

err := form.Run()
// name and email are populated
```

### Multi-page Form

```go
form := huh.NewForm(
    huh.NewGroup(
        huh.NewInput().Title("Name").Value(&name),
    ),
    huh.NewGroup(
        huh.NewSelect[string]().
            Title("Choose option").
            Options(
                huh.NewOption("Option 1", "opt1"),
                huh.NewOption("Option 2", "opt2"),
            ).
            Value(&choice),
    ),
    huh.NewGroup(
        huh.NewConfirm().
            Title("Continue?").
            Value(&confirmed),
    ),
)
```

### Field Types

**Input:**
```go
huh.NewInput().
    Title("Username").
    Placeholder("johndoe").
    Value(&username)
```

**Select:**
```go
huh.NewSelect[string]().
    Title("Pick color").
    Options(
        huh.NewOption("Red", "red"),
        huh.NewOption("Blue", "blue"),
    ).
    Value(&color)
```

**MultiSelect:**
```go
huh.NewMultiSelect[string]().
    Title("Pick toppings").
    Options(
        huh.NewOption("Cheese", "cheese"),
        huh.NewOption("Pepperoni", "pepperoni"),
    ).
    Value(&toppings)
```

**Confirm:**
```go
huh.NewConfirm().
    Title("Delete?").
    Affirmative("Yes").
    Negative("No").
    Value(&confirmed)
```

**Text (multiline):**
```go
huh.NewText().
    Title("Description").
    Value(&description)
```

## Patterns & Best Practices

### Component Delegation

**Delegate messages to components:**
```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd

    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.String() == "q" {
            return m, tea.Quit
        }
    }

    // Delegate to component
    m.list, cmd = m.list.Update(msg)
    return m, cmd
}
```

### Focus Management

```go
type model struct {
    inputs    []textinput.Model
    focusIndex int
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "tab":
            m.inputs[m.focusIndex].Blur()
            m.focusIndex = (m.focusIndex + 1) % len(m.inputs)
            m.inputs[m.focusIndex].Focus()
            return m, nil
        }
    }

    // Update focused input only
    var cmd tea.Cmd
    m.inputs[m.focusIndex], cmd = m.inputs[m.focusIndex].Update(msg)
    return m, cmd
}
```

### Loading States

```go
type model struct {
    state   string // "loading", "ready", "error"
    content string
    err     error
}

func (m model) View() string {
    switch m.state {
    case "loading":
        return "Loading..."
    case "error":
        return fmt.Sprintf("Error: %v", m.err)
    case "ready":
        return m.content
    default:
        return ""
    }
}
```

### Responsive Layout

```go
func (m model) View() string {
    width := m.width
    height := m.height

    // Adjust based on terminal size
    if width < 80 {
        return m.renderCompact()
    }
    return m.renderFull()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height

        // Update component sizes
        m.list.SetSize(msg.Width, msg.Height-4)
        return m, nil
    }
    return m, nil
}
```

## Common Pitfalls

**❌ Blocking in Update:**
```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    time.Sleep(1 * time.Second)  // NEVER do this
    return m, nil
}
```

**✅ Use Cmd:**
```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
        return tickMsg{}
    })
}
```

**❌ Not handling Quit:**
```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Missing quit handler - app won't exit
    return m, nil
}
```

**✅ Handle Quit:**
```go
case tea.KeyMsg:
    switch msg.String() {
    case "q", "ctrl+c":
        return m, tea.Quit
    }
```

## Reference Examples

This project has 48 working examples in `references/bubbletea/examples/`:

- `spinner/` - Loading indicators
- `list-simple/` - Selection lists
- `table/` - Data tables
- `textinput/` - Single-line input
- `textarea/` - Multi-line input
- `chat/` - Combined components
- `progress-static/` - Progress bars
- `file-picker/` - File selection

**Always reference these before implementing new patterns.**

## Checklist

- [ ] Model contains all state
- [ ] Init() returns initial Cmd
- [ ] Update() pure (no mutations, no blocking)
- [ ] Update() delegates to components
- [ ] View() pure rendering
- [ ] Handles "q" and "ctrl+c" for quit
- [ ] Uses tea.Cmd for async operations
- [ ] Responsive to tea.WindowSizeMsg
- [ ] Styles with Lipgloss
- [ ] References working examples

## Resources

- [Bubbletea Docs](https://github.com/charmbracelet/bubbletea)
- [Bubbles Components](https://github.com/charmbracelet/bubbles)
- [Lipgloss Styling](https://github.com/charmbracelet/lipgloss)
- [Huh Forms](https://github.com/charmbracelet/huh)
- Working examples: `references/bubbletea/examples/`
