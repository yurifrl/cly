package ompsummary

import "testing"

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
