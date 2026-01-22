package bundle

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

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
	brewFile := pkgconfig.GetString("modules.bundle.brew_file")
	if brewFile == "" {
		brewFile = "~/.config/cly/bundles/Brewfile"
	}
	return brewFile
}

func (b *BrewBundler) CheckDeps() error {
	if !commandExists("brew") {
		return fmt.Errorf("brew not found. Install Homebrew: https://brew.sh")
	}
	return nil
}

func (b *BrewBundler) Sync(bundleFile string, verbose bool, force bool, noUpdate bool, taps bool, mas bool) error {
	bundleFile = expandPath(bundleFile)

	// Only extract and sync taps if --taps flag is passed
	if taps {
		tapLines, err := extractTaps(bundleFile)
		if err != nil {
			return fmt.Errorf("failed to read Brewfile: %w", err)
		}

		if len(tapLines) > 0 {
			fmt.Printf("Syncing %d tap(s) from %s\n\n", len(tapLines), bundleFile)

			tapsFile, err := writeTapsToTempFile(tapLines)
			if err != nil {
				return fmt.Errorf("failed to create temp taps file: %w", err)
			}
			defer os.Remove(tapsFile)

			args := []string{"bundle", "--file=" + tapsFile}
			if verbose {
				args = append(args, "--verbose")
			}

			fmt.Printf("$ brew %s\n", strings.Join(args, " "))
			cmd := exec.Command("brew", args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				fmt.Printf("Warning: brew bundle taps had errors: %v\n", err)
			}
			fmt.Println()
		}
	}

	// Determine which file to use for bundle
	effectiveFile := bundleFile
	if !mas {
		// Filter out mas lines and create temp file
		content, err := filterMasLines(bundleFile)
		if err != nil {
			return fmt.Errorf("failed to filter mas lines: %w", err)
		}

		tmpFile, err := os.CreateTemp("", "brewfile-no-mas-*.rb")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.WriteString(content); err != nil {
			return fmt.Errorf("failed to write temp file: %w", err)
		}
		tmpFile.Close()

		effectiveFile = tmpFile.Name()
		fmt.Printf("Syncing Homebrew packages from %s (skipping mas)\n\n", bundleFile)
	} else {
		fmt.Printf("Syncing Homebrew packages from %s\n\n", bundleFile)
	}

	args := []string{"bundle", "--file=" + effectiveFile}
	if verbose {
		args = append(args, "--verbose")
	}
	if force {
		args = append(args, "--force")
	}
	if !noUpdate {
		args = append(args, "--upgrade")
	}
	args = append(args, "--cleanup")

	fmt.Printf("$ brew %s\n", strings.Join(args, " "))
	cmd := exec.Command("brew", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("brew bundle failed: %w", err)
	}

	return nil
}

func (b *BrewBundler) Check(bundleFile string) error {
	bundleFile = expandPath(bundleFile)

	fmt.Printf("Checking Homebrew packages from %s\n\n", bundleFile)

	args := []string{"bundle", "check", "--file=" + bundleFile}
	fmt.Printf("$ brew %s\n", strings.Join(args, " "))
	cmd := exec.Command("brew", args...)
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

	args := []string{"bundle", "cleanup", "--file=" + bundleFile}
	if force {
		args = append(args, "--force")
	}
	if verbose {
		args = append(args, "--verbose")
	}

	fmt.Printf("$ brew %s\n", strings.Join(args, " "))
	cmd := exec.Command("brew", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("brew bundle cleanup failed: %w", err)
	}

	return nil
}

// extractTaps reads a Brewfile and returns all tap lines.
func extractTaps(brewfile string) ([]string, error) {
	file, err := os.Open(brewfile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var taps []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "tap ") {
			taps = append(taps, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return taps, nil
}

// writeTapsToTempFile writes tap lines to a temporary file and returns its path.
func writeTapsToTempFile(taps []string) (string, error) {
	tmpFile, err := os.CreateTemp("", "brewfile-taps-*.rb")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	for _, tap := range taps {
		if _, err := tmpFile.WriteString(tap + "\n"); err != nil {
			os.Remove(tmpFile.Name())
			return "", err
		}
	}

	return tmpFile.Name(), nil
}

// filterMasLines reads a Brewfile and returns content without mas lines.
func filterMasLines(brewfile string) (string, error) {
	file, err := os.Open(brewfile)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "mas ") {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return strings.Join(lines, "\n"), nil
}
