package helpy

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRegistry_Order(t *testing.T) {
	Reset()
	defer Reset()

	Register(Entry{Section: "B", Flags: []string{"--b1"}, Order: 1})
	Register(Entry{Section: "A", Flags: []string{"--a1"}, Order: 2})
	Register(Entry{Section: "B", Flags: []string{"--b2"}, Order: 0})
	Register(Entry{Section: "A", Flags: []string{"--a2"}, Order: 1})

	got := All()
	want := []string{"--b2", "--b1", "--a2", "--a1"}
	for i, e := range got {
		if e.Flags[0] != want[i] {
			t.Errorf("order[%d] = %q, want %q", i, e.Flags[0], want[i])
		}
	}
}

func TestRenderText(t *testing.T) {
	Reset()
	defer Reset()

	Register(Entry{
		Section:     "Naming",
		Flags:       []string{"-n", "--name"},
		Value:       "<name>",
		Description: "Set session name.",
		EnvVars:     []string{"CLY_SESSION_NAME"},
		Examples:    []string{"cly pi -n foo"},
	})
	Register(Entry{
		Section:     "Naming",
		Flags:       []string{"--sety"},
		Description: "Override piwrap config.",
	})

	var buf bytes.Buffer
	RenderText(&buf, "cly pi — header", "trailer")
	out := buf.String()

	for _, want := range []string{
		"cly pi — header",
		"NAMING",
		"-n, --name <name>",
		"Set session name.",
		"CLY_SESSION_NAME",
		"cly pi -n foo",
		"--sety",
		"trailer",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n--- output ---\n%s", want, out)
		}
	}
}

func TestRenderJSON(t *testing.T) {
	Reset()
	defer Reset()

	Register(Entry{
		Section:     "Naming",
		Flags:       []string{"-n"},
		Description: "name",
	})
	Register(Entry{
		Section:     "Import",
		Flags:       []string{"--sety"},
		Description: "import",
	})

	var buf bytes.Buffer
	if err := RenderJSON(&buf, "0.1.0", map[string]string{"a": "b"}); err != nil {
		t.Fatal(err)
	}

	var got JSONOutput
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Version != "0.1.0" {
		t.Errorf("version = %q", got.Version)
	}
	if len(got.Sections) != 2 {
		t.Fatalf("sections = %d, want 2", len(got.Sections))
	}
	if got.Sections[0].Title != "Naming" || got.Sections[1].Title != "Import" {
		t.Errorf("section titles = %v", []string{got.Sections[0].Title, got.Sections[1].Title})
	}
}
