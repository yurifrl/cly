package claude

import (
	"reflect"
	"testing"
)

func TestTernACPArgs(t *testing.T) {
	tests := []struct {
		in  []string
		out []string
		ok  bool
	}{
		{nil, nil, true},
		{[]string{"-r"}, []string{"--sessions"}, true},
		{[]string{"-r", "abc"}, []string{"--resume", "abc"}, true},
		{[]string{"--resume", "abc"}, []string{"--resume", "abc"}, true},
		{[]string{"-c"}, []string{"--continue"}, true},
		{[]string{"--model", "opus"}, []string{"--model", "opus"}, true},
		{[]string{"-p", "hi"}, nil, false},
		{[]string{"-r", "abc", "--verbose"}, nil, false},
	}
	for _, tt := range tests {
		out, ok := ternACPArgs(tt.in)
		if ok != tt.ok || (ok && !reflect.DeepEqual(out, tt.out)) {
			t.Errorf("ternACPArgs(%v) = %v, %v; want %v, %v", tt.in, out, ok, tt.out, tt.ok)
		}
	}
}
