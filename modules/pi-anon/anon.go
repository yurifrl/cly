package pianon

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

)

// AnonSession represents a .anon.json file
type AnonSession struct {
	SessionID  string   `json:"session_id"`
	Timestamp  string   `json:"timestamp"`
	Words      string   `json:"words"`
	CalledFrom []string `json:"called_from"`
	CreatedAt  string   `json:"created_at"`
	Toggles    Toggles  `json:"toggles"`
	Dir        string   `json:"-"` // not persisted
}

type Toggles struct {
	Skills          bool `json:"skills"`
	Extensions      bool `json:"extensions"`
	PromptTemplates bool `json:"prompt_templates"`
}

var wordList = []string{
	"ace", "add", "age", "ago", "aid", "aim", "air", "all", "amp", "ant",
	"ape", "apt", "arc", "ark", "arm", "art", "ash", "ask", "ate", "awe",
	"axe", "bad", "bag", "ban", "bar", "bat", "bay", "bed", "bet", "big",
	"bin", "bit", "bog", "bow", "box", "boy", "bud", "bug", "bun", "bus",
	"but", "buy", "cab", "cam", "can", "cap", "car", "cat", "cow", "cub",
	"cup", "cut", "dab", "dam", "day", "den", "dew", "did", "dig", "dim",
	"dip", "dog", "dot", "dry", "dub", "dud", "due", "dug", "dun", "duo",
	"dye", "ear", "eat", "eel", "egg", "ego", "elk", "elm", "emu", "end",
	"era", "eve", "ewe", "eye", "fab", "fan", "far", "fat", "fax", "fed",
	"few", "fig", "fin", "fit", "fix", "fly", "fog", "for", "fox", "fry",
	"fun", "fur", "gag", "gal", "gap", "gas", "gem", "get", "gig", "gin",
	"gnu", "god", "got", "gum", "gun", "gut", "guy", "gym", "had", "ham",
	"has", "hat", "hay", "hen", "her", "hew", "hex", "hid", "him", "hip",
	"hit", "hog", "hop", "hot", "how", "hub", "hue", "hug", "hum", "hut",
	"ice", "icy", "ill", "imp", "ink", "inn", "ion", "ire", "irk", "ivy",
	"jab", "jag", "jam", "jar", "jaw", "jay", "jet", "jig", "job", "jog",
	"jot", "joy", "jug", "jut", "keg", "ken", "key", "kid", "kin", "kit",
	"lab", "lad", "lag", "lap", "law", "lay", "led", "leg", "let", "lid",
	"lip", "lit", "log", "lot", "low", "lug", "mad", "map", "mar", "mat",
	"maw", "max", "may", "men", "met", "mid", "mix", "mob", "mod", "mop",
	"mow", "mud", "mug", "mum", "nab", "nag", "nap", "net", "new", "nib",
	"nil", "nip", "nit", "nod", "nor", "not", "now", "nun", "nut", "oak",
	"oar", "oat", "odd", "ode", "off", "oft", "oil", "old", "one", "opt",
	"orb", "ore", "our", "out", "ova", "owe", "owl", "own", "pad", "pal",
	"pan", "par", "pat", "paw", "pay", "pea", "peg", "pen", "pep", "per",
	"pet", "pew", "pie", "pig", "pin", "pit", "ply", "pod", "pop", "pot",
	"pow", "pox", "pro", "pry", "pub", "pug", "pun", "pup", "pus", "put",
	"rag", "ram", "ran", "rap", "rat", "raw", "ray", "red", "ref", "rib",
	"rid", "rig", "rim", "rip", "rob", "rod", "rot", "row", "rub", "rug",
	"rum", "run", "rut", "rye", "sad", "sag", "sap", "sat", "saw", "say",
	"sea", "set", "sew", "shy", "sin", "sip", "sir", "sit", "six", "ski",
	"sky", "sly", "sob", "sod", "son", "sop", "sot", "sow", "soy", "spa",
	"spy", "sty", "sub", "sue", "sum", "sun", "sup", "tab", "tad", "tag",
	"tan", "tap", "tar", "tax", "tea", "ten", "the", "tie", "tin", "tip",
	"toe", "ton", "too", "top", "tot", "tow", "toy", "try", "tub", "tug",
	"two", "urn", "use", "van", "vat", "vet", "vex", "via", "vie", "vim",
	"vow", "wad", "wag", "war", "was", "wax", "way", "web", "wed", "wet",
	"who", "wig", "win", "wit", "woe", "wok", "won", "woo", "wow", "yak",
	"yam", "yap", "yaw", "yea", "yes", "yet", "yew", "you", "zap", "zed",
	"zen", "zip", "zoo",
}

func generateWords() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("%s-%s-%s",
		wordList[r.Intn(len(wordList))],
		wordList[r.Intn(len(wordList))],
		wordList[r.Intn(len(wordList))],
	)
}

func findAllSessions() ([]AnonSession, error) {
	matches, err := filepath.Glob("/tmp/*-pi-anon")
	if err != nil {
		return nil, err
	}

	var sessions []AnonSession
	for _, d := range matches {
		s, err := loadSession(d)
		if err != nil {
			continue
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func loadSession(dir string) (AnonSession, error) {
	data, err := os.ReadFile(filepath.Join(dir, ".anon.json"))
	if err != nil {
		return AnonSession{}, err
	}
	var s AnonSession
	if err := json.Unmarshal(data, &s); err != nil {
		return AnonSession{}, err
	}
	s.Dir = dir
	return s, nil
}

func saveSession(s AnonSession) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.Dir, ".anon.json"), data, 0644)
}

func findSession(query string) (AnonSession, error) {
	sessions, err := findAllSessions()
	if err != nil {
		return AnonSession{}, err
	}
	for _, s := range sessions {
		if s.Words == query || s.Timestamp == query {
			return s, nil
		}
	}
	return AnonSession{}, fmt.Errorf("no session found for: %s", query)
}



func addCalledFrom(s *AnonSession) {
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	for _, d := range s.CalledFrom {
		if d == cwd {
			return
		}
	}
	s.CalledFrom = append(s.CalledFrom, cwd)
}

func buildPiArgs(t Toggles) []string {
	var args []string
	if !t.Skills {
		args = append(args, "--no-skills")
	}
	if !t.Extensions {
		args = append(args, "--no-extensions")
	}
	if !t.PromptTemplates {
		args = append(args, "--no-prompt-templates")
	}
	return args
}

func launchPi(dir string, toggles Toggles) error {
	args := buildPiArgs(toggles)

	piPath, err := exec.LookPath("pi")
	if err != nil {
		return fmt.Errorf("pi not found in PATH")
	}

	cmd := exec.Command(piPath, args...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func newSession() error {
	ts := fmt.Sprintf("%d", time.Now().Unix())
	words := generateWords()
	dirName := fmt.Sprintf("%s-%s-pi-anon", ts, words)
	sessionDir := filepath.Join("/tmp", dirName)

	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return err
	}

	s := AnonSession{
		SessionID: dirName,
		Timestamp: ts,
		Words:     words,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Dir:       sessionDir,
		Toggles: Toggles{
			Skills:          true,
			Extensions:      true,
			PromptTemplates: true,
		},
	}
	addCalledFrom(&s)

	fmt.Println("🆕 New anonymous session")
	fmt.Printf("   Words: %s\n", words)
	fmt.Printf("   Dir:   %s\n", sessionDir)
	fmt.Println()

	// Interactive toggle picker
	toggles, cancelled := promptToggles()
	if cancelled {
		// Clean up the created dir
		os.RemoveAll(sessionDir)
		fmt.Println("Cancelled.")
		return nil
	}
	s.Toggles = toggles

	if err := saveSession(s); err != nil {
		return err
	}

	fmt.Println()

	disabled := []string{}
	if !s.Toggles.Skills {
		disabled = append(disabled, "skills")
	}
	if !s.Toggles.Extensions {
		disabled = append(disabled, "extensions")
	}
	if !s.Toggles.PromptTemplates {
		disabled = append(disabled, "prompt-templates")
	}
	if len(disabled) > 0 {
		fmt.Printf("   Disabled: %s\n", strings.Join(disabled, ", "))
	}
	fmt.Printf("\n   Resume later: cly pi-anon %s\n\n", words)

	return launchPi(sessionDir, s.Toggles)
}

func resumeSession(query string) error {
	s, err := findSession(query)
	if err != nil {
		return err
	}

	fmt.Printf("🔄 Resuming session: %s\n", s.Words)
	fmt.Printf("   Dir: %s\n", s.Dir)

	addCalledFrom(&s)
	if err := saveSession(s); err != nil {
		return err
	}

	fmt.Println()
	return launchPi(s.Dir, s.Toggles)
}
