package bundle

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	pkgconfig "github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/store"
)

// RustBundler manages Rust crates via cargo.
type RustBundler struct {
	*baseBundler
}

// NewRustBundler creates a new RustBundler.
func NewRustBundler(s store.Store) *RustBundler {
	b := &RustBundler{}
	rustFile := pkgconfig.GetString("modules.bundle.rust_file")
	if rustFile == "" {
		rustFile = "~/.config/Rsfile"
	}
	b.baseBundler = &baseBundler{
		name:            "rust",
		defaultFile:     rustFile,
		store:           s,
		installFn:       b.install,
		uninstallFn:     b.uninstall,
		listInstalledFn: b.listInstalled,
	}
	return b
}

func (b *RustBundler) CheckDeps() error {
	if !commandExists("cargo") {
		return fmt.Errorf("cargo not found. Add to Brewfile: 'rust', then run: cly bundle brew")
	}
	return nil
}

func (b *RustBundler) install(pkg string, verbose bool, force bool) error {
	crate, version := parseRustPkg(pkg)

	args := []string{"install", crate}
	if version != "" {
		args = append(args, "--version", version)
	}
	if force {
		args = append(args, "--force")
	}
	cmd := exec.Command("cargo", args...)

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if version != "" {
			fmt.Printf("  → Installing %s version %s\n", crate, version)
		}
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cargo install failed: %w", err)
	}
	return nil
}

func (b *RustBundler) uninstall(pkg string, verbose bool) error {
	basePkg := extractRustBase(pkg)

	cmd := exec.Command("cargo", "uninstall", basePkg)

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cargo uninstall failed: %w", err)
	}
	return nil
}

func (b *RustBundler) listInstalled() ([]string, error) {
	cmd := exec.Command("cargo", "install", "--list")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("cargo install --list failed: %w", err)
	}

	var crates []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Format: "crate-name v1.2.3"
		// Extract just the name
		parts := strings.Fields(line)
		if len(parts) > 0 {
			crates = append(crates, parts[0])
		}
	}

	return crates, nil
}

// parseRustPkg parses crate name and version from package spec.
// "crate --version 1.2.3" → "crate", "1.2.3"
// "crate" → "crate", ""
func parseRustPkg(pkg string) (crate string, version string) {
	parts := strings.Fields(pkg)
	if len(parts) == 0 {
		return "", ""
	}

	crate = parts[0]
	for i, p := range parts {
		if p == "--version" && i+1 < len(parts) {
			version = parts[i+1]
			break
		}
	}
	return crate, version
}

// extractRustBase extracts base crate name from spec.
// "crate-name" → "crate-name"
// "crate-name@1.0.0" → "crate-name"
// "crate-name@1.0.0 --force" → "crate-name"
func extractRustBase(pkg string) string {
	// Remove version specs: crate-name@1.0.0 → crate-name
	basePkg := strings.Split(pkg, "@")[0]
	// Remove flags
	parts := strings.Fields(basePkg)
	if len(parts) > 0 {
		return parts[0]
	}
	return basePkg
}
