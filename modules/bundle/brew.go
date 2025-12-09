package bundle

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// BrewBundler manages Homebrew packages via brew bundle
type BrewBundler struct{}

func (b *BrewBundler) Name() string {
	return "brew"
}

func (b *BrewBundler) DefaultFile() string {
	return filepath.Join(os.Getenv("HOME"), ".config", "Brewfile")
}

func (b *BrewBundler) StateFile() string {
	return "" // brew bundle manages its own state
}

func (b *BrewBundler) CheckDeps() error {
	if _, err := exec.LookPath("brew"); err != nil {
		return fmt.Errorf("brew not found. Install Homebrew: https://brew.sh")
	}
	return nil
}

func (b *BrewBundler) Sync(bundleFile string, dryRun bool) error {
	fmt.Printf("Syncing Homebrew packages from %s\n\n", bundleFile)

	args := []string{"bundle", "--file=" + bundleFile}
	if dryRun {
		args = append(args, "--no-lock")
		// For dry-run, use check to see what would change
		args = []string{"bundle", "check", "--file=" + bundleFile}
	}

	cmd := exec.Command("brew", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if dryRun {
			// brew bundle check exits non-zero if packages are missing
			fmt.Println("\nPackages need to be installed (run without --dry-run)")
			return nil
		}
		return fmt.Errorf("brew bundle failed: %w", err)
	}

	printGreen("\nDone!")
	return nil
}
