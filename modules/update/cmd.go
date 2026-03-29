package update

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/creativeprojects/go-selfupdate"
	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/style"
)

const (
	defaultSourceDir  = "/Users/yuri/Workdir/Yuri/cly"
	defaultInstallDir = "~/.local/bin"
	binaryName        = "cly"
	githubRepo        = "yurifrl/cly"
)

var (
	remote   bool
	bumpFlag string
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:     "update",
		Aliases: []string{"u"},
		Short: "Build and install cly from source",
		Long: `Update cly by building from local source or downloading from GitHub.

By default, builds from local source directory (configurable via
modules.update.source_dir in config) and installs to /usr/local/bin/cly.

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
		return updateRemote(cmd)
	}
	// Auto-detect: use local source if available, fall back to remote
	sourceDir := getSourceDir()
	if _, err := os.Stat(filepath.Join(sourceDir, "go.mod")); err == nil {
		// Default to patch bump when building locally
		if bumpFlag == "" {
			bumpFlag = "patch"
		}
		return updateLocal(cmd)
	}
	fmt.Printf("%s Source not found at %s, falling back to remote\n",
		style.YellowStyle.Render("⚠️"), sourceDir)
	return updateRemote(cmd)
}

func updateLocal(cmd *cobra.Command) error {
	sourceDir := getSourceDir()
	installDir := expandPath(defaultInstallDir)
	destPath := filepath.Join(installDir, binaryName)

	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return fmt.Errorf("source directory not found: %s\n%s",
			sourceDir,
			style.SubtleStyle.Render("Configure via modules.update.source_dir in config"))
	}

	// Check if caller passed an explicit output path (e.g. bootstrap from task)
	if out := os.Getenv("CLY_INSTALL_DEST"); out != "" {
		destPath = expandPath(out)
		installDir = filepath.Dir(destPath)
	}

	goMod := filepath.Join(sourceDir, "go.mod")
	if _, err := os.Stat(goMod); os.IsNotExist(err) {
		return fmt.Errorf("no go.mod found in %s — not a Go project", sourceDir)
	}

	version := readVersion(sourceDir)

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

	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("failed to create install directory: %w", err)
	}

	ldflags := fmt.Sprintf("-s -w -X github.com/yurifrl/cly/cmd.Version=%s", version)

	fmt.Printf("%s Building cly %s from %s\n",
		style.BlueStyle.Render("⚡"),
		version,
		sourceDir)

	// Build to a temp file, sign it, then atomically move into place
	tmpFile, err := os.CreateTemp("", "cly-build-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	os.Remove(tmpPath) // go build creates it fresh
	buildCmd := exec.Command("go", "build",
		fmt.Sprintf("-ldflags=%s", ldflags),
		"-o", tmpPath,
		".",
	)
	buildCmd.Dir = sourceDir
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("build failed: %w", err)
	}

	// Re-sign before moving so macOS allows spawning child processes
	if out, err := exec.Command("codesign", "--force", "--sign", "-", tmpPath).CombinedOutput(); err != nil {
		fmt.Printf("%s codesign failed (non-fatal): %v — %s\n",
			style.YellowStyle.Render("⚠️"), err, strings.TrimSpace(string(out)))
	}

	if err := copyFileExec(tmpPath, destPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to install binary: %w", err)
	}
	os.Remove(tmpPath)

	fmt.Printf("%s Installed to %s\n",
		style.GreenStyle.Render("✅"),
		destPath)

	installCompletions(destPath)

	return nil
}

func updateRemote(cmd *cobra.Command) error {
	currentVersion := cmd.Root().Version

	fmt.Printf("%s Checking latest release...\n",
		style.BlueStyle.Render("⚡"))

	ctx := context.Background()
	release, found, err := selfupdate.DetectLatest(ctx, selfupdate.NewRepositorySlug("yurifrl", "cly"))
	if err != nil {
		return fmt.Errorf("failed to check releases: %w\n%s",
			err,
			style.SubtleStyle.Render("Check your internet connection or try again later"))
	}
	if !found {
		fmt.Printf("%s Already up to date (%s)\n", style.GreenStyle.Render("✅"), currentVersion)
		return nil
	}
	if !release.GreaterThan(currentVersion) {
		fmt.Printf("%s Already up to date (%s)\n", style.GreenStyle.Render("✅"), currentVersion)
		return nil
	}

	fmt.Printf("%s Updating %s → %s...\n",
		style.BlueStyle.Render("⬇️ "),
		currentVersion,
		release.Version())

	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		return fmt.Errorf("could not locate executable: %w", err)
	}

	if err := selfupdate.DefaultUpdater().UpdateTo(ctx, release, exe); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	fmt.Printf("%s Updated to %s\n", style.GreenStyle.Render("✅"), release.Version())

	installCompletions(exe)

	return nil
}

func getSourceDir() string {
	dir := config.GetString("modules.update.source_dir")
	if dir != "" {
		return expandPath(dir)
	}
	// fall back to old key for backwards compat
	dir = config.GetString("modules.install.source_dir")
	if dir != "" {
		return expandPath(dir)
	}
	return defaultSourceDir
}

// readVersion returns the latest git tag in the source directory (e.g. "1.0.8").
// Strips leading "v" for use in ldflags.
func readVersion(sourceDir string) string {
	out, err := gitOutputIn(sourceDir, "describe", "--tags", "--abbrev=0")
	if err != nil || out == "" {
		return "dev"
	}
	return strings.TrimPrefix(strings.TrimSpace(out), "v")
}

// writeVersion creates an annotated git tag for the new version.
func writeVersion(sourceDir, version string) error {
	tag := "v" + version
	_, err := gitOutputIn(sourceDir, "tag", "-a", tag, "-m", "Release "+tag)
	return err
}

// gitOutputIn runs a git command in dir and returns stdout.
func gitOutputIn(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
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

func installCompletions(clyPath string) {
	if clyPath == "" {
		var err error
		clyPath, err = exec.LookPath(binaryName)
		if err != nil {
			return
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	compCmd := exec.CommandContext(ctx, clyPath, "completion", "fish", "install")
	compCmd.Stdout = os.Stdout
	compCmd.Stderr = os.Stderr
	if err := compCmd.Run(); err != nil {
		fmt.Printf("%s Fish completions failed (non-fatal): %v\n",
			style.YellowStyle.Render("⚠️"),
			err)
	}
}

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

// copyFileExec copies src to dst preserving executable permissions.
// Uses sudo cp if a direct write fails due to permissions.
func copyFileExec(src, dst string) error {
	// Try a direct atomic rename within the same directory first
	dir := filepath.Dir(dst)
	tmp, err := os.CreateTemp(dir, ".cly-install-*")
	if err == nil {
		tmpPath := tmp.Name()
		in, err2 := os.Open(src)
		if err2 == nil {
			_, err2 = io.Copy(tmp, in)
			in.Close()
		}
		tmp.Close()
		if err2 == nil {
			if err2 = os.Chmod(tmpPath, 0755); err2 == nil {
				if err2 = os.Rename(tmpPath, dst); err2 == nil {
					return nil
				}
			}
		}
		os.Remove(tmpPath)
	}

	// Fall back to sudo cp
	if out, err := exec.Command("sudo", "cp", src, dst).CombinedOutput(); err != nil {
		return fmt.Errorf("sudo cp failed: %w — %s", err, strings.TrimSpace(string(out)))
	}
	if out, err := exec.Command("sudo", "chmod", "755", dst).CombinedOutput(); err != nil {
		return fmt.Errorf("sudo chmod failed: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
