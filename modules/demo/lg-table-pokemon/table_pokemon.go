package lg_table_pokemon

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

// TypeColors maps Pokémon types to ANSI 256-color codes for rich terminal display.
var TypeColors = map[string]string{
	"Bug":      "106",
	"Dark":     "95",
	"Dragon":   "63",
	"Electric": "226",
	"Fairy":    "213",
	"Fighting": "167",
	"Fire":     "202",
	"Flying":   "117",
	"Ghost":    "99",
	"Grass":    "34",
	"Ground":   "178",
	"Ice":      "45",
	"Normal":   "252",
	"Poison":   "129",
	"Psychic":  "200",
	"Rock":     "137",
	"Steel":    "249",
	"Water":    "33",
}

type pokemon struct {
	number   string
	name     string
	type1    string
	type2    string
	japanese string
	romaji   string
}

// selectedRow is the 0-based data row index to highlight (cursor-like selection).
const selectedRow = 4

// pokedex contains 28 Pokémon entries covering all 18 types.
var pokedex = []pokemon{
	{"001", "Bulbasaur", "Grass", "Poison", "フシギダネ", "Fushigidane"},
	{"002", "Ivysaur", "Grass", "Poison", "フシギソウ", "Fushigisō"},
	{"003", "Venusaur", "Grass", "Poison", "フシギバナ", "Fushigibana"},
	{"004", "Charmander", "Fire", "", "ヒトカゲ", "Hitokage"},
	{"005", "Charmeleon", "Fire", "", "リザード", "Rizādo"},
	{"006", "Charizard", "Fire", "Flying", "リザードン", "Rizādon"},
	{"007", "Squirtle", "Water", "", "ゼニガメ", "Zenigame"},
	{"008", "Wartortle", "Water", "", "カメール", "Kamēru"},
	{"009", "Blastoise", "Water", "", "カメックス", "Kamekkusu"},
	{"025", "Pikachu", "Electric", "", "ピカチュウ", "Pikachū"},
	{"026", "Raichu", "Electric", "", "ライチュウ", "Raichū"},
	{"035", "Clefairy", "Fairy", "", "ピッピ", "Pippi"},
	{"065", "Alakazam", "Psychic", "", "フーディン", "Fūdin"},
	{"068", "Machamp", "Fighting", "", "カイリキー", "Kairikī"},
	{"094", "Gengar", "Ghost", "Poison", "ゲンガー", "Gengā"},
	{"095", "Onix", "Rock", "Ground", "イワーク", "Iwāku"},
	{"106", "Hitmonlee", "Fighting", "", "サワムラー", "Sawamurā"},
	{"123", "Scyther", "Bug", "Flying", "ストライク", "Sutoraiku"},
	{"131", "Lapras", "Water", "Ice", "ラプラス", "Rapurasu"},
	{"133", "Eevee", "Normal", "", "イーブイ", "Ībui"},
	{"143", "Snorlax", "Normal", "", "カビゴン", "Kabigon"},
	{"149", "Dragonite", "Dragon", "Flying", "カイリュー", "Kairyū"},
	{"151", "Mew", "Psychic", "", "ミュウ", "Myū"},
	{"197", "Umbreon", "Dark", "", "ブラッキー", "Burakkī"},
	{"208", "Steelix", "Steel", "Ground", "ハガネール", "Haganēru"},
	{"212", "Scizor", "Bug", "Steel", "ハッサム", "Hassamu"},
	{"282", "Gardevoir", "Psychic", "Fairy", "サーナイト", "Sānaito"},
	{"330", "Flygon", "Ground", "Dragon", "フライゴン", "Furaigon"},
}

// renderTable builds and returns a colorful Pokédex table with type-colored cells.
func renderTable() string {
	rows := make([][]string, len(pokedex))
	for i, p := range pokedex {
		num := "#" + p.number
		t2 := p.type2
		if t2 == "" {
			t2 = "—"
		}
		rows[i] = []string{num, p.name, p.type1, t2, p.japanese, p.romaji}
	}

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229")).
		Align(lipgloss.Center)

	selectedBg := lipgloss.Color("236")

	t := table.New().
		Headers("#", "NAME", "TYPE 1", "TYPE 2", "JAPANESE", "OFFICIAL ROM").
		Rows(rows...).
		Border(lipgloss.DoubleBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
		Width(80).
		StyleFunc(func(row, col int) lipgloss.Style {
			s := lipgloss.NewStyle().Padding(0, 1)

			if row == table.HeaderRow {
				return headerStyle.Padding(0, 1)
			}

			p := pokedex[row]
			type1Color := lipgloss.Color(TypeColors[p.type1])
			var type2Color lipgloss.Color
			hasType2 := p.type2 != ""
			if hasType2 {
				type2Color = lipgloss.Color(TypeColors[p.type2])
			}

			isSelected := row == selectedRow

			switch col {
			case 0: // Pokédex number
				s = s.Foreground(lipgloss.Color("243")).Align(lipgloss.Right)
			case 1: // Name — bold, colored by primary type
				s = s.Foreground(type1Color).Bold(true)
			case 2: // Type 1 — colored by type
				s = s.Foreground(type1Color)
			case 3: // Type 2 — colored by secondary type
				if hasType2 {
					s = s.Foreground(type2Color)
				} else {
					s = s.Foreground(lipgloss.Color("240")).Faint(true)
				}
			case 4: // Japanese name
				s = s.Foreground(lipgloss.Color("252"))
			case 5: // Official romanization
				s = s.Foreground(lipgloss.Color("245")).Italic(true)
			}

			if isSelected {
				// Selected row: bright foreground on dark background
				s = s.Background(selectedBg).Bold(true)
				switch col {
				case 1, 2:
					s = s.Foreground(type1Color)
				case 3:
					if hasType2 {
						s = s.Foreground(type2Color)
					} else {
						s = s.Foreground(lipgloss.Color("243"))
					}
				default:
					s = s.Foreground(lipgloss.Color("255"))
				}
			} else if row%2 != 0 {
				// Alternating row tint for readability
				s = s.Faint(true)
			}

			return s
		})

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196")).
		MarginBottom(1)

	return titleStyle.Render(fmt.Sprintf("⚡ Pokédex (%d entries)", len(pokedex))) + "\n" + t.Render()
}
