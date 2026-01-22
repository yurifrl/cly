package bundle

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	pkgconfig "github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/store"
)

// JsBundler manages JavaScript packages via pnpm.
type JsBundler struct {
	*baseBundler
}

// NewJsBundler creates a new JsBundler.
func NewJsBundler(s store.Store) *JsBundler {
	b := &JsBundler{}
	jsFile := pkgconfig.GetString("modules.bundle.js_file")
	if jsFile == "" {
		jsFile = "~/.config/cly/bundles/Jsfile"
	}
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
	if !commandExists("pnpm") {
		return fmt.Errorf("pnpm not found. Add to Brewfile: 'pnpm', then run: cly bundle brew")
	}
	// Ensure PNPM_HOME is set and in PATH for global installs
	pnpmHome := os.Getenv("PNPM_HOME")
	if pnpmHome == "" {
		home, _ := os.UserHomeDir()
		pnpmHome = home + "/Library/pnpm"
		os.Setenv("PNPM_HOME", pnpmHome)
	}
	path := os.Getenv("PATH")
	if !strings.Contains(path, pnpmHome) {
		os.Setenv("PATH", pnpmHome+":"+path)
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

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pnpm add failed: %s", string(output))
	}
	return nil
}

func (b *JsBundler) uninstall(pkg string, verbose bool) error {
	// Extract base package name (remove version spec)
	basePkg := extractJsBasePkg(pkg)
	cmd := exec.Command("pnpm", "remove", "-g", basePkg)

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pnpm remove failed: %w", err)
	}
	return nil
}

// listInstalled returns map of package name -> version
func (b *JsBundler) listInstalled() (map[string]string, error) {
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

	// Build desired map (name -> version)
	desiredMap := make(map[string]string)
	for _, pkg := range desired {
		base := extractJsBasePkg(pkg)
		version := "latest"
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

	// Find packages to install/upgrade
	var toInstall []string
	for _, pkg := range desired {
		base := extractJsBasePkg(pkg)
		desiredVer := desiredMap[base]
		installedVer, exists := installed[base]

		if !exists {
			toInstall = append(toInstall, pkg)
		} else if desiredVer != "latest" && installedVer != desiredVer {
			// Version mismatch - upgrade/downgrade
			toInstall = append(toInstall, pkg)
		} else if force {
			toInstall = append(toInstall, pkg)
		}
	}

	if len(toInstall) > 0 {
		printGreen(fmt.Sprintf("Installing %d packages...", len(toInstall)))
		args := append([]string{"add", "-g"}, toInstall...)
		cmd := exec.Command("pnpm", args...)
		if verbose {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}

		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("pnpm add failed: %s", string(output))
		}
	}

	printGreen("\nDone!")
	return nil
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
