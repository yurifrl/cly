package update

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/pkg/style"
)

var (
	checkOnly   bool
	forceUpdate bool
	skipConfirm bool
	targetVer   string
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update cly to the latest version",
		Long: `Check for and install the latest version of cly from GitHub releases.

By default, this command will:
  1. Check for the latest version
  2. Prompt for confirmation
  3. Download and install if a newer version is available

The binary is safely replaced with automatic backup and rollback on failure.`,
		RunE: run,
	}

	cmd.Flags().BoolVar(&checkOnly, "check", false, "Only check for updates without installing")
	cmd.Flags().BoolVar(&forceUpdate, "force", false, "Force reinstall even if on latest version")
	cmd.Flags().BoolVarP(&skipConfirm, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().StringVar(&targetVer, "version", "", "Install specific version (e.g., v0.2.5)")

	parent.AddCommand(cmd)
}

func run(cobraCmd *cobra.Command, args []string) error {
	// Get version from root command
	currentVersion := cobraCmd.Root().Version
	updater := New(currentVersion)

	// Show current version
	fmt.Printf("%s Current version: %s\n",
		style.GreenStyle.Render("✓"),
		updater.currentVer.Raw)

	// Check for updates
	fmt.Printf("%s Checking for updates...\n",
		style.BlueStyle.Render("⚡"))

	release, err := updater.CheckLatest()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w\n%s",
			err,
			style.SubtleStyle.Render("Check your internet connection or try again later"))
	}

	// Parse latest version
	latestVer, err := ParseVersion(release.Version)
	if err != nil {
		return fmt.Errorf("failed to parse latest version: %w", err)
	}

	// Compare versions
	needsUpdate := updater.currentVer.IsOlderThan(latestVer)

	if !needsUpdate && !forceUpdate {
		fmt.Printf("%s Already on latest version\n",
			style.GreenStyle.Render("✅"))
		return nil
	}

	if checkOnly {
		if needsUpdate {
			fmt.Printf("%s New version available: %s\n",
				style.BlueStyle.Render("🎉"),
				release.Version)
			fmt.Printf("\n%s\n",
				style.SubtleStyle.Render("Run 'cly update' to install"))
		} else {
			fmt.Printf("%s Already on latest version\n",
				style.GreenStyle.Render("✅"))
		}
		return nil
	}

	// Show update info
	if forceUpdate {
		fmt.Printf("%s Reinstalling version: %s\n",
			style.YellowStyle.Render("🔄"),
			release.Version)
	} else {
		fmt.Printf("%s New version available: %s\n",
			style.BlueStyle.Render("🎉"),
			release.Version)
	}

	// Confirm update
	if !skipConfirm && !confirmUpdate(release.Version) {
		fmt.Println(style.SubtleStyle.Render("Update cancelled"))
		return nil
	}

	// Find asset for current platform
	osName, arch := detectPlatform()
	asset, found := release.FindAssetForPlatform(osName, arch)
	if !found {
		return fmt.Errorf("no binary available for %s-%s", osName, arch)
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "cly-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// Download
	fmt.Printf("%s Downloading %s...\n",
		style.BlueStyle.Render("⬇️ "),
		asset.Name)

	if err := updater.Download(asset, tmpPath); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Install
	fmt.Printf("%s Installing update...\n",
		style.BlueStyle.Render("🔄"))

	currentBinary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current binary path: %w", err)
	}

	if err := updater.Install(tmpPath, currentBinary); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	fmt.Printf("%s Successfully updated to %s!\n",
		style.GreenStyle.Render("✅"),
		release.Version)

	return nil
}

func confirmUpdate(version string) bool {
	fmt.Printf("\n? Update to %s? (y/N) ", version)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
