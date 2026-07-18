package pi

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yurifrl/cly/pkg/embedfs"
)

func TestInstall_WritesPiCly(t *testing.T) {
	dest := t.TempDir()
	var out bytes.Buffer
	if err := embedfs.Install(embedded, "embedded", dest, false, &out); err != nil {
		t.Fatalf("install: %v", err)
	}
	p := filepath.Join(dest, "cly.ts")
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("missing %s: %v", p, err)
	}
	if !strings.Contains(out.String(), "wrote ") {
		t.Fatalf("expected wrote lines:\n%s", out.String())
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
