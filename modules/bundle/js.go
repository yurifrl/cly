package bundle

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	pkgconfig "github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/store"
)

// JsBundler manages JavaScript packages via pnpm or npm.
type JsBundler struct {
	*baseBundler
	manager string // "pnpm" or "npm"
}

// NewJsBundler creates a new JsBundler.
func NewJsBundler(s store.Store) *JsBundler {
	b := &JsBundler{}
	jsFile := pkgconfig.GetString("modules.bundle.js_file")
	if jsFile == "" {
		jsFile = "~/.config/cly/bundles/Jsfile"
	}
	mgr := strings.ToLower(strings.TrimSpace(pkgconfig.GetString("modules.bundle.js_manager")))
	if mgr == "" {
		mgr = "pnpm"
	}
	b.manager = mgr
	b.baseBundler = &baseBundler{
		name:        "js",
		defaultFile: jsFile,
		store:       s,
		installFn:   b.install,
		uninstallFn: b.uninstall,
	}
	return b
}

func (b *JsBundler) CheckDeps() error {
	binPath := findWorkingManager(b.manager)
	if binPath == "" {
		return fmt.Errorf("%s not found or not working. Add to Brewfile: '%s', then run: cly bundle brew", b.manager, b.manager)
	}
	if b.manager == "pnpm" {
		// Ensure PNPM_HOME is set and in PATH for global installs
		pnpmHome := os.Getenv("PNPM_HOME")
		if pnpmHome == "" {
			home, _ := os.UserHomeDir()
			pnpmHome = home + "/Library/pnpm"
			os.Setenv("PNPM_HOME", pnpmHome)
		}
	}
	// Prepend the directory containing the working binary so all exec calls use it
	dir := binPath[:strings.LastIndex(binPath, "/")]
	path := os.Getenv("PATH")
	if !strings.HasPrefix(path, dir+":") {
		os.Setenv("PATH", dir+":"+path)
	}
	return nil
}

// findWorkingManager returns the path to a pnpm/npm binary that actually executes.
// It prefers the one in PATH but falls back to known locations if the PATH
// one is broken (e.g. ~/Library/pnpm/bin/pnpm pointing to a blank placeholder).
func findWorkingManager(name string) string {
	candidates := []string{}
	if p, err := exec.LookPath(name); err == nil {
		candidates = append(candidates, p)
	}
	// Known fallback locations
	candidates = append(candidates, "/opt/homebrew/bin/"+name, "/usr/local/bin/"+name)

	seen := map[string]bool{}
	for _, p := range candidates {
		if seen[p] {
			continue
		}
		seen[p] = true
		cmd := exec.Command(p, "--version")
		if err := cmd.Run(); err == nil {
			return p
		}
	}
	return ""
}

// addSubcommand returns the install subcommand for the manager.
// pnpm uses "add", npm uses "install".
func (b *JsBundler) addSubcommand() string {
	if b.manager == "npm" {
		return "install"
	}
	return "add"
}

// removeSubcommand returns the uninstall subcommand for the manager.
func (b *JsBundler) removeSubcommand() string {
	if b.manager == "npm" {
		return "uninstall"
	}
	return "remove"
}

func (b *JsBundler) install(pkg string, verbose bool, force bool) error {
	args := []string{b.addSubcommand(), "-g", pkg}
	if force {
		args = append(args, "--force")
	}
	cmd := exec.Command(b.manager, args...)

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s failed: %s", b.manager, b.addSubcommand(), string(output))
	}
	return nil
}

func (b *JsBundler) uninstall(pkg string, verbose bool) error {
	// Extract base package name (remove version spec)
	basePkg := extractJsBasePkg(pkg)
	cmd := exec.Command(b.manager, b.removeSubcommand(), "-g", basePkg)

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s failed: %w", b.manager, b.removeSubcommand(), err)
	}
	return nil
}

// UninstallAll removes every package listed in the bundle file.
func (b *JsBundler) UninstallAll(bundleFile string, verbose bool) error {
	bundleFile = expandPath(bundleFile)
	desired, err := parseFile(bundleFile)
	if err != nil {
		return err
	}
	if len(desired) == 0 {
		printYellow("No packages listed in bundle file.")
		return nil
	}

	installed, err := b.listInstalled()
	if err != nil {
		return fmt.Errorf("failed to list installed packages: %w", err)
	}

	var toRemove []string
	var skipped []string
	for _, pkg := range desired {
		base := extractJsBasePkg(pkg)
		if _, ok := installed[base]; ok {
			toRemove = append(toRemove, base)
		} else {
			skipped = append(skipped, base)
		}
	}

	for _, base := range skipped {
		printYellow(fmt.Sprintf("Skipping (not installed): %s", base))
	}

	if len(toRemove) == 0 {
		printGreen("\nNothing to uninstall.")
		return nil
	}

	printGreen(fmt.Sprintf("Uninstalling %d packages with %s...", len(toRemove), b.manager))
	var failed int
	for _, base := range toRemove {
		printYellow(fmt.Sprintf("Removing: %s", base))
		if err := b.uninstall(base, verbose); err != nil {
			printRed(fmt.Sprintf("Warning: failed to uninstall %s: %v", base, err))
			failed++
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d packages failed to uninstall", failed)
	}
	printGreen("\nDone!")
	return nil
}

// listInstalled returns map of package name -> version
func (b *JsBundler) listInstalled() (map[string]string, error) {
	if b.manager == "npm" {
		return b.listInstalledNpm()
	}
	return b.listInstalledPnpm()
}

func (b *JsBundler) listInstalledPnpm() (map[string]string, error) {
	cmd := exec.Command("pnpm", "list", "-g", "--depth=0")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pnpm list failed: %w", err)
	}

	packages := make(map[string]string)
	lines := strings.Split(string(output), "\n")
	inDeps := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "dependencies:" {
			inDeps = true
			continue
		}
		if !inDeps || line == "" {
			continue
		}
		// Format: "@scope/name version" or "name version"
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			packages[parts[0]] = parts[1]
		} else if len(parts) == 1 {
			packages[parts[0]] = ""
		}
	}
	return packages, nil
}

// listInstalledNpm parses `npm list -g --depth=0 --parseable=false` tree output.
// Lines look like: "├── pkg@1.0.0", "└── @scope/pkg@1.0.0".
func (b *JsBundler) listInstalledNpm() (map[string]string, error) {
	cmd := exec.Command("npm", "list", "-g", "--depth=0")
	output, _ := cmd.Output() // npm exits non-zero on extraneous deps; ignore err

	packages := make(map[string]string)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		// Strip leading tree characters and whitespace
		trimmed := strings.TrimLeft(line, " │├└─")
		trimmed = strings.TrimSpace(trimmed)
		if trimmed == "" {
			continue
		}
		// Skip the first line which is the global path (contains "/" but no leading tree char)
		if strings.HasPrefix(line, "/") || strings.HasPrefix(line, "~") {
			continue
		}
		// Skip annotations like "(empty)"
		if strings.HasPrefix(trimmed, "(") {
			continue
		}
		base := extractJsBasePkg(trimmed)
		version := ""
		if len(trimmed) > len(base)+1 {
			version = trimmed[len(base)+1:]
		}
		packages[base] = version
	}
	return packages, nil
}

// Sync installs/updates/removes packages to match Jsfile.
func (b *JsBundler) Sync(bundleFile string, verbose bool, force bool, noUpdate bool, taps bool, mas bool) error {
	bundleFile = expandPath(bundleFile)

	desired, err := parseFile(bundleFile)
	if err != nil {
		return err
	}

	// Get installed packages (name -> version)
	installed, err := b.listInstalled()
	if err != nil {
		return fmt.Errorf("failed to list installed packages: %w", err)
	}

	// Build desired map (name -> version). Empty version = bare name (no "@"),
	// which means "install once, only upgrade with --upgrade".
	desiredMap := make(map[string]string)
	for _, pkg := range desired {
		base := extractJsBasePkg(pkg)
		version := ""
		if len(pkg) > len(base)+1 {
			version = pkg[len(base)+1:]
		}
		desiredMap[base] = version
	}

	// Remove packages not in desired
	for pkg := range installed {
		if _, want := desiredMap[pkg]; !want {
			printYellow(fmt.Sprintf("Removing: %s", pkg))
			if err := b.uninstall(pkg, verbose); err != nil {
				printRed(fmt.Sprintf("Warning: failed to uninstall %s: %v", pkg, err))
			}
		}
	}

	// Find packages to install (missing or version mismatch)
	var toInstall []string
	for _, pkg := range desired {
		base := extractJsBasePkg(pkg)
		desiredVer := desiredMap[base]
		installedVer, exists := installed[base]

		switch {
		case !exists:
			toInstall = append(toInstall, pkg)
		case desiredVer == "":
			// Bare name — installed, don't touch unless --upgrade or --force
			if upgradeFlag || force {
				toInstall = append(toInstall, pkg)
			}
		case !isLockedVersion(desiredVer):
			// latest, alpha, beta, next, etc. — floating tag, always update
			toInstall = append(toInstall, pkg)
		case installedVer != desiredVer:
			// Locked numeric version mismatch - upgrade/downgrade
			toInstall = append(toInstall, pkg)
		case force:
			toInstall = append(toInstall, pkg)
		}
	}

	if len(toInstall) > 0 {
		if parallelFlag {
			// Parallel installs with TUI progress
			if err := runParallelInstall(b.manager, b.addSubcommand(), toInstall, force, verbose); err != nil {
				return err
			}
		} else {
			// Default: install one at a time so each package is visible
			printGreen(fmt.Sprintf("Installing %d packages with %s...", len(toInstall), b.manager))
			var failed int
			for _, pkg := range toInstall {
				printYellow(fmt.Sprintf("Installing: %s", pkg))
				if err := b.install(pkg, verbose, force); err != nil {
					printRed(fmt.Sprintf("Warning: failed to install %s: %v", pkg, err))
					failed++
				}
			}
			if failed > 0 {
				return fmt.Errorf("%d packages failed to install", failed)
			}
		}
	}

	// --upgrade: bump all global packages to newest versions
	if upgradeFlag {
		printGreen("Upgrading global packages to latest...")
		var cmd *exec.Cmd
		if b.manager == "npm" {
			cmd = exec.Command("npm", "update", "-g")
		} else {
			cmd = exec.Command("pnpm", "update", "-g", "--latest")
		}
		if verbose {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s update failed: %s", b.manager, string(output))
		}
	}

	printGreen("\nDone!")
	return nil
}

// isLockedVersion reports true if the version spec starts with a digit or
// the standard semver prefixes used for exact/range locks (=, v). Floating
// dist-tags like "latest", "alpha", "beta", "next" return false and are
// always re-installed so they track upstream.
func isLockedVersion(v string) bool {
	if v == "" {
		return false
	}
	c := v[0]
	return (c >= '0' && c <= '9') || c == '=' || c == 'v' || c == '~' || c == '^'
}

// extractJsBasePkg extracts base package name from spec.
// @scope/pkg@1.0.0 → @scope/pkg
// pkg@1.0.0 → pkg
func extractJsBasePkg(pkg string) string {
	// Handle scoped packages (@scope/name@version)
	if strings.HasPrefix(pkg, "@") {
		// Find the second @ which is the version separator
		rest := pkg[1:]
		if idx := strings.Index(rest, "@"); idx > 0 {
			return pkg[:idx+1]
		}
		return pkg
	}
	// Regular package
	if idx := strings.Index(pkg, "@"); idx > 0 {
		return pkg[:idx]
	}
	return pkg
}
