package ompsummary

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureSidebarInstallsEmbeddedSource(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sidebars", "omp-cards.swift")
	if err := EnsureSidebarAt(p); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != SidebarSource {
		t.Fatalf("installed content differs from embedded source")
	}
}

func TestEnsureSidebarIsIdempotent(t *testing.T) {
	p := filepath.Join(t.TempDir(), "omp-cards.swift")
	if err := EnsureSidebarAt(p); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := EnsureSidebarAt(p); err != nil {
		t.Fatal(err)
	}
	fi2, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if !fi.ModTime().Equal(fi2.ModTime()) {
		t.Fatal("second install rewrote an identical file")
	}
}

func TestEnsureSidebarRestoresTamperedCopy(t *testing.T) {
	p := filepath.Join(t.TempDir(), "omp-cards.swift")
	if err := EnsureSidebarAt(p); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("// tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EnsureSidebarAt(p); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != SidebarSource {
		t.Fatal("tampered file was not restored to embedded source")
	}
}
