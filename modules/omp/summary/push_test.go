package ompsummary

import "testing"

func TestCard(t *testing.T) {
	cases := []struct {
		name  string
		entry *Entry
		want  string
	}{
		{
			name: "full entry",
			entry: &Entry{
				Title:    "fix login",
				UserAsk:  "why does login fail?",
				Goal:     "debug auth flow",
				Response: "found the nil check",
				Action:   "confirm staging env",
				Timeline: []Turn{
					{Who: "user", Text: "look at logs"},
					{Who: "ai", Text: "read auth.go"},
				},
				UpdatedAt: 1_000_000,
			},
			want: "fix login|now|why does login fail?|debug auth flow|found the nil check|confirm staging env|~|look at logs|read auth.go",
		},
		{
			name:  "header falls back to surface id",
			entry: &Entry{SurfaceID: "s1", Goal: "g", UpdatedAt: 1_000_000},
			want:  "s1|now|~|g|~|~|~",
		},
		{
			name:  "empty response and action padded",
			entry: &Entry{Title: "t", UserAsk: "q", Goal: "g", UpdatedAt: 1_000_000},
			want:  "t|now|q|g|~|~|~",
		},
		{
			name: "timeline mixed who in order",
			entry: &Entry{
				Title: "t",
				Timeline: []Turn{
					{Who: "ai", Text: "first"},
					{Who: "user", Text: "second"},
					{Who: "ai", Text: "third"},
				},
				UpdatedAt: 1_000_000,
			},
			want: "t|now|~|~|~|~|~|first|second|third",
		},
		{
			name: "payload hazards sanitized",
			entry: &Entry{
				Title:    "pi|pe",
				UserAsk:  "new\nline",
				Response: "~",
				UpdatedAt: 1_000_000,
			},
			want: "pi/pe|now|new line|~|~ |~|~",
		},
		{
			name:  "timeline capped at timelineMax rows",
			entry: &Entry{
				Title: "t",
				Timeline: []Turn{
					{Who: "user", Text: "1"},
					{Who: "ai", Text: "2"},
					{Who: "user", Text: "3"},
					{Who: "ai", Text: "4"},
					{Who: "user", Text: "5"},
					{Who: "ai", Text: "6"},
				},
				UpdatedAt: 1_000_000,
			},
			want: "t|now|~|~|~|~|~|1|2|3|4|5",
		},
		{
			name: "compaction row between action and timeline",
			entry: &Entry{
				Title:       "t",
				Compactions: 3,
				CompactedAt: 999_940, // 60s before the frozen clock → "1m"
				FoldNote:    "~327k older chars dropped",
				Timeline:    []Turn{{Who: "user", Text: "next"}},
				UpdatedAt:   1_000_000,
			},
			want: "t|now|~|~|~|~|3 folds · last 1m · ~327k older chars dropped|next",
		},
		{
			name:  "empty entry is stamp-only fallback",
			entry: &Entry{},
			want:  "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Freeze the clock so relStamp is deterministic.
			orig := nowUnix
			nowUnix = func() float64 { return 1_000_000 }
			defer func() { nowUnix = orig }()
			if got := Card(c.entry); got != c.want {
				t.Errorf("Card() =\n%q\nwant\n%q", got, c.want)
			}
		})
	}
}

func TestRelStamp(t *testing.T) {
	const now = 1_000_000.0
	cases := []struct {
		age  float64
		want string
	}{
		{0, "now"},
		{59, "now"},
		{60, "1m"},
		{120, "2m"},
		{3599, "59m"},
		{3600, "1h"},
		{7200, "2h"},
		{86399, "23h"},
		{86400, "1d"},
		{172800, "2d"},
	}
	for _, c := range cases {
		if got := relStamp(now-c.age, now); got != c.want {
			t.Errorf("relStamp(age=%v) = %q, want %q", c.age, got, c.want)
		}
	}
	if got := relStamp(now+10, now); got != "now" {
		t.Errorf("relStamp(future) = %q, want now", got)
	}
}
