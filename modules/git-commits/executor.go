package gitcommits

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CommitResult represents the result of executing a single commit group.
type CommitResult struct {
	Title string
	SHA   string
	Files int
}

// Execute creates git commits according to the plan.
func Execute(plan *CommitPlan, noVerify bool) ([]CommitResult, error) {
	// Precondition: at least one existing commit
	if _, err := gitExec("rev-parse", "HEAD"); err != nil {
		return nil, fmt.Errorf("no initial commit found — create one first")
	}

	// Record original HEAD for rollback
	originalHead, err := gitOutput("rev-parse", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Save the full staged diff for recovery (--binary for binary files like .gif)
	savedDiff, err := gitRawOutput("diff", "--cached", "--binary")
	if err != nil {
		return nil, fmt.Errorf("failed to save diff: %w", err)
	}
	// Ensure diff ends with newline (prevents corrupt patch on rollback)
	if savedDiff != "" && !strings.HasSuffix(savedDiff, "\n") {
		savedDiff += "\n"
	}

	// Unstage everything (working tree preserved)
	if _, err := gitExec("reset"); err != nil {
		return nil, fmt.Errorf("git reset failed: %w", err)
	}

	var results []CommitResult

	for i, group := range plan.Groups {
		err := executeGroup(group, noVerify)
		if err != nil {
			// Rollback
			rollbackErr := rollback(originalHead, savedDiff)
			if rollbackErr != nil {
				return results, fmt.Errorf("commit %d failed: %w\nROLLBACK ALSO FAILED: %v\nManual recovery: git reset --soft %s",
					i+1, err, rollbackErr, originalHead)
			}
			return results, fmt.Errorf("commit %d failed (rolled back): %w", i+1, err)
		}

		sha, _ := gitOutput("rev-parse", "--short", "HEAD")
		results = append(results, CommitResult{
			Title: group.Title,
			SHA:   sha,
			Files: len(group.Files),
		})
	}

	return results, nil
}

func executeGroup(group CommitGroup, noVerify bool) error {
	// Stage files for this group
	for _, f := range group.Files {
		switch f.Status {
		case StatusAdded, StatusModified:
			if out, err := gitExec("add", "--", f.Path); err != nil {
				return fmt.Errorf("git add %q: %s: %w", f.Path, strings.TrimSpace(out), err)
			}
		case StatusDeleted:
			if out, err := gitExec("rm", "--cached", "--", f.Path); err != nil {
				return fmt.Errorf("git rm %q: %s: %w", f.Path, strings.TrimSpace(out), err)
			}
		case StatusRenamed:
			if f.OldPath != "" {
				if out, err := gitExec("rm", "--cached", "--", f.OldPath); err != nil {
					return fmt.Errorf("git rm %q: %s: %w", f.OldPath, strings.TrimSpace(out), err)
				}
			}
			if out, err := gitExec("add", "--", f.Path); err != nil {
				return fmt.Errorf("git add %q: %s: %w", f.Path, strings.TrimSpace(out), err)
			}
		}
	}

	// Build commit command
	args := []string{"commit", "-m", group.Title}
	if group.Body != "" {
		args = append(args, "-m", group.Body)
	}
	if noVerify {
		args = append(args, "--no-verify")
	}

	if _, err := gitExec(args...); err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}

	return nil
}

func rollback(originalHead, savedDiff string) error {
	// Step 1: move HEAD back
	if _, err := gitExec("reset", "--soft", originalHead); err != nil {
		return fmt.Errorf("reset --soft failed: %w", err)
	}

	// Step 2: unstage everything
	if _, err := gitExec("reset"); err != nil {
		return fmt.Errorf("reset failed: %w", err)
	}

	// Step 3: re-apply the original staged changes
	if savedDiff == "" {
		return nil
	}

	// Write diff to temp file and apply
	tmpFile, err := os.CreateTemp("", "git-commits-rollback-*.patch")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(savedDiff); err != nil {
		tmpFile.Close()
		return fmt.Errorf("write patch: %w", err)
	}
	tmpFile.Close()

	cmd := exec.Command("git", "apply", "--cached", tmpFile.Name())
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("apply --cached failed: %s: %w", out, err)
	}

	return nil
}

// gitOutput runs a git command and returns trimmed stdout only.
func gitOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// gitRawOutput runs a git command and returns raw stdout without trimming.
// Use for diff output where trailing newlines matter.
func gitRawOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
