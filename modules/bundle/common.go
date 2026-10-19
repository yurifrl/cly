package bundle

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/yurifrl/cly/pkg/store"
	"github.com/yurifrl/cly/pkg/style"
)

func printGreen(msg string) {
	fmt.Println(style.GreenStyle.Render(msg))
}

func printYellow(msg string) {
	fmt.Println(style.YellowStyle.Render(msg))
}

func printRed(msg string) {
	fmt.Println(style.RedStyle.Render(msg))
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
	alwaysUpgrade   bool                     // always reinstall/upgrade all desired packages
}

func (b *baseBundler) Name() string {
	return b.name
}

func (b *baseBundler) DefaultFile() string {
	return b.defaultFile
}

func (b *baseBundler) Sync(bundleFile string, verbose bool, force bool, noUpdate bool, taps bool, mas bool) error {
	desired, err := parseFile(bundleFile)
	if err != nil {
		return err
	}

	// Use actual system state if available, otherwise fall back to store
	var installed []string
	useBaseNameComparison := false
	if b.listInstalledFn != nil {
		installed, err = b.listInstalledFn()
		if err != nil {
			return fmt.Errorf("failed to list installed packages: %w", err)
		}
		useBaseNameComparison = true
	} else {
		installed, err = b.store.List(b.name)
		if err != nil {
			return fmt.Errorf("failed to list installed packages: %w", err)
		}
	}

	// When using listInstalledFn, installed returns base names only.
	// We need to compare using base names and remove packages not in desired.
	var toRemove []string
	if useBaseNameComparison {
		toRemove = diffByBaseName(installed, desired)
	} else {
		toRemove = diff(installed, desired)
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

	var toInstall []string
	if force {
		// Force reinstall all desired packages - uninstall first
		toInstall = desired
		for _, pkg := range desired {
			isInstalled := false
			basePkg := extractBasePkg(pkg)
			for _, inst := range installed {
				if useBaseNameComparison {
					if inst == basePkg {
						isInstalled = true
						break
					}
				} else if inst == pkg {
					isInstalled = true
					break
				}
			}
			if isInstalled {
				printYellow(fmt.Sprintf("Force uninstalling: %s", pkg))
				_ = b.uninstallFn(pkg, verbose)
			}
		}
	} else if b.alwaysUpgrade {
		// Upgrade all desired packages without uninstalling first
		toInstall = desired
	} else {
		// Only install missing packages (compare base names if using system state)
		if useBaseNameComparison {
			toInstall = diffByBaseName(desired, installed)
		} else {
			toInstall = diff(desired, installed)
		}
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
	useBaseNameComparison := false
	if b.listInstalledFn != nil {
		installed, err = b.listInstalledFn()
		if err != nil {
			return fmt.Errorf("failed to list installed packages: %w", err)
		}
		useBaseNameComparison = true
	} else {
		installed, err = b.store.List(b.name)
		if err != nil {
			return fmt.Errorf("failed to list installed packages: %w", err)
		}
	}

	var toInstall, toRemove []string
	if useBaseNameComparison {
		toInstall = diffByBaseName(desired, installed)
		toRemove = diffByBaseName(installed, desired)
	} else {
		toInstall = diff(desired, installed)
		toRemove = diff(installed, desired)
	}

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

// syncAction is one planned sync operation for a package.
type syncAction struct {
	pkg  string // full spec from the bundle file (may carry pin or tag)
	base string
	verb string // install, upgrade, downgrade, reinstall
	from string
	to   string
}

// label is the sync-time line shown while the action runs.
func (a syncAction) label() string {
	switch a.verb {
	case "install":
		return fmt.Sprintf("Installing: %s", a.pkg)
	case "upgrade":
		return fmt.Sprintf("Upgrading: %s (%s -> %s)", a.base, a.from, a.to)
	case "downgrade":
		return fmt.Sprintf("Downgrading: %s (%s -> %s)", a.base, a.from, a.to)
	default:
		return fmt.Sprintf("Reinstalling: %s", a.base)
	}
}

// doneLabel is the past-tense summary shown by the parallel TUI.
func (a syncAction) doneLabel() string {
	switch a.verb {
	case "upgrade":
		return fmt.Sprintf("upgraded %s -> %s", a.from, a.to)
	case "downgrade":
		return fmt.Sprintf("downgraded %s -> %s", a.from, a.to)
	case "reinstall":
		return "reinstalled"
	default:
		return "installed"
	}
}

// syncSummary renders the end-of-sync tally.
func syncSummary(actions []syncAction, examined int) string {
	if len(actions) == 0 {
		return fmt.Sprintf("Nothing changed - all %d packages up to date", examined)
	}
	counts := make(map[string]int)
	for _, a := range actions {
		counts[a.verb]++
	}
	var parts []string
	add := func(n int, label string) {
		if n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, label))
		}
	}
	add(counts["install"], "installed")
	add(counts["upgrade"], "upgraded")
	add(counts["downgrade"], "downgraded")
	add(counts["reinstall"], "reinstalled")
	if upToDate := examined - len(actions); upToDate > 0 {
		parts = append(parts, fmt.Sprintf("%d up to date", upToDate))
	}
	return strings.Join(parts, ", ")
}

// fetchConcurrent resolves a version for each base package concurrently
// (metadata requests only, at most 8 at a time). Resolved entries are
// returned; failures are named in the error and simply absent from the map
// so callers can leave those packages untouched.
func fetchConcurrent(bases []string, tags map[string]string, fetch func(base string) (string, error)) (map[string]string, error) {
	type result struct {
		base    string
		version string
		err     error
	}
	results := make(chan result, len(bases))
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	for _, base := range bases {
		wg.Add(1)
		go func(base string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			v, err := fetch(base)
			results <- result{base: base, version: v, err: err}
		}(base)
	}
	wg.Wait()
	close(results)

	latest := make(map[string]string, len(bases))
	var failed []string
	for r := range results {
		if r.err != nil {
			failed = append(failed, r.base)
			continue
		}
		latest[r.base] = r.version
	}
	if len(failed) > 0 {
		sort.Strings(failed)
		return latest, fmt.Errorf("could not check latest versions of %s - leaving them untouched", strings.Join(failed, ", "))
	}
	return latest, nil
}

// normalizeVersion strips lock prefixes so versions compare cleanly.
func normalizeVersion(v string) string {
	return strings.TrimLeft(strings.TrimPrefix(v, "="), "v")
}

// compareVersions compares versions with basic semver ordering: dotted
// cores numerically, prerelease ranks below the release it belongs to
// (2.0.0-beta.1 < 2.0.0). Returns -1, 0 or 1.
func compareVersions(a, b string) int {
	ac, ap := splitSemver(normalizeVersion(a))
	bc, bp := splitSemver(normalizeVersion(b))
	if c := compareDotted(ac, bc, true); c != 0 {
		return c
	}
	switch {
	case ap == bp:
		return 0
	case ap == "":
		return 1 // release outranks any prerelease
	case bp == "":
		return -1
	}
	return compareDotted(ap, bp, false)
}

// splitSemver splits "1.2.3-beta.1" into core "1.2.3" and prerelease
// "beta.1"; build metadata after "+" is ignored.
func splitSemver(v string) (core, pre string) {
	if i := strings.IndexByte(v, '+'); i >= 0 {
		v = v[:i]
	}
	if i := strings.IndexByte(v, '-'); i >= 0 {
		return v[:i], v[i+1:]
	}
	return v, ""
}

// compareDotted compares dot-separated identifiers. padZero pads missing
// trailing segments with "0" (core semantics: 1.2 == 1.2.0); without it a
// missing segment is smaller (prerelease semantics per semver). Numeric
// identifiers rank below non-numeric ones in prereleases.
func compareDotted(a, b string, padZero bool) int {
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	for i := range n {
		av, bv := "", ""
		if i < len(as) {
			av = as[i]
		}
		if i < len(bs) {
			bv = bs[i]
		}
		if av == "" || bv == "" {
			if padZero {
				if av == "" && bv == "" {
					return 0
				}
				if av == "" {
					av = "0"
				} else {
					bv = "0"
				}
			} else if av == "" {
				return -1
			} else {
				return 1
			}
		}
		an, aerr := strconv.Atoi(av)
		bn, berr := strconv.Atoi(bv)
		switch {
		case aerr == nil && berr == nil:
			if an != bn {
				if an < bn {
					return -1
				}
				return 1
			}
		case aerr == nil:
			return -1 // numeric identifiers rank below non-numeric
		case berr == nil:
			return 1
		case av != bv:
			if av < bv {
				return -1
			}
			return 1
		}
	}
	return 0
}
