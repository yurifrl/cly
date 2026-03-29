package update

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
	localVersionFile  = ".local-version"
)

var (
	remote   bool
	bumpFlag string
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:     "update",
		Aliases: []string{"u"},
		Short:   "Build and install cly from source",
		Long: `Update cly by building from local source or downloading from GitHub.

By default, builds from local source, increments the local alpha counter
(.local-version), and installs to /usr/local/bin/cly.

Use --bump to cut a real release tag (patch/minor/major) and push it to
GitHub, which triggers CI to build and publish a GitHub release.

Use --remote to download the latest GitHub release instead.`,
		RunE: run,
	}

	cmd.Flags().BoolVar(&remote, "remote", false, "Download latest release from GitHub instead of building locally")
	cmd.Flags().StringVarP(&bumpFlag, "bump", "b", "", "Cut a real release tag: patch, minor, or major")
	cmd.Flags().Lookup("bump").NoOptDefVal = "patch"

	_ = cmd.RegisterFlagCompletionFunc("bump", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{
			"patch\tBump patch version (1.0.5 → 1.0.6)",
			"minor\tBump minor version (1.0.5 → 1.1.0)",
			"major\tBump major version (1.0.5 → 2.0.0)",
		}, cobra.ShellCompDirectiveNoFileComp
	})

	parent.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	if remote {
		return updateRemote(cmd)
	}
	sourceDir := getSourceDir()
	if _, err := os.Stat(filepath.Join(sourceDir, "go.mod")); err == nil {
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

	if out := os.Getenv("CLY_INSTALL_DEST"); out != "" {
		destPath = expandPath(out)
		installDir = filepath.Dir(destPath)
	}

	if _, err := os.Stat(filepath.Join(sourceDir, "go.mod")); os.IsNotExist(err) {
		return fmt.Errorf("no go.mod found in %s — not a Go project", sourceDir)
	}

	var version string

	if bumpFlag != "" {
		// Cut a real release: tag + push, reset local-version
		base, err := latestGitTag(sourceDir)
		if err != nil {
			base = "0.0.0"
		}
		newBase, err := bumpSemver(base, bumpFlag)
		if err != nil {
			return err
		}
		if err := createGitTag(sourceDir, newBase); err != nil {
			return err
		}
		if err := pushGitTag(sourceDir, "v"+newBase); err != nil {
			return fmt.Errorf("tagged locally but push failed: %w\nRun: git push origin v%s", err, newBase)
		}
		if err := writeLocalVersion(sourceDir, newBase+"-alpha.1"); err != nil {
			return err
		}
		fmt.Printf("%s Released %s → v%s (tag pushed, CI will build)\n",
			style.GreenStyle.Render("🚀"), base, newBase)
		version = newBase
	} else {
		// Normal update: increment alpha counter
		version = nextAlphaVersion(sourceDir)
		if err := writeLocalVersion(sourceDir, version); err != nil {
			return err
		}
		fmt.Printf("%s Local version: %s\n",
			style.BlueStyle.Render("🏷️ "), version)
	}

	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("failed to create install directory: %w", err)
	}

	ldflags := fmt.Sprintf("-s -w -X github.com/yurifrl/cly/cmd.Version=%s", version)

	fmt.Printf("%s Building cly %s from %s\n",
		style.BlueStyle.Render("⚡"), version, sourceDir)

	tmpFile, err := os.CreateTemp("", "cly-build-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	os.Remove(tmpPath)

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

	if out, err := exec.Command("codesign", "--force", "--sign", "-", tmpPath).CombinedOutput(); err != nil {
		fmt.Printf("%s codesign failed (non-fatal): %v — %s\n",
			style.YellowStyle.Render("⚠️"), err, strings.TrimSpace(string(out)))
	}

	if err := copyFileExec(tmpPath, destPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to install binary: %w", err)
	}
	os.Remove(tmpPath)

	fmt.Printf("%s Installed to %s\n", style.GreenStyle.Render("✅"), destPath)

	installCompletions(destPath)

	return nil
}

func updateRemote(cmd *cobra.Command) error {
	currentVersion := cmd.Root().Version

	fmt.Printf("%s Checking latest release...\n", style.BlueStyle.Render("⚡"))

	ctx := context.Background()
	release, found, err := selfupdate.DetectLatest(ctx, selfupdate.NewRepositorySlug("yurifrl", "cly"))
	if err != nil {
		return fmt.Errorf("failed to check releases: %w\n%s",
			err, style.SubtleStyle.Render("Check your internet connection or try again later"))
	}
	if !found || !release.GreaterThan(currentVersion) {
		fmt.Printf("%s Already up to date (%s)\n", style.GreenStyle.Render("✅"), currentVersion)
		return nil
	}

	fmt.Printf("%s Updating %s → %s...\n",
		style.BlueStyle.Render("⬇️ "), currentVersion, release.Version())

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

// nextAlphaVersion reads .local-version, syncs with latest git tag,
// increments the alpha counter, and returns the new version string.
func nextAlphaVersion(sourceDir string) string {
	gitTag, _ := latestGitTag(sourceDir)
	local := readLocalVersion(sourceDir)

	// Parse local: "1.0.11-alpha.3" → base="1.0.11", n=3
	base, n := parseAlpha(local)

	// If no base at all, start from 0.0.1
	if base == "" {
		base = "0.0.1"
	}

	// If git tag is newer than our local base, reset to that tag
	if gitTag != "" && isGreater(gitTag, base) {
		base = gitTag
		n = 1
	} else {
		n++
	}

	return fmt.Sprintf("%s-alpha.%d", base, n)
}

// parseAlpha splits "1.0.11-alpha.3" into ("1.0.11", 3).
// Falls back to (input, 0) if not in alpha format.
func parseAlpha(v string) (base string, n int) {
	idx := strings.Index(v, "-alpha.")
	if idx == -1 {
		return v, 0
	}
	base = v[:idx]
	n, _ = strconv.Atoi(v[idx+7:])
	return base, n
}

// readLocalVersion reads .local-version or returns empty string.
func readLocalVersion(sourceDir string) string {
	data, err := os.ReadFile(filepath.Join(sourceDir, localVersionFile))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// writeLocalVersion writes .local-version.
func writeLocalVersion(sourceDir, version string) error {
	return os.WriteFile(filepath.Join(sourceDir, localVersionFile), []byte(version+"\n"), 0644)
}

// latestGitTag returns the latest semver tag without the leading "v".
func latestGitTag(sourceDir string) (string, error) {
	out, err := gitOutputIn(sourceDir, "describe", "--tags", "--abbrev=0")
	if err != nil || out == "" {
		return "", err
	}
	return strings.TrimPrefix(out, "v"), nil
}

// createGitTag creates an annotated git tag for the version.
func createGitTag(sourceDir, version string) error {
	tag := "v" + version
	_, err := gitOutputIn(sourceDir, "tag", "-a", tag, "-m", "Release "+tag)
	if err != nil {
		return fmt.Errorf("git tag failed: %w", err)
	}
	fmt.Printf("%s Created tag %s\n", style.BlueStyle.Render("🏷️ "), tag)
	return nil
}

// pushGitTag pushes a tag to origin.
func pushGitTag(sourceDir, tag string) error {
	_, err := gitOutputIn(sourceDir, "push", "origin", tag)
	return err
}

// isGreater returns true if a > b (simple semver, no pre-release).
func isGreater(a, b string) bool {
	pa := parseSemver(a)
	pb := parseSemver(b)
	for i := range pa {
		if i >= len(pb) {
			return true
		}
		if pa[i] > pb[i] {
			return true
		}
		if pa[i] < pb[i] {
			return false
		}
	}
	return false
}

func parseSemver(v string) []int {
	// Strip leading v and any pre-release suffix
	v = strings.TrimPrefix(v, "v")
	v = strings.SplitN(v, "-", 2)[0]
	parts := strings.Split(v, ".")
	nums := make([]int, len(parts))
	for i, p := range parts {
		nums[i], _ = strconv.Atoi(p)
	}
	return nums
}

func bumpSemver(current, level string) (string, error) {
	v := strings.TrimPrefix(current, "v")
	v = strings.SplitN(v, "-", 2)[0] // strip any pre-release
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid version %q, expected X.Y.Z", current)
	}
	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])
	patch, _ := strconv.Atoi(parts[2])

	switch level {
	case "major":
		major++
		minor, patch = 0, 0
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

func gitOutputIn(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func getSourceDir() string {
	if dir := config.GetString("modules.update.source_dir"); dir != "" {
		return expandPath(dir)
	}
	if dir := config.GetString("modules.install.source_dir"); dir != "" {
		return expandPath(dir)
	}
	return defaultSourceDir
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
			style.YellowStyle.Render("⚠️"), err)
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

func copyFileExec(src, dst string) error {
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
	// fallback: sudo cp for protected dirs
	if out, err := exec.Command("sudo", "cp", src, dst).CombinedOutput(); err != nil {
		return fmt.Errorf("sudo cp failed: %w — %s", err, strings.TrimSpace(string(out)))
	}
	if out, err := exec.Command("sudo", "chmod", "755", dst).CombinedOutput(); err != nil {
		return fmt.Errorf("sudo chmod failed: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
