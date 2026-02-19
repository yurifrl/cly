package agents

import (
	"testing"

	"github.com/spf13/cobra"
)

func newTestRoot() *cobra.Command {
	root := &cobra.Command{Use: "cly"}
	Register(root)
	return root
}

func TestRegister_RunSubcommand(t *testing.T) {
	root := newTestRoot()
	// "agents run" should exist
	cmd, _, err := root.Find([]string{"agents", "run"})
	if err != nil {
		t.Fatalf("find agents run: %v", err)
	}
	if cmd.Use != "run" {
		t.Fatalf("expected 'run', got %q", cmd.Use)
	}

	// start should have -d flag
	f := cmd.Flags().Lookup("detach")
	if f == nil {
		t.Fatal("missing --detach flag")
	}
	if f.Shorthand != "d" {
		t.Fatalf("expected shorthand 'd', got %q", f.Shorthand)
	}

	// start should have --rm flag
	f = cmd.Flags().Lookup("rm")
	if f == nil {
		t.Fatal("missing --rm flag")
	}
}

func TestRegister_NoDaemonOrSyncSubcommand(t *testing.T) {
	root := newTestRoot()

	// "daemon" should not exist
	cmd, _, _ := root.Find([]string{"agents", "daemon"})
	if cmd != nil && cmd.Use == "daemon" {
		t.Fatal("daemon subcommand should be removed")
	}

	// "sync" should not exist
	cmd, _, _ = root.Find([]string{"agents", "sync"})
	if cmd != nil && cmd.Use == "sync" {
		t.Fatal("sync subcommand should be removed")
	}
}

func TestRegister_StatusStopConfigure(t *testing.T) {
	root := newTestRoot()

	for _, name := range []string{"status", "stop", "configure"} {
		cmd, _, err := root.Find([]string{"agents", name})
		if err != nil {
			t.Fatalf("find agents %s: %v", name, err)
		}
		if cmd.Use != name {
			t.Fatalf("expected %q, got %q", name, cmd.Use)
		}
	}
}

func TestRegister_BareAgentsShowsHelp(t *testing.T) {
	root := newTestRoot()
	cmd, _, _ := root.Find([]string{"agents"})
	// runDefault should not be set (bare agents = help)
	if cmd.RunE != nil {
		t.Fatal("bare 'agents' should not have RunE (show help instead)")
	}
}
