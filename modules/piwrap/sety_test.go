package piwrap

import (
	"reflect"
	"testing"
)

func TestExtractPiwrapFlags_Basic(t *testing.T) {
	got, err := extractPiwrapFlags([]string{"-p", "hi"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Sety.HasImportID || got.Sety.HasImportOverride {
		t.Errorf("expected no sety values, got %+v", got.Sety)
	}
	if got.DryRun || got.Helpy {
		t.Errorf("expected no flag activations")
	}
	if !reflect.DeepEqual(got.Rest, []string{"-p", "hi"}) {
		t.Errorf("rest = %v", got.Rest)
	}
}

func TestExtractPiwrapFlags_SetyImportID(t *testing.T) {
	cases := [][]string{
		{"--sety", "session_import.id=019e5057"},
		{"--sety=session_import.id=019e5057"},
		{"-y", "session_import.id=019e5057"},
		{"-y=session_import.id=019e5057"},
		{"--sety-string", "session_import.id=019e5057"},
	}
	for _, c := range cases {
		got, err := extractPiwrapFlags(c)
		if err != nil {
			t.Fatalf("%v: unexpected error: %v", c, err)
		}
		if !got.Sety.HasImportID || got.Sety.ImportID != "019e5057" {
			t.Errorf("%v: ImportID = %q, has=%v", c, got.Sety.ImportID, got.Sety.HasImportID)
		}
		if len(got.Rest) != 0 {
			t.Errorf("%v: expected empty Rest, got %v", c, got.Rest)
		}
	}
}

func TestExtractPiwrapFlags_SetyImportOverride(t *testing.T) {
	cases := []struct {
		args []string
		want bool
	}{
		{[]string{"--sety", "session_import.override=true"}, true},
		{[]string{"--sety", "session_import.override=false"}, false},
		{[]string{"--sety", "session_import.override=1"}, true},
		{[]string{"--sety", "session_import.override=0"}, false},
		{[]string{"--sety", "session_import.override=TRUE"}, true},
	}
	for _, c := range cases {
		got, err := extractPiwrapFlags(c.args)
		if err != nil {
			t.Fatalf("%v: unexpected error: %v", c.args, err)
		}
		if !got.Sety.HasImportOverride || got.Sety.ImportOverride != c.want {
			t.Errorf("%v: override = %v, want %v", c.args, got.Sety.ImportOverride, c.want)
		}
	}
}

func TestExtractPiwrapFlags_BadFormat(t *testing.T) {
	cases := [][]string{
		{"--sety"},                       // missing value
		{"--sety", "noequals"},           // no =
		{"--sety", "=val"},               // empty key
		{"--sety-string"},                // missing value
	}
	for _, c := range cases {
		_, err := extractPiwrapFlags(c)
		if err == nil {
			t.Errorf("%v: expected SETY_FORMAT error, got nil", c)
			continue
		}
		if err.Code != CodeSetyFormat {
			t.Errorf("%v: code = %q, want %q", c, err.Code, CodeSetyFormat)
		}
	}
}

func TestExtractPiwrapFlags_UnknownKey(t *testing.T) {
	_, err := extractPiwrapFlags([]string{"--sety", "modules.piwrap.session_file_name_prefix=foo"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err.Code != CodeSetyUnknownKey {
		t.Errorf("code = %q, want %q", err.Code, CodeSetyUnknownKey)
	}
}

func TestExtractPiwrapFlags_BadBool(t *testing.T) {
	_, err := extractPiwrapFlags([]string{"--sety", "session_import.override=maybe"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err.Code != CodeSetyParse {
		t.Errorf("code = %q, want %q", err.Code, CodeSetyParse)
	}
}

func TestExtractPiwrapFlags_StringForBoolRejected(t *testing.T) {
	_, err := extractPiwrapFlags([]string{"--sety-string", "session_import.override=true"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err.Code != CodeSetyParse {
		t.Errorf("code = %q, want %q", err.Code, CodeSetyParse)
	}
}

func TestExtractPiwrapFlags_DryRunHelpy(t *testing.T) {
	got, err := extractPiwrapFlags([]string{"--dry-run", "--helpy", "-o", "json", "-p", "hi"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.DryRun || !got.Helpy || !got.HelpyJSON {
		t.Errorf("flags = %+v", got)
	}
	if !reflect.DeepEqual(got.Rest, []string{"-p", "hi"}) {
		t.Errorf("rest = %v, want [-p hi]", got.Rest)
	}
}

func TestExtractPiwrapFlags_HelpyOutputFollowsHelpy(t *testing.T) {
	// `-o json` only consumed when --helpy precedes it.
	got, err := extractPiwrapFlags([]string{"-o", "json", "-p", "hi"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.HelpyJSON {
		t.Errorf("HelpyJSON should be false when --helpy is absent")
	}
	if !reflect.DeepEqual(got.Rest, []string{"-o", "json", "-p", "hi"}) {
		t.Errorf("rest = %v", got.Rest)
	}
}

func TestExtractPiwrapFlags_PassThrough(t *testing.T) {
	got, err := extractPiwrapFlags([]string{
		"-p", "hi",
		"--sety", "session_import.id=abc",
		"--model", "gpt-4o",
		"-y", "session_import.override=true",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Sety.HasImportID || got.Sety.ImportID != "abc" {
		t.Errorf("ImportID = %q", got.Sety.ImportID)
	}
	if !got.Sety.HasImportOverride || !got.Sety.ImportOverride {
		t.Errorf("Override not set")
	}
	want := []string{"-p", "hi", "--model", "gpt-4o"}
	if !reflect.DeepEqual(got.Rest, want) {
		t.Errorf("rest = %v, want %v", got.Rest, want)
	}
}
