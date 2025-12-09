package bundle

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
)

// JsBundler manages JavaScript packages via bun
type JsBundler struct{}

func (b *JsBundler) Name() string {
	return "js"
}

func (b *JsBundler) DefaultFile() string {
	return filepath.Join(os.Getenv("HOME"), ".config", "Jsfile")
}

func (b *JsBundler) StateFile() string {
	return filepath.Join(os.Getenv("HOME"), ".config", "js_bundle_state")
}

func (b *JsBundler) CheckDeps() error {
	if _, err := exec.LookPath("bun"); err != nil {
		return fmt.Errorf("bun not found. Install it: curl -fsSL https://bun.sh/install | bash")
	}
	return nil
}

// normalizePackage converts GitHub shorthand to bun format
// user/repo -> github:user/repo
// user/repo@tag -> github:user/repo#tag
// @scope/pkg stays as-is
func normalizePackage(pkg string) string {
	// GitHub shorthand: user/repo (not starting with @, has exactly one slash)
	re := regexp.MustCompile(`^([^@][^/]+)/([^@]+)(@.*)?$`)
	if matches := re.FindStringSubmatch(pkg); matches != nil {
		user := matches[1]
		repo := matches[2]
		version := matches[3]
		if version != "" {
			// Convert @tag to #tag
			version = "#" + version[1:]
		}
		return fmt.Sprintf("github:%s/%s%s", user, repo, version)
	}
	return pkg
}

func (b *JsBundler) Sync(bundleFile string, dryRun bool) error {
	fmt.Printf("Syncing JavaScript tools from %s\n\n", bundleFile)

	desired, err := ParseBundleFile(bundleFile)
	if err != nil {
		return fmt.Errorf("failed to parse bundle file: %w", err)
	}

	installed, err := LoadState(b.StateFile())
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	toInstall, toRemove := DiffPackages(desired, installed)

	if dryRun {
		if len(toInstall) > 0 {
			fmt.Println("Would install:")
			for _, pkg := range toInstall {
				normalized := normalizePackage(pkg)
				if normalized != pkg {
					printGreen("  + %s → %s", pkg, normalized)
				} else {
					printGreen("  + %s", pkg)
				}
			}
		}
		if len(toRemove) > 0 {
			fmt.Println("Would remove:")
			for _, pkg := range toRemove {
				printYellow("  - %s", pkg)
			}
		}
		if len(toInstall) == 0 && len(toRemove) == 0 {
			fmt.Println("Nothing to do")
		}
		return nil
	}

	// Remove packages no longer in file
	for _, pkg := range toRemove {
		printYellow("Uninstalling: %s", pkg)
		cmd := exec.Command("bun", "remove", "-g", pkg)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			printRed("✗ Failed to uninstall %s", pkg)
		} else {
			printGreen("✓ Uninstalled %s", pkg)
		}
	}

	// Install packages
	var failed []string
	var successful []string

	for _, pkg := range desired {
		normalized := normalizePackage(pkg)
		printGreen("Installing: %s", pkg)
		if normalized != pkg {
			printYellow("  → Resolved to: %s", normalized)
		}

		cmd := exec.Command("bun", "install", "-g", normalized)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			printRed("✗ Failed to install %s", pkg)
			failed = append(failed, pkg)
		} else {
			printGreen("✓ Installed %s", pkg)
			successful = append(successful, pkg)
		}
	}

	// Save state
	if err := SaveState(b.StateFile(), successful); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	fmt.Println()
	printGreen("Done!")

	if len(failed) > 0 {
		printRed("Failed installations:")
		for _, pkg := range failed {
			fmt.Printf("  %s\n", pkg)
		}
		return fmt.Errorf("%d packages failed to install", len(failed))
	}

	return nil
}
