---
name: charm-stack
description: Build terminal UIs with Bubbletea v2, Bubbles v2, Lipgloss v2, and Huh v2. Use when creating TUI applications, interactive forms, styled terminal output, or when user mentions Bubbletea, Bubbles, Lipgloss, Huh, Charm, or TUI development.
---

# Charm Stack v2 TUI Development

Build terminal UIs using the Charm v2 stack (released Feb 2026):
- **Bubble Tea v2** — TUI framework (Elm Architecture)
- **Bubbles v2** — Reusable TUI components
- **Lip Gloss v2** — Terminal styling & layout
- **Huh v2** — Interactive forms & prompts

## Installation

```bash
go get charm.land/bubbletea/v2
go get charm.land/bubbles/v2
go get charm.land/lipgloss/v2
go get charm.land/huh/v2
```

> **Import paths changed in v2** — use `charm.land/` not `github.com/charmbracelet/`.

## Your Role: TUI Architect

✅ Use Elm Architecture (Model-Update-View)
✅ Return `tea.View` from View() (not string)
✅ Use `tea.KeyPressMsg` (not `tea.KeyMsg`)
✅ Declare terminal features in View() (not commands)
✅ Use getter/setter methods on Bubbles components
✅ Use functional options for Bubbles constructors
✅ Follow CLY module patterns

❌ Do NOT return string from View()
❌ Do NOT use `tea.EnterAltScreen` commands
❌ Do NOT use `tea.KeyMsg` for key matching (use `tea.KeyPressMsg`)
❌ Do NOT use `lipgloss.AdaptiveColor` (removed — use `LightDark`)
❌ Do NOT set Bubbles fields directly (use SetWidth/SetHeight)
❌ Do NOT use old import paths (`github.com/charmbracelet/...`)

---

## Core Architecture: The Elm Architecture

### The Three Functions

```go
// Model — application state
type model struct {
    cursor   int
    choices  []string
    selected map[int]struct{}
}

// Init — initial command
func (m model) Init() tea.Cmd {
    return nil
}

// Update — handle messages, return updated model + command
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyPressMsg:
        switch msg.String() {
        case "ctrl+c", "q":
            return m, tea.Quit
        }
    }
    return m, nil
}

// View — returns tea.View (NOT string)
func (m model) View() tea.View {
    s := "Hello, World!\n"
    return tea.NewView(s)
}
```

### Message Flow

```
User Input → Msg → Update → Model → View → Screen
    ↑                                    ↓
    └────────── tea.Cmd ─────────────────┘
```

- Messages are immutable events
- Update returns new model (don't mutate)
- Commands run async, generate more messages
- View is a pure function of model state

---

## Bubble Tea v2

### Basic Program

```go
package main

import (
    "fmt"
    "os"

    tea "charm.land/bubbletea/v2"
)

type model struct {
    choices  []string
    cursor   int
    selected map[int]struct{}
}

func initialModel() model {
    return model{
        choices:  []string{"Buy carrots", "Buy celery", "Buy kohlrabi"},
        selected: make(map[int]struct{}),
    }
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyPressMsg:
        switch msg.String() {
        case "ctrl+c", "q":
            return m, tea.Quit
        case "up", "k":
            if m.cursor > 0 {
                m.cursor--
            }
        case "down", "j":
            if m.cursor < len(m.choices)-1 {
                m.cursor++
            }
        case "enter", "space":
            if _, ok := m.selected[m.cursor]; ok {
                delete(m.selected, m.cursor)
            } else {
                m.selected[m.cursor] = struct{}{}
            }
        }
    }
    return m, nil
}

func (m model) View() tea.View {
    s := "What should we buy at the market?\n\n"
    for i, choice := range m.choices {
        cursor := " "
        if m.cursor == i {
            cursor = ">"
        }
        checked := " "
        if _, ok := m.selected[i]; ok {
            checked = "x"
        }
        s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice)
    }
    s += "\nPress q to quit.\n"
    return tea.NewView(s)
}

func main() {
    p := tea.NewProgram(initialModel())
    if _, err := p.Run(); err != nil {
        fmt.Printf("Error: %v", err)
        os.Exit(1)
    }
}
```

### Declarative View (v2 Key Change)

In v2, `View()` returns `tea.View` instead of `string`. Terminal features are declared on the view struct — no more imperative commands:

```go
func (m model) View() tea.View {
    v := tea.NewView(m.renderContent())

    // Declare terminal features (replaces tea.EnterAltScreen, tea.EnableMouseCellMotion, etc.)
    v.AltScreen = true
    v.MouseMode = tea.MouseModeCellMotion
    v.ReportFocus = true
    v.WindowTitle = "My App"

    // Real cursor support
    if m.showCursor {
        v.Cursor = &tea.Cursor{
            Position: tea.Position{X: m.cursorX, Y: m.cursorY},
            Shape:    tea.CursorBlock,
            Blink:    true,
            Color:    lipgloss.Green,
        }
    }

    // Keyboard enhancements (progressive, supported terminals only)
    v.KeyboardEnhancements.ReportEventTypes = true // key release events

    return v
}
```

**View struct fields:**
```go
type View struct {
    Content                   string
    OnMouse                   func(msg MouseMsg) Cmd
    Cursor                    *Cursor
    BackgroundColor           color.Color
    ForegroundColor           color.Color
    WindowTitle               string
    ProgressBar               *ProgressBar
    AltScreen                 bool
    ReportFocus               bool
    DisableBracketedPasteMode bool
    MouseMode                 MouseMode
    KeyboardEnhancements      KeyboardEnhancements
}
```

### Key Messages (v2 Breaking Change)

Keys are now split into `tea.KeyPressMsg` and `tea.KeyReleaseMsg`. Use `msg.String()` for matching:

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyPressMsg:
        switch msg.String() {
        case "q", "ctrl+c":
            return m, tea.Quit
        case "space":
            // Space bar returns "space" (not " ")
        case "shift+enter":
            // New! Not possible in v1
        case "ctrl+alt+super+enter":
            // Full modifier support
        }
    case tea.KeyReleaseMsg:
        // Key release events (requires keyboard enhancements)
    }
    return m, nil
}
```

**Key struct fields:**
- `key.Code` — replaces `key.Type`
- `key.Text` — replaces `key.Runes`
- `key.Mod` — replaces separate Alt bool
- `key.BaseCode` — key on US PC-101 layout (handy for international keyboards)
- `key.IsRepeat` — auto-repeating held key
- `key.Keystroke()` — always includes modifier info (e.g. `"ctrl+shift+alt+a"`)

### Mouse Messages (v2 Breaking Change)

Split into specific types:

```go
case tea.MouseClickMsg:
    if msg.Button == tea.MouseLeft {
        // Left click
    }
case tea.MouseReleaseMsg:
    // Mouse button released
case tea.MouseWheelMsg:
    // Scroll
case tea.MouseMotionMsg:
    // Mouse moved
```

Mouse mode is set declaratively in View():
```go
func (m model) View() tea.View {
    v := tea.NewView("Move that mouse!")
    v.MouseMode = tea.MouseModeAllMotion
    return v
}
```

### Paste Messages

```go
case tea.PasteMsg:
    m.text += msg.Content
case tea.PasteStartMsg:
    // Paste started
case tea.PasteEndMsg:
    // Paste ended
```

### Commands

```go
// One-off message (function returning tea.Msg)
func checkServer() tea.Msg {
    res, err := http.Get("https://example.com")
    if err != nil {
        return errMsg{err}
    }
    return statusMsg(res.StatusCode)
}

// Usage in Update:
return m, checkServer // pass the function directly

// Command returning a Cmd (for wrapping)
func fetchData() tea.Cmd {
    return func() tea.Msg {
        // async work...
        return dataMsg{data}
    }
}

// Ticking (timers)
type tickMsg time.Time
func tick() tea.Cmd {
    return tea.Tick(time.Second, func(t time.Time) tea.Msg {
        return tickMsg(t)
    })
}

// Batch commands
return m, tea.Batch(cmdA, cmdB, cmdC)

// Print above the TUI
return m, tea.Println("Status: done")

// Clipboard (v2 new!)
return m, tea.SetClipboard("copied text")
return m, tea.ReadClipboard()
// Listen for: case tea.ClipboardMsg:
```

### Background Color Detection

```go
func (m model) Init() tea.Cmd {
    return tea.RequestBackgroundColor
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.BackgroundColorMsg:
        m.isDark = msg.IsDark()
        m.styles = newStyles(msg.IsDark())
    }
    return m, nil
}
```

### Environment Variables (v2 new!)

Bubble Tea sends `tea.EnvMsg` at startup — essential for SSH apps:

```go
case tea.EnvMsg:
    m.term = msg.Getenv("TERM")
```

### Color Profile Detection

```go
case tea.ColorProfileMsg:
    m.colorProfile = msg.Profile
```

### Keyboard Enhancements Detection

```go
case tea.KeyboardEnhancementsMsg:
    if msg.SupportsKeyDisambiguation() {
        // Terminal supports enhanced key input
    }
```

### Program Options

```go
p := tea.NewProgram(model,
    tea.WithWindowSize(80, 24),           // For testing
    tea.WithColorProfile(profile),        // Manual color profile
    tea.WithEnvironment(envs),            // SSH: pass client envs
    tea.WithInput(ttyIn),                 // Custom input
    tea.WithOutput(ttyOut),               // Custom output
)

// Open TTY manually (useful when stdin/stdout are piped)
ttyIn, ttyOut, err := tea.OpenTTY()
```

### Debugging

```go
if len(os.Getenv("DEBUG")) > 0 {
    f, err := tea.LogToFile("debug.log", "debug")
    if err != nil {
        fmt.Println("fatal:", err)
        os.Exit(1)
    }
    defer f.Close()
}
```

---

## Bubbles v2

All Bubbles updated for v2 with consistent patterns:

### Import Paths

```go
import "charm.land/bubbles/v2/spinner"
import "charm.land/bubbles/v2/textinput"
import "charm.land/bubbles/v2/textarea"
import "charm.land/bubbles/v2/list"
import "charm.land/bubbles/v2/table"
import "charm.land/bubbles/v2/viewport"
import "charm.land/bubbles/v2/progress"
import "charm.land/bubbles/v2/help"
import "charm.land/bubbles/v2/paginator"
import "charm.land/bubbles/v2/filepicker"
import "charm.land/bubbles/v2/timer"
import "charm.land/bubbles/v2/stopwatch"
import "charm.land/bubbles/v2/cursor"
import "charm.land/bubbles/v2/key"
```

### v2 Universal Changes

1. **Functional options** for constructors:
   ```go
   // v1: viewport.New(80, 24)
   vp := viewport.New(viewport.WithWidth(80), viewport.WithHeight(24))

   // v1: stopwatch.NewWithInterval(500ms)
   sw := stopwatch.New(stopwatch.WithInterval(500 * time.Millisecond))

   // v1: timer.NewWithInterval(30s, 100ms)
   t := timer.New(30*time.Second, timer.WithInterval(100*time.Millisecond))
   ```

2. **Getters/setters** instead of exported fields:
   ```go
   // v1: vp.Width = 40 / fmt.Println(vp.Width)
   vp.SetWidth(40)
   fmt.Println(vp.Width())
   ```

3. **DefaultKeyMap is a function**:
   ```go
   // v1: textinput.DefaultKeyMap (variable)
   km := textinput.DefaultKeyMap()   // function
   km := textarea.DefaultKeyMap()
   km := paginator.DefaultKeyMap()
   ```

4. **Light/dark styles**:
   ```go
   h := help.New()
   h.Styles = help.DefaultStyles(isDark)     // pass bool
   h.Styles = help.DefaultDarkStyles()       // or pick directly
   h.Styles = help.DefaultLightStyles()
   // Same pattern for: list, textarea, textinput
   ```

### Spinner

```go
import "charm.land/bubbles/v2/spinner"

type model struct {
    spinner spinner.Model
}

func initialModel() model {
    s := spinner.New()
    s.Spinner = spinner.Dot
    s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
    return model{spinner: s}
}

func (m model) Init() tea.Cmd {
    return m.spinner.Tick() // v2: use Model.Tick() not spinner.Tick()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    m.spinner, cmd = m.spinner.Update(msg)
    return m, cmd
}

func (m model) View() tea.View {
    return tea.NewView(m.spinner.View() + " Loading...")
}
```

### Text Input

```go
import "charm.land/bubbles/v2/textinput"

type model struct {
    input textinput.Model
}

func initialModel() model {
    ti := textinput.New()
    ti.Placeholder = "Type something..."
    ti.Focus()
    ti.SetWidth(40) // v2: setter method
    return model{input: ti}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyPressMsg:
        switch msg.String() {
        case "enter":
            // process m.input.Value()
        case "ctrl+c":
            return m, tea.Quit
        }
    }
    var cmd tea.Cmd
    m.input, cmd = m.input.Update(msg)
    return m, cmd
}

func (m model) View() tea.View {
    return tea.NewView(m.input.View())
}
```

**v2 style changes for textinput:**
```go
// v1: ti.PromptStyle, ti.TextStyle, ti.PlaceholderStyle, ti.CompletionStyle
// v2: grouped under Styles struct
styles := textinput.DefaultStyles(isDark)
ti.SetStyles(styles)
// Access: styles.Focused.Prompt, styles.Focused.Text, etc.
```

### Text Area

```go
import "charm.land/bubbles/v2/textarea"

ta := textarea.New()
ta.Placeholder = "Write something..."
ta.Focus()

// v2 style changes:
// v1: ta.FocusedStyle / ta.BlurredStyle
// v2: ta.Styles.Focused / ta.Styles.Blurred
styles := textarea.DefaultStyles(isDark)

// v2 new methods:
ta.Column()           // current column
ta.ScrollYOffset()    // scroll position
ta.MoveToBeginning()  // jump to start
ta.MoveToEnd()        // jump to end

// v2: SetCursor renamed
// v1: ta.SetCursor(col)
ta.SetCursorColumn(col) // v2
```

### List

```go
import "charm.land/bubbles/v2/list"

type item struct {
    title, desc string
}
func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

func initialModel() model {
    items := []list.Item{
        item{title: "Raspberry Pi", desc: "A tiny computer"},
        item{title: "Arduino", desc: "A microcontroller"},
    }
    // v2: DefaultStyles takes isDark bool
    l := list.New(items, list.NewDefaultDelegate(), 20, 14)
    l.Title = "My Items"
    return model{list: l}
}
```

### Table

```go
import "charm.land/bubbles/v2/table"

columns := []table.Column{
    {Title: "Name", Width: 20},
    {Title: "Age", Width: 5},
}
rows := []table.Row{
    {"Alice", "25"},
    {"Bob", "30"},
}
t := table.New(
    table.WithColumns(columns),
    table.WithRows(rows),
    table.WithFocused(true),
)
// v2: use getter/setter for width/height
t.SetWidth(40)
t.SetHeight(10)
```

### Viewport

```go
import "charm.land/bubbles/v2/viewport"

// v2: functional options
vp := viewport.New(viewport.WithWidth(80), viewport.WithHeight(24))
vp.SetContent("long content here...")

// v2 new: horizontal scrolling (arrow keys)
// v2 new: soft wrapping
vp.SoftWrap = true

// v2 new: line number gutter
vp.LeftGutterFunc = func(info viewport.GutterContext) string {
    if info.Soft {
        return "     | "
    }
    if info.Index >= info.TotalLines {
        return "   ~ | "
    }
    return fmt.Sprintf("%4d | ", info.Index+1)
}

// v2 new: regex highlighting
vp.SetHighlights(regexp.MustCompile("error").FindAllStringIndex(vp.GetContent(), -1))
vp.HighlightNext()
vp.HighlightPrevious()
vp.ClearHighlights()

// v2 new: per-line styling
vp.StyleLineFunc = func(line int, content string) string {
    // return styled content
}

// v2: HighPerformanceRendering removed (no longer needed)
```

### Progress

```go
import "charm.land/bubbles/v2/progress"

// v2: complete color API overhaul
p := progress.New(
    progress.WithColors(lipgloss.Color("#FF0000"), lipgloss.Color("#0000FF")), // gradient
    progress.WithScaled(true),           // scale blend to filled portion
    progress.WithDefaultBlend(),         // v2 replaces WithDefaultGradient()
)
p.SetWidth(40) // v2: setter

// Single color:
p := progress.New(progress.WithColors(lipgloss.Color("#FF7CCB")))

// Dynamic coloring:
p := progress.New(progress.WithColorFunc(func(total, current float64) color.Color {
    // return computed color
}))
```

### Help

```go
import "charm.land/bubbles/v2/help"
import "charm.land/bubbles/v2/key"

type keyMap struct {
    Up   key.Binding
    Down key.Binding
    Quit key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
    return []key.Binding{k.Up, k.Down, k.Quit}
}
func (k keyMap) FullHelp() [][]key.Binding {
    return [][]key.Binding{{k.Up, k.Down}, {k.Quit}}
}

var keys = keyMap{
    Up:   key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
    Down: key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
    Quit: key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

h := help.New()
h.SetWidth(80)                       // v2: setter
h.Styles = help.DefaultStyles(isDark) // v2: light/dark aware
```

### Key Bindings

```go
import "charm.land/bubbles/v2/key"

// Define bindings
binding := key.NewBinding(
    key.WithKeys("k", "up"),
    key.WithHelp("↑/k", "move up"),
)

// Match in Update
case tea.KeyPressMsg:
    switch {
    case key.Matches(msg, keys.Up):
        // handle up
    case key.Matches(msg, keys.Down):
        // handle down
    }
```

---

## Lip Gloss v2

### Import

```go
import "charm.land/lipgloss/v2"
```

### Styles

```go
var style = lipgloss.NewStyle().
    Bold(true).
    Italic(true).
    Foreground(lipgloss.Color("#FAFAFA")).
    Background(lipgloss.Color("#7D56F4")).
    PaddingTop(2).
    PaddingLeft(4).
    Width(22)

// IMPORTANT: Use lipgloss.Println for auto color downsampling (standalone)
// Inside Bubble Tea, downsampling is automatic
lipgloss.Println(style.Render("Hello, kitty"))
```

### Colors (v2 Change: colors are now image/color.Color)

```go
// True color
lipgloss.Color("#FF0000")

// ANSI 256
lipgloss.Color("86")

// ANSI 16
lipgloss.Color("5")

// Named constants
lipgloss.Red
lipgloss.Green
lipgloss.BrightCyan

// RGB
lipgloss.RGBColor{R: 255, G: 0, B: 0}
```

### Adaptive Colors (v2 Breaking Change)

`lipgloss.AdaptiveColor` is **removed**. Use `LightDark`:

```go
// With Bubble Tea (recommended):
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.BackgroundColorMsg:
        m.styles = newStyles(msg.IsDark())
    }
    return m, nil
}

func newStyles(isDark bool) styles {
    lightDark := lipgloss.LightDark(isDark)
    return styles{
        title: lipgloss.NewStyle().Foreground(lightDark(
            lipgloss.Color("#333333"),  // light bg
            lipgloss.Color("#FAFAFA"),  // dark bg
        )),
    }
}

// Standalone:
hasDarkBG, _ := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
lightDark := lipgloss.LightDark(hasDarkBG)
myColor := lightDark(lipgloss.Color("#333"), lipgloss.Color("#FFF"))

// Quick migration with compat package:
import "charm.land/lipgloss/v2/compat"
color := compat.AdaptiveColor{
    Light: lipgloss.Color("#f1f1f1"),
    Dark:  lipgloss.Color("#cccccc"),
}
```

### Color Utilities

```go
dark := lipgloss.Darken(c, 0.5)
light := lipgloss.Lighten(c, 0.35)
comp := lipgloss.Complementary(c)
alpha := lipgloss.Alpha(c, 0.2)

// 1D gradient
colors := lipgloss.Blend1D(10, color1, color2)

// 2D gradient with rotation
colors := lipgloss.Blend2D(80, 24, 45.0, color1, color2, color3)
```

### Borders

```go
var style = lipgloss.NewStyle().
    BorderStyle(lipgloss.RoundedBorder()).
    BorderForeground(lipgloss.Color("63")).
    BorderTop(true).
    BorderBottom(true).
    BorderLeft(true).
    BorderRight(true)

// Gradient borders (v2 new!)
s := lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    BorderForegroundBlend(lipgloss.Color("#FF0000"), lipgloss.Color("#0000FF"))

// Border types: NormalBorder, RoundedBorder, ThickBorder, DoubleBorder,
//               HiddenBorder, MarkdownBorder, ASCIIBorder
```

### Underline Styles (v2 new!)

```go
s := lipgloss.NewStyle().
    UnderlineStyle(lipgloss.UnderlineCurly).
    UnderlineColor(lipgloss.Color("#FF0000"))
// Styles: UnderlineNone, UnderlineSingle, UnderlineDouble,
//         UnderlineCurly, UnderlineDotted, UnderlineDashed
```

### Hyperlinks (v2 new!)

```go
s := lipgloss.NewStyle().
    Foreground(lipgloss.Color("#7B2FBE")).
    Hyperlink("https://charm.land")
lipgloss.Println(s.Render("Visit Charm"))
```

### Layout

```go
// Join horizontally/vertically
lipgloss.JoinHorizontal(lipgloss.Bottom, blockA, blockB)
lipgloss.JoinVertical(lipgloss.Center, blockA, blockB)

// Place in whitespace
lipgloss.Place(80, 24, lipgloss.Center, lipgloss.Center, content)
lipgloss.PlaceHorizontal(80, lipgloss.Center, content)
lipgloss.PlaceVertical(24, lipgloss.Bottom, content)

// Measure
w, h := lipgloss.Size(block)
width := lipgloss.Width(block)
height := lipgloss.Height(block)

// Wrap (preserves ANSI styles and hyperlinks)
wrapped := lipgloss.Wrap(styledText, 40, " ")
```

### Compositing (v2 new!)

```go
// Layer-based compositing
a := lipgloss.NewLayer(pickles).X(4).Y(2).Z(1)
b := lipgloss.NewLayer(bitterMelon).X(22).Y(1)
c := lipgloss.NewLayer(sriracha).X(11).Y(7)
output := compositor.Compose(a, b, c).Render()
```

### Tables (lipgloss sub-package)

```go
import "charm.land/lipgloss/v2/table"

t := table.New().
    Border(lipgloss.RoundedBorder()).
    BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("99"))).
    StyleFunc(func(row, col int) lipgloss.Style {
        switch {
        case row == table.HeaderRow:
            return headerStyle
        case row%2 == 0:
            return evenRowStyle
        default:
            return oddRowStyle
        }
    }).
    Headers("LANGUAGE", "FORMAL", "INFORMAL").
    Rows(rows...)

lipgloss.Println(t)
```

### Lists and Trees (lipgloss sub-packages)

```go
import "charm.land/lipgloss/v2/list"
import "charm.land/lipgloss/v2/tree"

// List
l := list.New("A", "B", "C").
    Enumerator(list.Roman).
    EnumeratorStyle(enumStyle).
    ItemStyle(itemStyle)

// Tree
t := tree.Root(".").
    Child("macOS", "Linux", "BSD").
    Enumerator(tree.RoundedEnumerator)

lipgloss.Println(l)
lipgloss.Println(t)
```

### Printing (v2 Change: use lipgloss writers for standalone)

```go
// Inside Bubble Tea: downsampling is automatic, just use View()
// Standalone: use lipgloss print functions (drop-in fmt replacements)
lipgloss.Print(s)
lipgloss.Println(s)
lipgloss.Printf("hello %s", styled)
lipgloss.Fprint(os.Stderr, s)
lipgloss.Sprint(s)  // render to variable
```

---

## Huh v2 (Forms)

### Import

```go
import "charm.land/huh/v2"
```

### Standalone Form

```go
var name string
var confirmed bool

form := huh.NewForm(
    huh.NewGroup(
        huh.NewInput().
            Title("What's your name?").
            Value(&name).
            Validate(func(s string) error {
                if s == "" {
                    return errors.New("name required")
                }
                return nil
            }),

        huh.NewConfirm().
            Title("Continue?").
            Value(&confirmed),
    ),
)

err := form.Run()
```

### Field Types

```go
// Input (single line)
huh.NewInput().Title("Name").Placeholder("Enter name...").Value(&name)

// Text (multi line)
huh.NewText().Title("Description").CharLimit(400).Value(&desc)

// Select
huh.NewSelect[string]().
    Title("Pick one").
    Options(
        huh.NewOption("Option A", "a"),
        huh.NewOption("Option B", "b"),
    ).
    Value(&choice)

// MultiSelect
huh.NewMultiSelect[string]().
    Title("Pick many").
    Options(huh.NewOptions("A", "B", "C")...).
    Limit(2).
    Value(&choices)

// Confirm
huh.NewConfirm().Title("Sure?").Affirmative("Yes!").Negative("No.").Value(&ok)
```

### Single Field Shorthand

```go
var name string
huh.NewInput().Title("Name?").Value(&name).Run()
```

### Dynamic Forms

```go
huh.NewSelect[string]().
    Value(&state).
    TitleFunc(func() string {
        if country == "US" { return "State" }
        return "Province"
    }, &country).
    OptionsFunc(func() []huh.Option[string] {
        return huh.NewOptions(fetchStates(country)...)
    }, &country)
```

### Huh in Bubble Tea

```go
type model struct {
    form *huh.Form
}

func newModel() model {
    return model{
        form: huh.NewForm(
            huh.NewGroup(
                huh.NewSelect[string]().
                    Key("class").
                    Options(huh.NewOptions("Warrior", "Mage", "Rogue")...).
                    Title("Choose your class"),
            ),
        ),
    }
}

func (m model) Init() tea.Cmd {
    return m.form.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    form, cmd := m.form.Update(msg)
    if f, ok := form.(*huh.Form); ok {
        m.form = f
    }
    return m, cmd
}

func (m model) View() tea.View {
    if m.form.State == huh.StateCompleted {
        class := m.form.GetString("class")
        return tea.NewView(fmt.Sprintf("You chose: %s", class))
    }
    return tea.NewView(m.form.View())
}
```

### Themes & Accessibility

```go
form.WithTheme(huh.ThemeDracula())  // Charm, Dracula, Catppuccin, Base16, Default
form.WithAccessible(os.Getenv("ACCESSIBLE") != "")
```

### Spinner

```go
import "charm.land/huh/v2/spinner"

err := spinner.New().
    Title("Processing...").
    Action(doWork).
    Run()
```

---

## CLY Module Integration

### Simple TUI Module

```go
package mymodule

import (
    "fmt"
    "github.com/spf13/cobra"
    tea "charm.land/bubbletea/v2"
    "charm.land/lipgloss/v2"
)

func Register(parent *cobra.Command) {
    cmd := &cobra.Command{
        Use:   "mymodule",
        Short: "My TUI module",
        RunE:  run,
    }
    parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
    p := tea.NewProgram(initialModel())
    _, err := p.Run()
    return err
}

type model struct {
    // state
}

func initialModel() model {
    return model{}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyPressMsg:
        switch msg.String() {
        case "q", "ctrl+c":
            return m, tea.Quit
        }
    }
    return m, nil
}

func (m model) View() tea.View {
    return tea.NewView("Hello from my module!")
}
```

### Full-screen TUI Module

```go
func (m model) View() tea.View {
    v := tea.NewView(m.renderContent())
    v.AltScreen = true
    return v
}
```

### Using Shared Styles

```go
import "github.com/yurifrl/cly/pkg/style"

fmt.Println(style.TitleStyle.Render("Title"))
fmt.Println(style.GreenStyle.Render("Success"))
```

---

## v1 to v2 Migration Quick Reference

| v1 | v2 |
|---|---|
| `import tea "github.com/charmbracelet/bubbletea"` | `import tea "charm.land/bubbletea/v2"` |
| `import "github.com/charmbracelet/bubbles/..."` | `import "charm.land/bubbles/v2/..."` |
| `import "github.com/charmbracelet/lipgloss"` | `import "charm.land/lipgloss/v2"` |
| `import "github.com/charmbracelet/huh"` | `import "charm.land/huh/v2"` |
| `View() string` | `View() tea.View` |
| `return "content"` | `return tea.NewView("content")` |
| `tea.KeyMsg` | `tea.KeyPressMsg` / `tea.KeyReleaseMsg` |
| `key.Type`, `key.Runes` | `key.Code`, `key.Text` |
| `msg.Alt` (bool) | `key.Mod` |
| `tea.MouseMsg` | `tea.MouseClickMsg` / `tea.MouseWheelMsg` / etc. |
| `tea.EnterAltScreen` (Cmd) | `v.AltScreen = true` (View) |
| `tea.EnableMouseCellMotion` (Cmd) | `v.MouseMode = tea.MouseModeCellMotion` (View) |
| `lipgloss.AdaptiveColor{Light, Dark}` | `lipgloss.LightDark(isDark)(light, dark)` |
| `fmt.Println(styled)` | `lipgloss.Println(styled)` |
| `viewport.New(w, h)` | `viewport.New(viewport.WithWidth(w), viewport.WithHeight(h))` |
| `vp.Width = 40` | `vp.SetWidth(40)` |
| `vp.Width` (field) | `vp.Width()` (method) |
| `textinput.DefaultKeyMap` (var) | `textinput.DefaultKeyMap()` (func) |
| `spinner.Tick()` (pkg func) | `m.spinner.Tick()` (method) |
| `viewport.HighPerformanceRendering` | Removed (not needed) |
| `ti.PromptStyle` | `styles.Focused.Prompt` |
| `ta.FocusedStyle` | `ta.Styles.Focused` |
| `progress.WithGradient(a, b)` | `progress.WithColors(a, b)` |
| `progress.WithDefaultGradient()` | `progress.WithDefaultBlend()` |
| `space` key returns `" "` | `space` key returns `"space"` |
