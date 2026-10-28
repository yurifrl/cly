package envs

import (
	"bytes"
	"strings"
	"testing"
)

func TestFishLiteral(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"simple", "'simple'"},
		{"has 'quote'", `'has \'quote\''`},
		{"", "''"},
	}
	for _, tc := range tests {
		if got := fishLiteral(tc.in); got != tc.want {
			t.Errorf("fishLiteral(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestEnvQuote(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"plain-value", "plain-value"},
		{"has space", `'has space'`},
		{"it's", `'it'\''s'`},
		{"dollar$sign", `'dollar$sign'`},
		{"a=b", "a=b"},
	}
	for _, tc := range tests {
		if got := envQuote(tc.in); got != tc.want {
			t.Errorf("envQuote(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestValidEnvLabel(t *testing.T) {
	valid := []string{"API_KEY", "x", "A1_b", "HOME_DIR_2"}
	invalid := []string{"", "username", "password", "notesPlain", "server", "port", "database",
		"1starts-digit", "has-dash", "has space", "UPPER lower"}
	for _, l := range valid {
		if !validEnvLabel(l) {
			t.Errorf("validEnvLabel(%q) = false, want true", l)
		}
	}
	for _, l := range invalid {
		if validEnvLabel(l) {
			t.Errorf("validEnvLabel(%q) = true, want false", l)
		}
	}
}

func TestFilterValidFields(t *testing.T) {
	fields := []Field{
		{Label: "API_KEY", Value: "v1", Section: "work"},
		{Label: "password", Value: "secret"},
		{Label: "notesPlain", Value: "junk"},
		{Label: "", Value: "empty"},
		{Label: "VALID", Value: "v2", Section: "personal"},
	}
	got := filterValidFields(fields)
	if len(got) != 2 {
		t.Fatalf("filterValidFields kept %d fields, want 2: %+v", len(got), got)
	}
	if got[0].Label != "API_KEY" || got[1].Label != "VALID" {
		t.Errorf("unexpected survivors: %+v", got)
	}
}

func TestDecodeFieldsSectionLowercase(t *testing.T) {
	parsed := item{
		Fields: []struct {
			Label   string `json:"label"`
			Value   string `json:"value"`
			Type    string `json:"type"`
			Section *struct {
				Label string `json:"label"`
			} `json:"section"`
		}{
			{Label: "KEY", Value: "v", Section: &struct {
				Label string `json:"label"`
			}{Label: "Work"}},
			{Label: "OTHER", Value: "v"},
		},
	}
	got := decodeFields(parsed)
	if len(got) != 2 {
		t.Fatalf("decodeFields len = %d, want 2", len(got))
	}
	if got[0].Section != "work" {
		t.Errorf("section = %q, want %q", got[0].Section, "work")
	}
	if got[1].Section != "" {
		t.Errorf("missing section should be empty, got %q", got[1].Section)
	}
}

func TestBlobRoundTrip(t *testing.T) {
	fields := []Field{
		{Label: "A", Value: "one", Section: "general"},
		{Label: "B", Value: "with\ttab and more", Section: "work"},
		{Label: "C", Value: "'quoted'", Section: ""},
	}

	var lines []string
	for _, f := range fields {
		lines = append(lines, f.Label+"\t"+f.Section+"\t"+f.Value)
	}
	encoded := base64Encode(strings.Join(lines, "\n"))
	raw, err := base64Decode(encoded)
	if err != nil {
		t.Fatalf("base64Decode: %v", err)
	}

	var got []Field
	for _, line := range strings.Split(raw, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			continue
		}
		got = append(got, Field{Label: parts[0], Section: parts[1], Value: parts[2]})
	}
	if len(got) != len(fields) {
		t.Fatalf("round trip got %d fields, want %d", len(got), len(fields))
	}
	for i := range fields {
		if got[i] != fields[i] {
			t.Errorf("field %d = %+v, want %+v", i, got[i], fields[i])
		}
	}
}

func TestWriteOutput(t *testing.T) {
	fields := []Field{
		{Label: "TOKEN", Value: "abc", Section: "work"},
		{Label: "TOKEN", Value: "def", Section: ""},
	}

	// sections off, plain
	var buf bytes.Buffer
	writeOutput(&buf, fields, false, false)
	want := "TOKEN=abc\nTOKEN=def\n"
	if buf.String() != want {
		t.Errorf("plain output = %q, want %q", buf.String(), want)
	}

	// sections on, plain — prefixed AND unprefixed lines, as in the original
	buf.Reset()
	writeOutput(&buf, fields, false, true)
	want = "WORK_TOKEN=abc\nTOKEN=abc\nGENERAL_TOKEN=def\nTOKEN=def\n"
	if buf.String() != want {
		t.Errorf("sections output = %q, want %q", buf.String(), want)
	}

	// fish, sections off
	buf.Reset()
	writeOutput(&buf, fields[:1], true, false)
	want = "set -gx TOKEN 'abc'\n"
	if buf.String() != want {
		t.Errorf("fish output = %q, want %q", buf.String(), want)
	}
}
