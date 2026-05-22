package piwrap

import (
	"reflect"
	"testing"
)

func TestKebabCase(t *testing.T) {
	cases := map[string]string{
		"Refactor Auth":      "refactor-auth",
		"foo  bar__baz":      "foo-bar-baz",
		"--Hello, World!--":  "hello-world",
		"alreadyKebab":       "alreadykebab",
		"snake_case_thing":   "snake-case-thing",
		"  trim me  ":        "trim-me",
		"v1.2.3 release":     "v1-2-3-release",
	}
	for in, want := range cases {
		if got := kebabCase(in); got != want {
			t.Errorf("kebabCase(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEncodeCwd(t *testing.T) {
	cases := map[string]string{
		"/Users/yuri/Workdir/Yuri/cly": "--Users-yuri-Workdir-Yuri-cly--",
		"/tmp":                          "--tmp--",
	}
	for in, want := range cases {
		if got := encodeCwd(in); got != want {
			t.Errorf("encodeCwd(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBuildSessionPath(t *testing.T) {
	got := buildSessionPath("/home/u", "/Users/yuri/Workdir/Yuri/cly", "cly", "My Session")
	want := "/home/u/.pi/agent/sessions/--Users-yuri-Workdir-Yuri-cly--/cly-my-session.jsonl"
	if got != want {
		t.Errorf("buildSessionPath default = %q, want %q", got, want)
	}

	got = buildSessionPath("/home/u", "/tmp", "cly-custom-session", "My Session")
	want = "/home/u/.pi/agent/sessions/--tmp--/cly-custom-session-my-session.jsonl"
	if got != want {
		t.Errorf("buildSessionPath custom prefix = %q, want %q", got, want)
	}
}

func TestHasSessionFlag(t *testing.T) {
	cases := []struct {
		in   []string
		want bool
	}{
		{[]string{"-p", "hi"}, false},
		{[]string{"--session", "abc"}, true},
		{[]string{"--session=abc"}, true},
		{[]string{"--session-dir", "/x"}, false},
	}
	for _, c := range cases {
		if got := hasSessionFlag(c.in); got != c.want {
			t.Errorf("hasSessionFlag(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestExtractName(t *testing.T) {
	cases := []struct {
		in       []string
		wantName string
		wantRest []string
	}{
		{[]string{"-n", "foo", "-p", "hi"}, "foo", []string{"-p", "hi"}},
		{[]string{"--name", "bar"}, "bar", []string{}},
		{[]string{"--name=baz", "-p"}, "baz", []string{"-p"}},
		{[]string{"-n=qux", "@file"}, "qux", []string{"@file"}},
		{[]string{"-p", "hi"}, "", []string{"-p", "hi"}},
		{[]string{}, "", []string{}},
		// Second --name passes through.
		{[]string{"-n", "first", "--name", "second"}, "first", []string{"--name", "second"}},
	}
	for _, c := range cases {
		gotName, gotRest := extractName(c.in)
		if gotName != c.wantName || !reflect.DeepEqual(gotRest, c.wantRest) {
			t.Errorf("extractName(%v) = (%q, %v), want (%q, %v)",
				c.in, gotName, gotRest, c.wantName, c.wantRest)
		}
	}
}
