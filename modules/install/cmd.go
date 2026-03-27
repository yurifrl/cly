package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/modules/update"
	"github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/style"
)

const (
	defaultSourceDir  = "/Users/yuri/Workdir/Yuri/cly"
	defaultInstallDir = "~/.local/bin"
	binaryName        = "cly"
)

var (
	remote   bool
	bumpFlag string
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Build and install cly from source",
		Long: `Install cly by building from local source or downloading from GitHub.

By default, builds from local source directory (configurable via
modules.install.source_dir in config) and installs to ~/.local/bin/cly.

Use --remote to download the latest release from GitHub instead.`,
		RunE: run,
	}

	cmd.Flags().BoolVar(&remote, "remote", false, "Install from GitHub release instead of local source")
	cmd.Flags().StringVarP(&bumpFlag, "bump", "b", "", "Bump version before building (patch, minor, major). Default: patch if flag given without value")
	cmd.Flags().Lookup("bump").NoOptDefVal = "patch"

	_ = cmd.RegisterFlagCompletionFunc("bump", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"patch\tBump patch version (1.0.5 → 1.0.6)", "minor\tBump minor version (1.0.5 → 1.1.0)", "major\tBump major version (1.0.5 → 2.0.0)"}, cobra.ShellCompDirectiveNoFileComp
	})

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	if remote {
		return installRemote(cmd)
	}
	return installLocal(cmd)
}

// installLocal builds from local source and installs
func installLocal(cmd *cobra.Command) error {
	sourceDir := getSourceDir()
	installDir := expandPath(defaultInstallDir)
	destPath := filepath.Join(installDir, binaryName)

	// Verify source directory exists
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return fmt.Errorf("source directory not found: %s\n%s",
			sourceDir,
			style.SubtleStyle.Render("Configure via modules.install.source_dir in config"))
	}

	// Verify it's a Go project
	goMod := filepath.Join(sourceDir, "go.mod")
	if _, err := os.Stat(goMod); os.IsNotExist(err) {
		return fmt.Errorf("no go.mod found in %s — not a Go project", sourceDir)
	}

	// Read VERSION file
	version := readVersion(sourceDir)

	// Bump version if requested
	if bumpFlag != "" {
		newVersion, err := bumpVersion(version, bumpFlag)
		if err != nil {
			return err
		}
		if err := writeVersion(sourceDir, newVersion); err != nil {
			return err
		}
		fmt.Printf("%s Version bumped: %s → %s\n",
			style.BlueStyle.Render("🏷️ "),
			version,
			newVersion)
		version = newVersion
	}

	// Ensure install directory exists
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("failed to create install directory: %w", err)
	}

	// Build ldflags
	ldflags := fmt.Sprintf("-s -w -X github.com/yurifrl/cly/cmd.Version=%s", version)

	fmt.Printf("%s Building cly %s from %s\n",
		style.BlueStyle.Render("⚡"),
		version,
		sourceDir)

	// Run go build
	buildCmd := exec.Command("go", "build",
		fmt.Sprintf("-ldflags=%s", ldflags),
		"-o", destPath,
		".",
	)
	buildCmd.Dir = sourceDir
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	fmt.Printf("%s Installed to %s\n",
		style.GreenStyle.Render("✅"),
		destPath)

	// Install fish completions
	installCompletions()

	return nil
}

// installRemote downloads latest release from GitHub
func installRemote(cmd *cobra.Command) error {
	currentVersion := cmd.Root().Version
	updater := update.New(currentVersion)

	fmt.Printf("%s Checking latest release...\n",
		style.BlueStyle.Render("⚡"))

	release, err := updater.CheckLatest()
	if err != nil {
		return fmt.Errorf("failed to check releases: %w\n%s",
			err,
			style.SubtleStyle.Render("Check your internet connection or try again later"))
	}

	fmt.Printf("%s Found version %s\n",
		style.BlueStyle.Render("📦"),
		release.Version)

	// Find asset for current platform
	osName, arch := update.DetectPlatform()
	asset, found := release.FindAssetForPlatform(osName, arch)
	if !found {
		return fmt.Errorf("no binary available for %s-%s", osName, arch)
	}

	// Download to temp file
	tmpFile, err := os.CreateTemp("", "cly-install-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	fmt.Printf("%s Downloading %s...\n",
		style.BlueStyle.Render("⬇️ "),
		asset.Name)

	if err := updater.Download(asset, tmpPath); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Install to destination
	installDir := expandPath(defaultInstallDir)
	destPath := filepath.Join(installDir, binaryName)

	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("failed to create install directory: %w", err)
	}

	fmt.Printf("%s Installing to %s...\n",
		style.BlueStyle.Render("🔄"),
		destPath)

	if err := updater.Install(tmpPath, destPath); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	fmt.Printf("%s Installed cly %s to %s\n",
		style.GreenStyle.Render("✅"),
		release.Version,
		destPath)

	// Install fish completions
	installCompletions()

	return nil
}

// getSourceDir returns the configured source directory
func getSourceDir() string {
	dir := config.GetString("modules.install.source_dir")
	if dir != "" {
		return expandPath(dir)
	}
	return defaultSourceDir
}

// readVersion reads the VERSION file from the source directory
func readVersion(sourceDir string) string {
	versionFile := filepath.Join(sourceDir, "VERSION")
	data, err := os.ReadFile(versionFile)
	if err != nil {
		return "dev"
	}
	return strings.TrimSpace(string(data))
}

func writeVersion(sourceDir, version string) error {
	versionFile := filepath.Join(sourceDir, "VERSION")
	return os.WriteFile(versionFile, []byte(version+"\n"), 0644)
}

func bumpVersion(current, level string) (string, error) {
	v := strings.TrimPrefix(current, "v")
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid version format %q, expected X.Y.Z", current)
	}

	major, minor, patch := 0, 0, 0
	if _, err := fmt.Sscanf(parts[0], "%d", &major); err != nil {
		return "", fmt.Errorf("invalid major version: %s", parts[0])
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &minor); err != nil {
		return "", fmt.Errorf("invalid minor version: %s", parts[1])
	}
	if _, err := fmt.Sscanf(parts[2], "%d", &patch); err != nil {
		return "", fmt.Errorf("invalid patch version: %s", parts[2])
	}

	switch level {
	case "major":
		major++
		minor = 0
		patch = 0
	case "minor":
		minor++
		patch = 0
	case "patch":
		patch++
	default:
		return "", fmt.Errorf("invalid bump level %q, use: patch, minor, or major", level)
	}

	return fmt.Sprintf("%d.%d.%d", major, minor, patch), nil
}

// installCompletions runs fish completion install
func installCompletions() {
	clyPath, err := exec.LookPath(binaryName)
	if err != nil {
		return
	}

	compCmd := exec.Command(clyPath, "completion", "fish", "install")
	compCmd.Stdout = os.Stdout
	compCmd.Stderr = os.Stderr
	if err := compCmd.Run(); err != nil {
		fmt.Printf("%s Fish completions failed (non-fatal): %v\n",
			style.YellowStyle.Render("⚠️"),
			err)
	}
}

// expandPath expands ~ to home directory
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
