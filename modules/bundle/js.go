package bundle

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	pkgconfig "github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/store"
)

// JsBundler manages JavaScript packages via pnpm.
type JsBundler struct {
	*baseBundler
}

// PackageJSON represents a package.json file.
type PackageJSON struct {
	Dependencies map[string]string `json:"dependencies"`
}

// pnpmPackage represents a package from pnpm list --json output.
type pnpmPackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// NewJsBundler creates a new JsBundler.
func NewJsBundler(s store.Store) *JsBundler {
	b := &JsBundler{}
	jsFile := pkgconfig.GetString("modules.bundle.js_file")
	if jsFile == "" {
		jsFile = "~/.config/cly/package.json"
	}
	b.baseBundler = &baseBundler{
		name:            "js",
		defaultFile:     jsFile,
		store:           s,
		installFn:       b.install,
		uninstallFn:     b.uninstall,
		listInstalledFn: b.listInstalled,
	}
	return b
}

func (b *JsBundler) CheckDeps() error {
	if !commandExists("pnpm") {
		return fmt.Errorf("pnpm not found. Add to Brewfile: 'pnpm', then run: cly bundle brew")
	}
	return nil
}

func (b *JsBundler) install(pkg string, verbose bool, force bool) error {
	args := []string{"add", "-g", pkg}
	if force {
		args = append(args, "--force")
	}
	cmd := exec.Command("pnpm", args...)

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pnpm add failed: %w", err)
	}

	// Always prune after install
	if err := b.prune(verbose); err != nil {
		return fmt.Errorf("pnpm prune failed: %w", err)
	}

	return nil
}

func (b *JsBundler) uninstall(pkg string, verbose bool) error {
	cmd := exec.Command("pnpm", "remove", "-g", pkg)

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pnpm remove failed: %w", err)
	}
	return nil
}

func (b *JsBundler) listInstalled() ([]string, error) {
	cmd := exec.Command("pnpm", "list", "-g", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pnpm list failed: %w", err)
	}

	var packages []pnpmPackage
	if err := json.Unmarshal(output, &packages); err != nil {
		return nil, fmt.Errorf("failed to parse pnpm output: %w", err)
	}

	var names []string
	for _, pkg := range packages {
		names = append(names, pkg.Name)
	}
	return names, nil
}

func (b *JsBundler) prune(verbose bool) error {
	cmd := exec.Command("pnpm", "prune", "--global")
	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd.Run()
}

// Sync overrides baseBundler.Sync to handle package.json format.
func (b *JsBundler) Sync(bundleFile string, verbose bool, force bool, noUpdate bool, taps bool) error {
	bundleFile = expandPath(bundleFile)

	// Ensure directory exists
	dir := filepath.Dir(bundleFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create package.json if it doesn't exist
	if _, err := os.Stat(bundleFile); os.IsNotExist(err) {
		emptyPkg := PackageJSON{Dependencies: make(map[string]string)}
		data, _ := json.MarshalIndent(emptyPkg, "", "  ")
		if err := os.WriteFile(bundleFile, data, 0644); err != nil {
			return fmt.Errorf("failed to create package.json: %w", err)
		}
	}

	// Read package.json
	pkg, err := parsePackageJSON(bundleFile)
	if err != nil {
		return err
	}

	// Convert dependencies to package list
	var desired []string
	for name, version := range pkg.Dependencies {
		if version != "" && version != "latest" && version != "*" {
			desired = append(desired, fmt.Sprintf("%s@%s", name, version))
		} else {
			desired = append(desired, name)
		}
	}

	// Get installed packages
	installed, err := b.listInstalled()
	if err != nil {
		return fmt.Errorf("failed to list installed packages: %w", err)
	}

	// Remove packages not in desired
	toRemove := diff(installed, desired)
	for _, pkg := range toRemove {
		printYellow(fmt.Sprintf("Removing: %s", pkg))
		if err := b.uninstall(pkg, verbose); err != nil {
			printRed(fmt.Sprintf("Warning: failed to uninstall %s: %v", pkg, err))
		}
		if err := b.store.Remove(b.name, pkg); err != nil {
			printRed(fmt.Sprintf("Warning: failed to remove %s from store: %v", pkg, err))
		}
	}

	// Install missing packages
	var toInstall []string
	if force {
		toInstall = desired
	} else {
		toInstall = diff(desired, installed)
	}

	var failed []string
	for _, pkg := range toInstall {
		printGreen(fmt.Sprintf("Installing: %s", pkg))
		if err := b.install(pkg, verbose, force); err != nil {
			printRed(fmt.Sprintf("Failed to install %s: %v", pkg, err))
			failed = append(failed, pkg)
			continue
		}
		// Extract base name for store (remove version specs)
		baseName := pkg
		if idx := indexOf(pkg, '@'); idx > 0 {
			baseName = pkg[:idx]
		}
		if err := b.store.Add(b.name, baseName); err != nil {
			printRed(fmt.Sprintf("Warning: failed to add %s to store: %v", pkg, err))
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("failed to install %d package(s): %v", len(failed), failed)
	}

	printGreen("\nDone!")
	return nil
}

// Check overrides baseBundler.Check to handle package.json format.
func (b *JsBundler) Check(bundleFile string) error {
	bundleFile = expandPath(bundleFile)

	// Read package.json
	pkg, err := parsePackageJSON(bundleFile)
	if err != nil {
		return err
	}

	// Convert dependencies to package list
	var desired []string
	for name, version := range pkg.Dependencies {
		if version != "" && version != "latest" && version != "*" {
			desired = append(desired, fmt.Sprintf("%s@%s", name, version))
		} else {
			desired = append(desired, name)
		}
	}

	// Get installed packages
	installed, err := b.listInstalled()
	if err != nil {
		return fmt.Errorf("failed to list installed packages: %w", err)
	}

	toInstall := diff(desired, installed)
	toRemove := diff(installed, desired)

	if len(toInstall) == 0 && len(toRemove) == 0 {
		printGreen("Everything is in sync")
		return nil
	}

	if len(toInstall) > 0 {
		printGreen("Would install:")
		for _, pkg := range toInstall {
			fmt.Printf("  + %s\n", pkg)
		}
	}

	if len(toRemove) > 0 {
		printYellow("Would remove:")
		for _, pkg := range toRemove {
			fmt.Printf("  - %s\n", pkg)
		}
	}

	return fmt.Errorf("changes needed")
}

// parsePackageJSON reads and parses a package.json file.
func parsePackageJSON(path string) (*PackageJSON, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read package.json: %w", err)
	}

	var pkg PackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse package.json: %w", err)
	}

	if pkg.Dependencies == nil {
		pkg.Dependencies = make(map[string]string)
	}

	return &pkg, nil
}

func indexOf(s string, c rune) int {
	for i, r := range s {
		if r == c {
			return i
		}
	}
	return -1
}
