package skills

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yurifrl/cly/pkg/embedfs"
)

func TestInstall_WritesAllEmbeddedFiles(t *testing.T) {
	dest := t.TempDir()
	var out bytes.Buffer
	if err := embedfs.Install(embedded, "embedded", dest, false, &out); err != nil {
		t.Fatalf("install: %v", err)
	}
	skill := filepath.Join(dest, "agents-session", "SKILL.md")
	b, err := os.ReadFile(skill)
	if err != nil {
		t.Fatalf("read %s: %v", skill, err)
	}
	if !bytes.Contains(b, []byte("# agents-session")) {
		t.Fatalf("SKILL.md content unexpected")
	}
	if !strings.Contains(out.String(), "wrote "+skill) {
		t.Fatalf("stdout missing wrote line:\n%s", out.String())
	}
}

func TestInstall_OverwritesExisting(t *testing.T) {
	dest := t.TempDir()
	if err := embedfs.Install(embedded, "embedded", dest, false, &bytes.Buffer{}); err != nil {
		t.Fatalf("first install: %v", err)
	}
	var out bytes.Buffer
	if err := embedfs.Install(embedded, "embedded", dest, false, &out); err != nil {
		t.Fatalf("second install: %v", err)
	}
	if !strings.Contains(out.String(), "overwrote ") {
		t.Fatalf("expected overwrote line:\n%s", out.String())
	}
}

func TestInstall_DryRun(t *testing.T) {
	dest := t.TempDir()
	var out bytes.Buffer
	if err := embedfs.Install(embedded, "embedded", dest, true, &out); err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	entries, _ := os.ReadDir(dest)
	if len(entries) != 0 {
		t.Fatalf("dry-run wrote files: %v", entries)
	}
	if !strings.Contains(out.String(), "would write ") {
		t.Fatalf("expected would-write line:\n%s", out.String())
	}
}

func TestInstallSelected_CherryPick(t *testing.T) {
	dest := t.TempDir()
	if err := embedfs.InstallSelected(embedded, "embedded", dest, []string{"agents-session"}, false, &bytes.Buffer{}); err != nil {
		t.Fatalf("cherry-pick: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "agents-session", "SKILL.md")); err != nil {
		t.Fatalf("expected agents-session installed: %v", err)
	}
}

func TestInstallSelected_UnknownNameErrors(t *testing.T) {
	dest := t.TempDir()
	err := embedfs.InstallSelected(embedded, "embedded", dest, []string{"nope"}, false, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown package") {
		t.Fatalf("expected unknown-package error, got %v", err)
	}
}
