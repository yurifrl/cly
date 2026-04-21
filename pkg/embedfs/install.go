// Package embedfs mirrors files from an embedded filesystem onto disk.
// Used by cly modules that ship bundled assets (skills, pi extensions, ...).
package embedfs

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Install walks srcFS starting at root and mirrors every file into destDir.
// Files under root/<pkg>/... become destDir/<pkg>/... — the root prefix is stripped.
// Overwrites existing files. When dryRun is true, no filesystem changes are made.
// Writes one status line per file to out: "wrote ...", "overwrote ...", or "would write ...".
func Install(srcFS fs.FS, root, destDir string, dryRun bool, out io.Writer) error {
	return InstallSelected(srcFS, root, destDir, nil, dryRun, out)
}

// InstallSelected behaves like Install but restricts installation to the named
// top-level packages (direct children of root). Unknown names return an error.
// A nil or empty names slice installs everything (same as Install).
func InstallSelected(srcFS fs.FS, root, destDir string, names []string, dryRun bool, out io.Writer) error {
	if len(names) > 0 {
		entries, err := fs.ReadDir(srcFS, root)
		if err != nil {
			return err
		}
		available := map[string]bool{}
		for _, e := range entries {
			if e.IsDir() {
				available[e.Name()] = true
			}
		}
		for _, n := range names {
			if !available[n] {
				return fmt.Errorf("unknown package %q (available: %v)", n, keys(available))
			}
		}
	}
	selected := map[string]bool{}
	for _, n := range names {
		selected[n] = true
	}
	return fs.WalkDir(srcFS, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == root {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		if len(selected) > 0 {
			top := rel
			if idx := strings.IndexByte(filepath.ToSlash(rel), '/'); idx >= 0 {
				top = filepath.ToSlash(rel)[:idx]
			}
			if !selected[top] {
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
		}
		destPath := filepath.Join(destDir, rel)

		if d.IsDir() {
			if dryRun {
				return nil
			}
			return os.MkdirAll(destPath, 0o755)
		}

		_, statErr := os.Stat(destPath)
		exists := statErr == nil

		if dryRun {
			fmt.Fprintf(out, "would write %s\n", destPath)
			return nil
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return err
		}

		data, err := fs.ReadFile(srcFS, p)
		if err != nil {
			return err
		}
		if err := os.WriteFile(destPath, data, 0o644); err != nil {
			return err
		}
		verb := "wrote"
		if exists {
			verb = "overwrote"
		}
		fmt.Fprintf(out, "%s %s\n", verb, destPath)
		return nil
	})
}

// ResolveTarget expands a leading "~/" to the user home and cleans the path.
func ResolveTarget(p string) (string, error) {
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		p = filepath.Join(home, p[2:])
	}
	return filepath.Clean(p), nil
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
