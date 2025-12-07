package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	"github.com/NSXBet/nsx-cli/shared/interact"
)

var (
	forceUpdate      bool
	specifiedVersion string
)

func init() {
	updateCmd.Flags().BoolVarP(&forceUpdate, "force", "f", false, "Force update even if already on latest version")
	updateCmd.Flags().
		StringVarP(&specifiedVersion, "version", "v", "", "Specify a version to update to (default: latest)")
	RootCmd.AddCommand(updateCmd)
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update NSX CLI to the latest version",
	Long: `Update NSX CLI to the latest version or a specified version.
This command checks for a newer version of NSX CLI and updates your installation.`,
	RunE: runUpdate,
}

func runUpdate(cmd *cobra.Command, args []string) error {
	interact.Info("Checking for updates...")

	targetVersion := specifiedVersion
	if targetVersion == "" {
		// Get latest version from GitHub
		latestVersion, err := getLatestVersion()
		if err != nil {
			return fmt.Errorf("failed to check for latest version: %w", err)
		}
		targetVersion = latestVersion
	}

	currentVersion := version
	if currentVersion == "dev" {
		interact.Warn("Running a development build, cannot determine current version")
		if !forceUpdate {
			interact.Info("Use --force to update anyway")
			return nil
		}
	} else if !forceUpdate && isSameVersion(currentVersion, targetVersion) {
		interact.Success("Already running the latest version: %s", currentVersion)
		return nil
	}

	interact.Info("Updating to version %s", targetVersion)
	if err := performUpdate(targetVersion); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	interact.Success("Successfully updated to version %s", targetVersion)
	return nil
}

func performUpdate(version string) error {
	tmpDir, err := os.MkdirTemp("", "nsx-update")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			interact.Error("Failed to remove temp directory: %v", err)
		}
	}()

	osName := runtime.GOOS
	arch := runtime.GOARCH

	downloadOS := osName
	switch osName {
	case "darwin":
		downloadOS = "Darwin"
	case "linux":
		downloadOS = "Linux"
	case "windows":
		downloadOS = "Windows"
	}

	downloadArch := arch
	switch arch {
	case "amd64":
		downloadArch = "x86_64"
	case "386":
		downloadArch = "i386"
	}

	extension := "tar.gz"
	if osName == "windows" {
		extension = "zip"
	}

	fileName := fmt.Sprintf("nsx_%s_%s.%s", downloadOS, downloadArch, extension)
	downloadURL := fmt.Sprintf("%s/releases/download/v%s/%s", getProxyURL(), version, fileName)

	interact.Info("Downloading from %s", downloadURL)

	filePath := filepath.Join(tmpDir, fileName)
	if err := downloadFile(downloadURL, filePath); err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}

	binaryName := "nsx"
	if osName == "windows" {
		binaryName = "nsx.exe"
	}

	if extension == "tar.gz" {
		// Extract tar.gz
		cmd := exec.Command("tar", "-xzf", filePath, "-C", tmpDir, binaryName)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to extract update: %w", err)
		}
	} else {
		// For Windows, implement zip extraction logic
		return fmt.Errorf("windows update not implemented in this version")
	}

	// Get path to current executable
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to determine executable path: %w", err)
	}

	// Create backup of current executable
	backupPath := execPath + ".bak"
	if err := copyFile(execPath, backupPath); err != nil {
		interact.Warn("Failed to create backup: %v", err)
	}

	// Replace current executable with new one
	extractedBinary := filepath.Join(tmpDir, binaryName)

	// Try to replace directly first
	err = copyFile(extractedBinary, execPath)
	if err != nil && os.IsPermission(err) {
		// If permission denied, try with sudo
		interact.Info("Insufficient permissions. Attempting with sudo...")

		sudoCmd := exec.Command("sudo", "cp", extractedBinary, execPath)
		sudoCmd.Stdin = os.Stdin
		sudoCmd.Stdout = os.Stdout
		sudoCmd.Stderr = os.Stderr

		if err := sudoCmd.Run(); err != nil {
			return fmt.Errorf("failed to install update with sudo: %w", err)
		}

		// Make executable
		chmodCmd := exec.Command("sudo", "chmod", "+x", execPath)
		if err := chmodCmd.Run(); err != nil {
			return fmt.Errorf("failed to make binary executable: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to install update: %w", err)
	}

	return nil
}

func downloadFile(url, dest string) error {
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer func() {
		if err := out.Close(); err != nil {
			interact.Error("Failed to close file: %v", err)
		}
	}()

	client := &http.Client{Timeout: 5 * time.Minute}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			interact.Error("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	return err
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := sourceFile.Close(); err != nil {
			interact.Error("Failed to close file: %v", err)
		}
	}()

	// Create the destination file
	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if err := destFile.Close(); err != nil {
			interact.Error("Failed to close file: %v", err)
		}
	}()

	// Set permissions to match original
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	err = os.Chmod(dst, srcInfo.Mode())
	if err != nil {
		return err
	}

	// Copy the contents
	_, err = io.Copy(destFile, sourceFile)
	return err
}
