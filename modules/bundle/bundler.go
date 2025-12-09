package bundle

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Bundler defines the interface for package managers
type Bundler interface {
	Name() string
	DefaultFile() string
	StateFile() string
	CheckDeps() error
	Sync(bundleFile string, dryRun bool) error
}

// BaseBundler provides common functionality for bundlers that use state files
type BaseBundler struct {
	name      string
	file      string
	stateFile string
}

// ParseBundleFile reads packages from a bundle file, ignoring comments and blanks
func ParseBundleFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
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

	return packages, scanner.Err()
}

// LoadState reads the state file to get previously installed packages
func LoadState(path string) ([]string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var packages []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			packages = append(packages, line)
		}
	}

	return packages, scanner.Err()
}

// SaveState writes the current installed packages to state file
func SaveState(path string, packages []string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, pkg := range packages {
		fmt.Fprintln(file, pkg)
	}

	return nil
}

// DiffPackages computes packages to install and remove
func DiffPackages(desired, installed []string) (toInstall, toRemove []string) {
	desiredSet := make(map[string]bool)
	for _, pkg := range desired {
		desiredSet[pkg] = true
	}

	installedSet := make(map[string]bool)
	for _, pkg := range installed {
		installedSet[pkg] = true
	}

	// Packages to install: in desired but not in installed
	for _, pkg := range desired {
		if !installedSet[pkg] {
			toInstall = append(toInstall, pkg)
		}
	}

	// Packages to remove: in installed but not in desired
	for _, pkg := range installed {
		if !desiredSet[pkg] {
			toRemove = append(toRemove, pkg)
		}
	}

	return
}

// Color helpers
const (
	colorRed    = "\033[0;31m"
	colorGreen  = "\033[0;32m"
	colorYellow = "\033[1;33m"
	colorReset  = "\033[0m"
)

func printGreen(format string, args ...interface{}) {
	fmt.Printf(colorGreen+format+colorReset+"\n", args...)
}

func printYellow(format string, args ...interface{}) {
	fmt.Printf(colorYellow+format+colorReset+"\n", args...)
}

func printRed(format string, args ...interface{}) {
	fmt.Printf(colorRed+format+colorReset+"\n", args...)
}
