package bundle

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// PythonBundler manages Python tools via uv
type PythonBundler struct{}

func (b *PythonBundler) Name() string {
	return "python"
}

func (b *PythonBundler) DefaultFile() string {
	return filepath.Join(os.Getenv("HOME"), ".config", "Pythonfile")
}

func (b *PythonBundler) StateFile() string {
	return filepath.Join(os.Getenv("HOME"), ".config", "python_bundle_state")
}

func (b *PythonBundler) CheckDeps() error {
	if _, err := exec.LookPath("uv"); err != nil {
		return fmt.Errorf("uv not found. Install it: curl -LsSf https://astral.sh/uv/install.sh | sh")
	}
	return nil
}

func (b *PythonBundler) Sync(bundleFile string, dryRun bool) error {
	fmt.Printf("Syncing Python tools from %s\n\n", bundleFile)

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
		printYellow("Uninstalling: %s", pkg)
		cmd := exec.Command("uv", "tool", "uninstall", pkg)
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
		printGreen("Installing: %s", pkg)
		cmd := exec.Command("uv", "tool", "install", pkg)
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
