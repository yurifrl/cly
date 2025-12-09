package bundle

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GoBundler manages Go binaries via go install
type GoBundler struct {
	gobin string
}

func (b *GoBundler) Name() string {
	return "go"
}

func (b *GoBundler) DefaultFile() string {
	return filepath.Join(os.Getenv("HOME"), ".config", "Gofile")
}

func (b *GoBundler) StateFile() string {
	return filepath.Join(os.Getenv("HOME"), ".config", "go_bundle_state")
}

func (b *GoBundler) CheckDeps() error {
	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("go not found. Install Go: https://go.dev/dl/")
	}
	b.setupGobin()
	return nil
}

func (b *GoBundler) setupGobin() {
	// Try mise first
	if out, err := exec.Command("mise", "where", "go").Output(); err == nil {
		miseRoot := strings.TrimSpace(string(out))
		if miseRoot != "" {
			b.gobin = filepath.Join(miseRoot, "bin")
			os.Setenv("GOPATH", miseRoot)
			os.Setenv("GOBIN", b.gobin)
			return
		}
	}

	// Fallback to go env
	if out, err := exec.Command("go", "env", "GOPATH").Output(); err == nil {
		gopath := strings.TrimSpace(string(out))
		if gopath != "" {
			b.gobin = filepath.Join(gopath, "bin")
		}
	}
}

func (b *GoBundler) Sync(bundleFile string, dryRun bool) error {
	fmt.Printf("Syncing Go binaries from %s\n", bundleFile)
	if b.gobin != "" {
		fmt.Printf("Target directory: %s\n", b.gobin)
	}
	fmt.Println()

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
				printGreen("  + %s", pkg)
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
		binaryName := filepath.Base(pkg)
		binaryPath := filepath.Join(b.gobin, binaryName)
		if _, err := os.Stat(binaryPath); err == nil {
			printYellow("Removing: %s", binaryPath)
			if err := os.Remove(binaryPath); err != nil {
				printRed("Failed to remove %s: %v", binaryPath, err)
			} else {
				printGreen("✓ Removed %s", binaryName)
			}
		}
	}

	// Install packages
	var failed []string
	var successful []string

	for _, pkg := range desired {
		printGreen("Installing: %s", pkg)
		cmd := exec.Command("go", "install", pkg+"@latest")
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
