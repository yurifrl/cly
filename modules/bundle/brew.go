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

	if dryRun {
		// Show what would be installed
		fmt.Println("Checking missing packages:")
		checkCmd := exec.Command("brew", "bundle", "check", "--file="+bundleFile, "--verbose")
		checkCmd.Stdout = os.Stdout
		checkCmd.Stderr = os.Stderr
		checkCmd.Run() // ignore error, exits non-zero if packages missing

		// Show what would be removed
		fmt.Println("\nPackages to remove (not in Brewfile):")
		cleanupCmd := exec.Command("brew", "bundle", "cleanup", "--file="+bundleFile)
		cleanupCmd.Stdout = os.Stdout
		cleanupCmd.Stderr = os.Stderr
		cleanupCmd.Run()
		return nil
	}

	// Install packages
	installCmd := exec.Command("brew", "bundle", "--file="+bundleFile)
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("brew bundle failed: %w", err)
	}

	// Remove packages not in Brewfile
	fmt.Println("\nCleaning up packages not in Brewfile...")
	cleanupCmd := exec.Command("brew", "bundle", "cleanup", "--file="+bundleFile, "--force")
	cleanupCmd.Stdout = os.Stdout
	cleanupCmd.Stderr = os.Stderr
	cleanupCmd.Run() // don't fail on cleanup errors

	printGreen("\nDone!")
	return nil
}
