package zs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func IsInsideZellij() bool {
	return os.Getenv("ZELLIJ") != ""
}

func ListSessionNames() ([]string, error) {
	debugf("running: zellij list-sessions -s")
	out, err := exec.Command("zellij", "list-sessions", "-s").CombinedOutput()
	if err != nil {
		text := string(out)
		if strings.Contains(text, "No active zellij sessions") || strings.Contains(text, "No active sessions") {
			debugf("no active sessions reported")
			return nil, nil
		}
		debugf("zellij list-sessions failed output=%q", text)
		return nil, fmt.Errorf("list zellij sessions: %w", err)
	}

	result := parseSessionNames(string(out))
	return result, nil
}

func parseSessionNames(output string) []string {
	var sessions []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		clean := ansiPattern.ReplaceAllString(line, "")
		fields := strings.Fields(clean)
		if len(fields) > 0 {
			sessions = append(sessions, fields[0])
		}
	}
	return sessions
}

func SessionExists(name string, sessions []string) bool {
	for _, session := range sessions {
		if session == name {
			return true
		}
	}
	return false
}

func SessionNameForDir(dir, provided string) string {
	if provided != "" {
		return provided
	}
	return sanitizeSessionName(filepath.Base(dir))
}

func sanitizeSessionName(name string) string {
	replacer := strings.NewReplacer(" ", "_", ".", "_", ":", "_")
	return replacer.Replace(name)
}

func AttachSession(name string) error {
	args := []string{"zellij", "attach", name}
	debugf("attach args=%s", shellJoin(args))
	return execZellij(args...)
}

func BuildCreateSessionArgs(name, dir, layout string) []string {
	if layout == "" {
		layout = "default"
	}
	return []string{"zellij", "--session", name, "--new-session-with-layout", layout, "options", "--default-cwd", dir}
}

func CreateSession(name, dir, layout string) error {
	args := BuildCreateSessionArgs(name, dir, layout)
	debugf("create session args=%s", shellJoin(args))
	return execZellij(args...)
}

func BuildNewTabArgs(name, dir, layout string) []string {
	args := []string{"action", "new-tab"}
	if layout != "" {
		args = append(args, "--layout", layout)
	}
	args = append(args, "--name", name, "--cwd", dir)
	return args
}

func NewTab(name, dir, layout string) error {
	args := append([]string{"zellij"}, BuildNewTabArgs(name, dir, layout)...)
	debugf("new tab args=%s", shellJoin(args))
	if runtimeCfg.DryRun {
		printDryRun(args...)
		return nil
	}

	cmd := exec.Command("zellij", BuildNewTabArgs(name, dir, layout)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func GoToTab(name string) error {
	args := []string{"zellij", "action", "go-to-tab-name", name}
	debugf("go to tab args=%s", shellJoin(args))
	if runtimeCfg.DryRun {
		printDryRun(args...)
		return nil
	}

	cmd := exec.Command("zellij", "action", "go-to-tab-name", name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func ListZoxideDirs() ([]string, error) {
	debugf("running: zoxide query -l")
	out, err := exec.Command("zoxide", "query", "-l").CombinedOutput()
	if err != nil {
		text := strings.TrimSpace(string(out))
		if text == "" {
			debugf("zoxide returned no directories")
			return nil, nil
		}
		debugf("zoxide query failed output=%q", text)
		return nil, fmt.Errorf("list zoxide directories: %w", err)
	}

	var dirs []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			dirs = append(dirs, line)
		}
	}
	return dirs, nil
}

func execZellij(argv ...string) error {
	debugf("exec zellij argv=%s", shellJoin(argv))
	if runtimeCfg.DryRun {
		printDryRun(argv...)
		return nil
	}

	binary := argv[0]
	path, err := exec.LookPath(binary)
	if err != nil {
		return fmt.Errorf("%s not found in PATH", binary)
	}
	return syscall.Exec(path, argv, os.Environ())
}
