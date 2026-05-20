package gitcommits

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/yurifrl/cly/pkg/style"
)

// listSubmodulePaths returns submodule paths declared in .gitmodules,
// resolved relative to the repo root.
func listSubmodulePaths() []string {
	root := repoRoot()
	if _, err := os.Stat(filepath.Join(root, ".gitmodules")); err != nil {
		return nil
	}
	out, err := gitOutput("config", "--file", ".gitmodules", "--get-regexp", `^submodule\..*\.path$`)
	if err != nil || out == "" {
		return nil
	}
	var paths []string
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 {
			paths = append(paths, fields[1])
		}
	}
	return paths
}

// submoduleStatus describes what's dirty in a submodule.
type submoduleStatus struct {
	Path       string
	HasStaged  bool
	Unstaged   []string // unstaged-modified or untracked paths (porcelain lines)
}

// dirtySubmodules returns submodule paths that have uncommitted changes
// (staged, unstaged, or untracked) in their working tree.
func dirtySubmodules() []string {
	statuses := submoduleStatuses()
	var out []string
	for _, s := range statuses {
		out = append(out, s.Path)
	}
	return out
}

// submoduleStatuses returns one entry per dirty submodule with staged/unstaged breakdown.
func submoduleStatuses() []submoduleStatus {
	paths := listSubmodulePaths()
	if len(paths) == 0 {
		return nil
	}
	root := repoRoot()
	var dirty []submoduleStatus
	for _, p := range paths {
		abs := filepath.Join(root, p)
		if _, err := os.Stat(filepath.Join(abs, ".git")); err != nil {
			continue
		}
		cmd := exec.Command("git", "status", "--porcelain")
		cmd.Dir = abs
		outBytes, err := cmd.Output()
		if err != nil {
			continue
		}
		trimmed := bytes.TrimSpace(outBytes)
		if len(trimmed) == 0 {
			continue
		}
		s := submoduleStatus{Path: p}
		// Canonical staged-changes check: use the *same* diff invocation that
		// GetChangeset relies on. `--quiet` can disagree with parsed output
		// (e.g. empty rename-detected diffs), so trust the parser's view.
		diffCmd := exec.Command("git", "-c", "diff.submodule=short", "diff", "--cached", "-M", "--no-color", "--binary")
		diffCmd.Dir = abs
		if diffOut, err := diffCmd.Output(); err == nil && len(bytes.TrimSpace(diffOut)) > 0 {
			s.HasStaged = true
		}
		for _, line := range strings.Split(string(trimmed), "\n") {
			if len(line) < 2 {
				continue
			}
			index := line[0]
			worktree := line[1]
			if index == '?' || worktree != ' ' {
				s.Unstaged = append(s.Unstaged, line)
			}
		}
		dirty = append(dirty, s)
	}
	return dirty
}

// confirmSubmoduleCommit asks the user whether to commit dirty submodules first.
// Returns true on yes (default), false on no.
func confirmSubmoduleCommit(paths []string) bool {
	fmt.Println(style.YellowStyle.Render(fmt.Sprintf("📦 %d submodule(s) have uncommitted changes:", len(paths))))
	for _, p := range paths {
		fmt.Printf("   - %s\n", p)
	}
	fmt.Println(style.SubtleStyle.Render("  Commit these submodules first via cly git-commits? [Y]es  [n]o"))
	fmt.Print("→ ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	lower := strings.ToLower(strings.TrimSpace(input))
	return lower == "" || lower == "y" || lower == "yes"
}

// commitSubmodules invokes the cly git-commits pipeline (this same binary)
// inside each submodule path, propagating the relevant flags.
func commitSubmodules(paths []string, opts pipelineOpts) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate cly binary: %w", err)
	}
	root := repoRoot()
	for _, p := range paths {
		fmt.Println()
		fmt.Println(style.BlueStyle.Render(fmt.Sprintf("📦 git-commits in submodule: %s", p)))

		args := []string{"git-commits"}
		if opts.DryRun {
			args = append(args, "--dry-run")
		}
		if opts.Yes {
			args = append(args, "--yes")
		}
		if opts.All {
			args = append(args, "--all")
		}
		if opts.NoVerify {
			args = append(args, "--no-verify")
		}
		if opts.Strategy != "" && opts.Strategy != StrategyFile {
			args = append(args, "--strategy", opts.Strategy)
		}
		if opts.Ignored {
			args = append(args, "--ignored")
		}
		if opts.NoSubmodule {
			args = append(args, "--no-submodule")
		}
		if opts.Prompt != "" {
			args = append(args, "--prompt", opts.Prompt)
		}
		// Intentionally NOT propagated: --json, --yolo, --push (push happens
		// only for the parent repo).

		cmd := exec.Command(exe, args...)
		cmd.Dir = filepath.Join(root, p)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("submodule %s: %w", p, err)
		}
		// Stage the updated submodule pointer in the parent so it lands in
		// the parent commit alongside the rest of the changes.
		if out, err := gitExec("add", "--", p); err != nil {
			return fmt.Errorf("stage submodule pointer %s: %s: %w", p, strings.TrimSpace(out), err)
		}
	}
	return nil
}

// stageUnstagedSubmodulePointers `git add`s every submodule path whose
// pointer in the parent working tree differs from the index. Returns
// the paths that were staged.
func stageUnstagedSubmodulePointers() ([]string, error) {
	paths := listSubmodulePaths()
	if len(paths) == 0 {
		return nil, nil
	}
	var staged []string
	for _, p := range paths {
		cmd := exec.Command("git", "diff", "--quiet", "--", p)
		cmd.Dir = repoRoot()
		if err := cmd.Run(); err == nil {
			continue
		}
		if out, err := gitExec("add", "--", p); err != nil {
			return staged, fmt.Errorf("stage submodule %s: %s: %w", p, strings.TrimSpace(out), err)
		}
		staged = append(staged, p)
	}
	return staged, nil
}
