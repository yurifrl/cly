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

func TestRegister_NewSubcommands(t *testing.T) {
	root := newTestRoot()

	for _, name := range []string{"sync", "start", "add", "logs", "status", "stop"} {
		cmd, _, err := root.Find([]string{"agents", name})
		if err != nil {
			t.Fatalf("find agents %s: %v", name, err)
		}
		if cmd.Use == "" {
			t.Fatalf("expected subcommand %q to exist", name)
		}
	}
}

func TestRegister_RemovedSubcommands(t *testing.T) {
	root := newTestRoot()

	for _, name := range []string{"run", "configure", "daemon"} {
		cmd, _, _ := root.Find([]string{"agents", name})
		if cmd != nil && cmd.Name() == name {
			t.Fatalf("subcommand %q should be removed", name)
		}
	}
}

func TestRegister_LogsFlags(t *testing.T) {
	root := newTestRoot()
	cmd, _, err := root.Find([]string{"agents", "logs"})
	if err != nil {
		t.Fatalf("find agents logs: %v", err)
	}

	if cmd.Flags().Lookup("follow") == nil {
		t.Fatal("missing --follow on logs")
	}
	if cmd.Flags().Lookup("tail") == nil {
		t.Fatal("missing --tail on logs")
	}
}

func TestRegister_BareAgentsShowsHelp(t *testing.T) {
	root := newTestRoot()
	cmd, _, _ := root.Find([]string{"agents"})
	if cmd.RunE != nil {
		t.Fatal("bare 'agents' should not have RunE (show help instead)")
	}
}
