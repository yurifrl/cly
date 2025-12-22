package bundle

import (
	"fmt"
	"os"
	"os/exec"

	pkgconfig "github.com/yurifrl/cly/pkg/config"
)

// BrewBundler wraps brew bundle command.
type BrewBundler struct{}

// NewBrewBundler creates a new BrewBundler.
func NewBrewBundler() *BrewBundler {
	return &BrewBundler{}
}

func (b *BrewBundler) Name() string {
	return "brew"
}

func (b *BrewBundler) DefaultFile() string {
	brewFile := pkgconfig.GetString("bundle.brew_file")
	if brewFile == "" {
		brewFile = "~/.config/Brewfile"
	}
	return brewFile
}

func (b *BrewBundler) CheckDeps() error {
	if !commandExists("brew") {
		return fmt.Errorf("brew not found. Install Homebrew: https://brew.sh")
	}
	return nil
}

func (b *BrewBundler) Sync(bundleFile string, verbose bool, force bool, taps bool) error {
	bundleFile = expandPath(bundleFile)

	// Only install taps if --taps flag is passed
	if taps {
		tapsFile := bundleFile + ".taps"
		if _, err := os.Stat(tapsFile); err == nil {
			fmt.Printf("Syncing taps from %s\n\n", tapsFile)

			args := []string{"bundle", "--file=" + tapsFile}
			if verbose {
				args = append(args, "--verbose")
			}
			if force {
				args = append(args, "--force")
			}

			cmd := exec.Command("brew", args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				fmt.Printf("Warning: brew bundle taps had errors: %v\n", err)
			}
			fmt.Println()
		}
	}

	fmt.Printf("Syncing Homebrew packages from %s\n\n", bundleFile)

	args := []string{"bundle", "--file=" + bundleFile}
	if verbose {
		args = append(args, "--verbose")
	}
	if force {
		args = append(args, "--force")
	}

	cmd := exec.Command("brew", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("brew bundle failed: %w", err)
	}

	printGreen("\nDone!")
	return nil
}

func (b *BrewBundler) Check(bundleFile string) error {
	bundleFile = expandPath(bundleFile)

	// Check taps file if it exists
	tapsFile := bundleFile + ".taps"
	if _, err := os.Stat(tapsFile); err == nil {
		fmt.Printf("Checking taps from %s\n\n", tapsFile)

		cmd := exec.Command("brew", "bundle", "check", "--file="+tapsFile)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("taps changes needed")
		}
		fmt.Println()
	}

	fmt.Printf("Checking Homebrew packages from %s\n\n", bundleFile)

	cmd := exec.Command("brew", "bundle", "check", "--file="+bundleFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// brew bundle check exits non-zero if packages are missing
		return fmt.Errorf("changes needed")
	}

	printGreen("Everything is in sync")
	return nil
}

func (b *BrewBundler) Cleanup(bundleFile string, verbose bool, force bool) error {
	bundleFile = expandPath(bundleFile)
	fmt.Printf("Cleaning up Homebrew packages not in %s\n\n", bundleFile)

	args := []string{"bundle", "cleanup", "--file=" + bundleFile}
	if force {
		args = append(args, "--force")
	}
	if verbose {
		args = append(args, "--verbose")
	}

	cmd := exec.Command("brew", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("brew bundle cleanup failed: %w", err)
	}

	printGreen("\nCleanup done!")
	return nil
}
