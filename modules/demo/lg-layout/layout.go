package lg_layout

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// Hardcoded gradient colors (pink to deep red).
var gradientColors = []string{
	"#F25D94", "#EE5490", "#EA4B8B", "#E64287", "#E23982",
	"#DE307E", "#DA2779", "#D61E75", "#D21570", "#CE0C6C",
	"#CA0367", "#C40060", "#BA0058", "#B00050", "#A60048",
	"#9C0040", "#920038", "#880030", "#7E0028", "#740020",
}

// Color definitions.
var (
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
)

// Tabs.
var (
	activeTabBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      " ",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "┘",
		BottomRight: "└",
	}

	tabBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "┴",
		BottomRight: "┴",
	}

	tabStyle = lipgloss.NewStyle().
			Border(tabBorder, true).
			BorderForeground(highlight).
			Padding(0, 1)

	activeTabStyle = tabStyle.
			Border(activeTabBorder, true)

	tabGapStyle = lipgloss.NewStyle().
			Border(lipgloss.Border{Bottom: "─"}, false, false, true, false).
			BorderForeground(highlight)
)

// Title gradient.
func titleGradient(title string) string {
	runes := []rune(title)
	var b strings.Builder
	for i, r := range runes {
		colorIdx := i % len(gradientColors)
		s := lipgloss.NewStyle().Foreground(lipgloss.Color(gradientColors[colorIdx]))
		b.WriteString(s.Render(string(r)))
	}
	return b.String()
}

// Description box.
var (
	descStyle = lipgloss.NewStyle().MarginTop(1)

	infoStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(subtle)
)

// Dialog box.
var (
	dialogBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#874BFD")).
			Padding(1, 0).
			BorderTop(true).
			BorderLeft(true).
			BorderRight(true).
			BorderBottom(true)

	buttonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF7DB")).
			Background(lipgloss.Color("#888B7E")).
			Padding(0, 3).
			MarginTop(1)

	activeButtonStyle = buttonStyle.
				Foreground(lipgloss.Color("#FFF7DB")).
				Background(lipgloss.Color("#F25D94")).
				MarginRight(2).
				Underline(true)
)

// List styles.
var (
	listStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(subtle).
			MarginRight(2).
			Height(8).
			Width(22)

	listHeader = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(subtle).
			MarginRight(2).
			Render

	listItem = lipgloss.NewStyle().PaddingLeft(2).Render

	checkMark = lipgloss.NewStyle().SetString("✓").
			Foreground(special).
			PaddingRight(1).
			String()

	listDone = func(s string) string {
		return checkMark + lipgloss.NewStyle().
			Strikethrough(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#969B86", Dark: "#696969"}).
			Render(s)
	}
)

// History.
var (
	historyStyle = lipgloss.NewStyle().
			Align(lipgloss.Left).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Margin(1, 3, 0, 0).
			Padding(1, 2).
			Height(19).
			Width(26)

	historyHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(lipgloss.Color("#6124DF")).
				Bold(true).
				Padding(0, 1).
				Width(22).
				Align(lipgloss.Center)
)

// Status bar.
var (
	statusNugget = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#343433", Dark: "#C1C6B2"}).
			Background(lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#353533"})

	statusStyle = lipgloss.NewStyle().
			Inherit(statusBarStyle).
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#FF5F87")).
			Padding(0, 1).
			MarginRight(1)

	encodingStyle = statusNugget.
			Background(lipgloss.Color("#A550DF")).
			Align(lipgloss.Right)

	statusText = lipgloss.NewStyle().Inherit(statusBarStyle)

	fishCakeStyle = statusNugget.Background(lipgloss.Color("#6124DF"))
)

// renderLayout creates the full layout showcase.
func renderLayout() string {
	physicalWidth, _, _ := term.GetSize(int(os.Stdout.Fd()))
	if physicalWidth == 0 {
		physicalWidth = 80
	}

	doc := strings.Builder{}

	// ── Tabs ────────────────────────────────────────────────────────────
	{
		row := lipgloss.JoinHorizontal(
			lipgloss.Top,
			activeTabStyle.Render("Lip Gloss"),
			tabStyle.Render("Blush"),
			tabStyle.Render("Eye Shadow"),
			tabStyle.Render("Mascara"),
			tabStyle.Render("Foundation"),
		)
		gap := tabGapStyle.Render(strings.Repeat(" ", max(0, physicalWidth-lipgloss.Width(row)-2)))
		row = lipgloss.JoinHorizontal(lipgloss.Bottom, row, gap)
		doc.WriteString(row + "\n\n")
	}

	// ── Title with gradient ─────────────────────────────────────────────
	{
		desc := lipgloss.JoinVertical(lipgloss.Left,
			descStyle.Render("Style Definitions for Nice Terminal Layouts"),
			infoStyle.Render("From Charm"+strings.Repeat(" ", 7)+"https://github.com/charmbracelet/lipgloss"),
		)

		row := lipgloss.JoinHorizontal(lipgloss.Top, titleGradient("Lip Gloss"), "  ", desc)
		doc.WriteString(row + "\n\n")
	}

	// ── Dialog ──────────────────────────────────────────────────────────
	{
		okButton := activeButtonStyle.Render("Yes")
		cancelButton := buttonStyle.Render("Maybe")

		question := lipgloss.NewStyle().Width(50).Align(lipgloss.Center).Render(
			"Are you sure you want to eat marmalade?")
		buttons := lipgloss.JoinHorizontal(lipgloss.Top, okButton, cancelButton)
		ui := lipgloss.JoinVertical(lipgloss.Center, question, buttons)

		dialog := lipgloss.Place(physicalWidth, 9,
			lipgloss.Center, lipgloss.Center,
			dialogBoxStyle.Render(ui),
		)

		doc.WriteString(dialog + "\n\n")
	}

	// ── 3-Column Lists ──────────────────────────────────────────────────
	{
		list1 := listStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				listHeader("Citrus Fruits"),
				listItem("Grapefruit"),
				listItem("Yuzu"),
				listDone("Lemon"),
				listItem("Tangerine"),
				listDone("Orange"),
				listItem("Kumquat"),
			),
		)

		list2 := listStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				listHeader("Actual Lip Gloss Brands"),
				listItem("Glossier"),
				listDone("Claire's Boutique"),
				listItem("Nyx"),
				listItem("Mac"),
				listDone("Milk"),
			),
		)

		list3 := listStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				listHeader("Programming Languages"),
				listItem("Go"),
				listItem("Rust"),
				listDone("Python"),
				listItem("TypeScript"),
				listItem("Zig"),
			),
		)

		lists := lipgloss.JoinHorizontal(lipgloss.Top, list1, list2, list3)
		doc.WriteString(lipgloss.PlaceHorizontal(physicalWidth, lipgloss.Center, lists) + "\n\n")
	}

	// ── History ─────────────────────────────────────────────────────────
	{
		history := historyStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				historyHeaderStyle.Render("History"),
				"",
				"Lip Gloss v1.0 — Initial release",
				"",
				"Lip Gloss v0.5 — Beta",
				"  • Added gradient support",
				"  • Tab support",
				"  • Layout system",
				"",
				"Lip Gloss v0.1 — Alpha",
				"  • Basic styling",
				"  • Color support",
			),
		)
		doc.WriteString(lipgloss.PlaceHorizontal(physicalWidth, lipgloss.Right, history) + "\n")
	}

	// ── Status Bar ──────────────────────────────────────────────────────
	{
		w := lipgloss.Width

		statusKey := statusStyle.Render("STATUS")
		encoding := encodingStyle.Render("UTF-8")
		fishCake := fishCakeStyle.Render("🍥 Fish Cake")
		statusVal := statusText.
			Width(physicalWidth - w(statusKey) - w(encoding) - w(fishCake)).
			Render("Ravishing")

		bar := lipgloss.JoinHorizontal(lipgloss.Top,
			statusKey,
			statusVal,
			encoding,
			fishCake,
		)

		doc.WriteString(statusBarStyle.Width(physicalWidth).Render(bar))
	}

	return doc.String()
}
