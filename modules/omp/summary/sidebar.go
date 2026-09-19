package ompsummary

import (
	_ "embed"
	"os"
	"path/filepath"
)

// SidebarSource is the omp-cards cmux custom sidebar shipped inside the
// binary. The daemon installs it on start, so a binary distribution carries
// its own sidebar — no repo checkout required on the user's machine.
//
//go:embed omp-cards.swift
var SidebarSource string

// SidebarPath is where cmux picks up custom sidebars.
func SidebarPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "cmux", "sidebars", "omp-cards.swift"), nil
}

// EnsureSidebar installs the embedded sidebar to the cmux sidebars directory:
// writes when the deployed copy is missing or differs from the binary's
// version (the binary is the source of truth), no-op otherwise. Idempotent.
func EnsureSidebar() error {
	p, err := SidebarPath()
	if err != nil {
		return err
	}
	return EnsureSidebarAt(p)
}

// EnsureSidebarAt is EnsureSidebar against an explicit path (testable).
func EnsureSidebarAt(path string) error {
	if cur, err := os.ReadFile(path); err == nil && string(cur) == SidebarSource {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(SidebarSource), 0o644)
}
