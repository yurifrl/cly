package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	fs "github.com/NSXBet/nsx-cli/shared/filesystem"
	"github.com/NSXBet/nsx-cli/shared/interact"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync .cursor/rules from NSXBet/cursor-rules repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		return sync()
	},
}

func init() {
	RootCmd.AddCommand(syncCmd)
}

func sync() error {
	token := os.Getenv("GHA_PAT")
	if token == "" {
		interact.Error("GHA_PAT environment variable is not set. Please set it with a GitHub token.")
		return errors.New("missing GHA_PAT token")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if err := copyCursorRules(cwd); err != nil {
		interact.Error("Failed to download rules: %v", err)
		return err
	}

	interact.Success("Cursor rules successfully downloaded to .cursor/rules/")
	return nil
}

func copyCursorRules(destination string) error {
	rulesDir := filepath.Join(destination, ".cursor", "rules")
	if err := os.MkdirAll(rulesDir, fs.DirectoryPermission); err != nil {
		return fmt.Errorf("failed to create rules directory: %w", err)
	}

	tempDir, err := os.MkdirTemp("", "cursor-rules-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	repoUrl := fmt.Sprintf("https://%s@github.com/NSXBet/cursor-rules.git", os.Getenv("GHA_PAT"))
	cloneCmd := exec.Command("git", "clone", "--depth", "1", repoUrl, tempDir)
	if err := cloneCmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	sourceRulesDir := filepath.Join(tempDir, ".cursor", "rules")
	if _, err := os.Stat(sourceRulesDir); os.IsNotExist(err) {
		return fmt.Errorf("rules directory not found in repository")
	}

	if err := filepath.Walk(sourceRulesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourceRulesDir, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(rulesDir, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, fs.DirectoryPermission)
		}

		return fs.CopyFile(path, targetPath)
	}); err != nil {
		return fmt.Errorf("failed to copy rules files: %w", err)
	}

	return nil
}
