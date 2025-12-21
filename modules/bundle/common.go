package bundle

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/yurifrl/cly/pkg/store"
)

const (
	colorRed    = "\033[0;31m"
	colorGreen  = "\033[0;32m"
	colorYellow = "\033[1;33m"
	colorReset  = "\033[0m"
)

func printGreen(msg string) {
	fmt.Printf("%s%s%s\n", colorGreen, msg, colorReset)
}

func printYellow(msg string) {
	fmt.Printf("%s%s%s\n", colorYellow, msg, colorReset)
}

func printRed(msg string) {
	fmt.Printf("%s%s%s\n", colorRed, msg, colorReset)
}

// expandPath expands ~ to home directory.
func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}

// parseFile reads a bundle file and returns package names.
// Ignores comments (lines starting with #) and empty lines.
func parseFile(path string) ([]string, error) {
	path = expandPath(path)

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open bundle file: %w", err)
	}
	defer file.Close()

	var packages []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		packages = append(packages, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read bundle file: %w", err)
	}

	return packages, nil
}

// diff returns items in a that are not in b.
func diff(a, b []string) []string {
	bSet := make(map[string]bool)
	for _, item := range b {
		bSet[item] = true
	}

	var result []string
	for _, item := range a {
		if !bSet[item] {
			result = append(result, item)
		}
	}
	return result
}

// commandExists checks if a command is available in PATH.
func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// baseBundler provides common functionality for bundlers that use the Store.
type baseBundler struct {
	name            string
	defaultFile     string
	store           store.Store
	installFn       func(pkg string, verbose bool, force bool) error
	uninstallFn     func(pkg string, verbose bool) error
	listInstalledFn func() ([]string, error) // optional: check actual system state
}

func (b *baseBundler) Name() string {
	return b.name
}

func (b *baseBundler) DefaultFile() string {
	return b.defaultFile
}

func (b *baseBundler) Sync(bundleFile string, verbose bool, force bool, taps bool) error {
	desired, err := parseFile(bundleFile)
	if err != nil {
		return err
	}

	// Use actual system state if available, otherwise fall back to store
	var installed []string
	if b.listInstalledFn != nil {
		installed, err = b.listInstalledFn()
		if err != nil {
			return fmt.Errorf("failed to list installed packages: %w", err)
		}
	} else {
		installed, err = b.store.List(b.name)
		if err != nil {
			return fmt.Errorf("failed to list installed packages: %w", err)
		}
	}

	toRemove := diff(installed, desired)
	for _, pkg := range toRemove {
		printYellow(fmt.Sprintf("Removing: %s", pkg))
		if err := b.uninstallFn(pkg, verbose); err != nil {
			printRed(fmt.Sprintf("Warning: failed to uninstall %s: %v", pkg, err))
		}
		if err := b.store.Remove(b.name, pkg); err != nil {
			printRed(fmt.Sprintf("Warning: failed to remove %s from store: %v", pkg, err))
		}
	}

	var toInstall []string
	if force {
		// Force reinstall all desired packages
		toInstall = desired
	} else {
		// Only install missing packages
		toInstall = diff(desired, installed)
	}

	var failed []string
	for _, pkg := range toInstall {
		printGreen(fmt.Sprintf("Installing: %s", pkg))
		if err := b.installFn(pkg, verbose, force); err != nil {
			printRed(fmt.Sprintf("Failed to install %s: %v", pkg, err))
			failed = append(failed, pkg)
			continue
		}
		if err := b.store.Add(b.name, pkg); err != nil {
			printRed(fmt.Sprintf("Warning: failed to add %s to store: %v", pkg, err))
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("failed to install %d package(s): %v", len(failed), failed)
	}

	printGreen("\nDone!")
	return nil
}

func (b *baseBundler) Check(bundleFile string) error {
	desired, err := parseFile(bundleFile)
	if err != nil {
		return err
	}

	// Use actual system state if available, otherwise fall back to store
	var installed []string
	if b.listInstalledFn != nil {
		installed, err = b.listInstalledFn()
		if err != nil {
			return fmt.Errorf("failed to list installed packages: %w", err)
		}
	} else {
		installed, err = b.store.List(b.name)
		if err != nil {
			return fmt.Errorf("failed to list installed packages: %w", err)
		}
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

func (b *baseBundler) Cleanup(bundleFile string, verbose bool, force bool) error {
	desired, err := parseFile(bundleFile)
	if err != nil {
		return err
	}

	installed, err := b.store.List(b.name)
	if err != nil {
		return fmt.Errorf("failed to list installed packages: %w", err)
	}

	toRemove := diff(installed, desired)
	if len(toRemove) == 0 {
		printGreen("Nothing to clean up")
		return nil
	}

	for _, pkg := range toRemove {
		printYellow(fmt.Sprintf("Removing: %s", pkg))
		if err := b.uninstallFn(pkg, verbose); err != nil {
			printRed(fmt.Sprintf("Warning: failed to uninstall %s: %v", pkg, err))
		}
		if err := b.store.Remove(b.name, pkg); err != nil {
			printRed(fmt.Sprintf("Warning: failed to remove %s from store: %v", pkg, err))
		}
	}

	printGreen("\nCleanup done!")
	return nil
}
