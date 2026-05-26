package every

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestRegister(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	Register(root)
	var found *cobra.Command
	for _, c := range root.Commands() {
		if c.Use == "every <interval> [-n NAME] -- <command...>" || strings.HasPrefix(c.Use, "every") {
			found = c
		}
	}
	if found == nil {
		t.Fatal("every command not registered")
	}
	subs := map[string]bool{"status": false, "logs": false, "prune": false}
	for _, c := range found.Commands() {
		for k := range subs {
			if strings.HasPrefix(c.Use, k) {
				subs[k] = true
			}
		}
	}
	for k, v := range subs {
		if !v {
			t.Errorf("missing subcommand %s", k)
		}
	}
}

func TestPruneCmd(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLY_EVERY_DIR", dir)

	now := time.Now()
	WriteState(StatePath(dir, "orph"), &State{Name: "orph", PID: 0, LastRunAt: now.Add(-30 * 24 * time.Hour)})
	WriteState(StatePath(dir, "active"), &State{Name: "active", PID: 1 << 30 /* unlikely live */, LastRunAt: now})

	root := &cobra.Command{Use: "root"}
	Register(root)
	out := &bytes.Buffer{}
	root.SetOut(out)
	root.SetErr(out)
	root.SetArgs([]string{"every", "prune"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out.String(), "orphan:  1") {
		t.Fatalf("expected orphan in output, got: %s", out.String())
	}
}

func TestStatusCmdEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLY_EVERY_DIR", dir)
	root := &cobra.Command{Use: "root"}
	Register(root)
	out := &bytes.Buffer{}
	root.SetOut(out)
	root.SetErr(out)
	root.SetArgs([]string{"every", "status"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out.String(), "TASK") {
		t.Fatalf("expected header, got: %s", out.String())
	}
}

func TestAutoName(t *testing.T) {
	a := autoName([]string{"echo", "hi"})
	if len(a) != 8 {
		t.Fatalf("autoName length: %s", a)
	}
	b := autoName([]string{"echo", "hi"})
	if a != b {
		t.Fatal("autoName not deterministic")
	}
}
