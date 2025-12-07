package builder

import (
	"fmt"
	"os/exec"
	"strings"
)

// initGitRepo initializes a new git repository
func initGitRepo(dir string, debug bool) error {
	cmd := exec.Command("git", "init", "-b", "main")
	cmd.Dir = dir
	if err := RunCommandWithLoader(cmd, "Initializing repository", debug); err != nil {
		return fmt.Errorf("failed to run 'git init': %w", err)
	}
	return nil
}

// createInitialCommit creates an initial commit for code generation
func createInitialCommit(dir string, debug bool) error {
	gitCommands := []struct {
		cmd  []string
		desc string
	}{
		{[]string{"git", "add", "."}, "Adding initial files to staging"},
		{[]string{"git", "commit", "-m", "initial commit - before code generation"}, "Creating initial commit"},
	}

	for _, gitCmd := range gitCommands {
		cmd := exec.Command(gitCmd.cmd[0], gitCmd.cmd[1:]...)
		cmd.Dir = dir
		if err := RunCommandWithLoader(cmd, gitCmd.desc, debug); err != nil {
			return fmt.Errorf("failed to run '%s': %w", strings.Join(gitCmd.cmd, " "), err)
		}
	}
	return nil
}

// updateFinalCommit updates the commit with all generated files
func updateFinalCommit(dir string, debug bool) error {
	gitCommands := []struct {
		cmd  []string
		desc string
	}{
		{[]string{"git", "add", "."}, "Adding all generated files"},
		{
			[]string{"git", "commit", "--amend", "-m", "initial commit"},
			"Updating commit with generated files",
		},
	}

	for _, gitCmd := range gitCommands {
		cmd := exec.Command(gitCmd.cmd[0], gitCmd.cmd[1:]...)
		cmd.Dir = dir
		if err := RunCommandWithLoader(cmd, gitCmd.desc, debug); err != nil {
			return fmt.Errorf("failed to run '%s': %w", strings.Join(gitCmd.cmd, " "), err)
		}
	}
	return nil
}

// isGitRepository checks if the current directory is a git repository
func isGitRepository(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = dir
	return cmd.Run() == nil
}
