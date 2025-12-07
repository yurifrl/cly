package builder

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/NSXBet/nsx-cli/shared/helpers"
	"github.com/NSXBet/nsx-cli/shared/interact"
)

const gistURL = "/gists/refs/heads/main/heynemann/.golangci.yml"

// runMakeGen runs the make gen command to generate code
func runMakeGen(dir string, debug bool) error {
	cmd := exec.Command("make", "gen")
	cmd.Dir = dir
	return RunCommandWithLoader(cmd, fmt.Sprintf("Running 'make gen' in %s", dir), debug)
}

// runGoModTidy runs go mod tidy to organize dependencies
func runGoModTidy(dir string, debug bool) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir
	return RunCommandWithLoader(cmd, fmt.Sprintf("Running 'go mod tidy' in %s", dir), debug)
}

// runMakeFormat runs the make format command to format code
func runMakeFormat(dir string, debug bool) error {
	cmd := exec.Command("make", "format")
	cmd.Dir = dir
	return RunCommandWithLoader(cmd, fmt.Sprintf("Running 'make format' in %s", dir), debug)
}

// fixDependencyConflicts fixes known dependency conflicts
func fixDependencyConflicts(dir string, debug bool) error {
	dependencyCommands := []struct {
		cmd  []string
		desc string
	}{
		{[]string{"go", "get", "-u", "github.com/containerd/containerd"}, "Updating containerd to latest version"},
		{[]string{"go", "get", "github.com/containerd/containerd/api@v1.8.0"}, "Setting containerd API to v1.8.0"},
		{[]string{"go", "get", "-u", "github.com/NSXBet/nsf@latest"}, "Updating nsf to latest version"},
	}

	for _, depCmd := range dependencyCommands {
		cmd := exec.Command(depCmd.cmd[0], depCmd.cmd[1:]...)
		cmd.Dir = dir
		if err := RunCommandWithLoader(cmd, depCmd.desc, debug); err != nil {
			return fmt.Errorf("failed to run '%s': %w", strings.Join(depCmd.cmd, " "), err)
		}
	}
	return nil
}

// FetchGolangCIConfig fetches golangci-lint config from gist and creates .golangci.yml
func FetchGolangCIConfig(proxyURL, dir string, debug bool) error {
	// Check if current directory is a git repository
	if !isGitRepository(dir) {
		interact.Warn("current directory is not a git repository")
	}

	err := helpers.DownloadFileToPath(proxyURL+gistURL, dir, debug)
	if err != nil {
		return fmt.Errorf("failed to download golangci-lint config: %w", err)
	}

	return nil
}

// installWizCursor install the wiz-cursor tool in the specific directory.
func installWizCursor(dir string, debug bool) error {
	cmd := exec.Command(
		"bash",
		"-c",
		"curl -fsSL https://raw.githubusercontent.com/NSXBet/wiz-cursor/refs/heads/main/install.sh | bash",
	)
	cmd.Dir = dir

	return RunCommandWithLoader(cmd, "Installing wiz-cursor", debug)
}
