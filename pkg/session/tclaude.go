package session

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/yurifrl/cly/pkg/envs"
)

// ExecTclaude replaces the process with tclaude (tern-agents-ui), which
// renders the chat natively in Tern. It only returns on failure (e.g.
// tclaude not in PATH), so callers fall back to ExecClaude.
func (s *Session) ExecTclaude(yolo bool, opts ...ExecOption) error {
	path, err := exec.LookPath("tclaude")
	if err != nil {
		return err
	}

	var cfg execConfig
	for _, o := range opts {
		o(&cfg)
	}
	if err := envs.SetSessionName(s.Name); err != nil {
		return err
	}
	env := os.Environ()
	if cfg.taskListID != "" {
		env = append(env, "CLAUDE_CODE_TASK_LIST_ID="+cfg.taskListID)
	}
	if cfg.baseURL != "" {
		env = append(env, "ANTHROPIC_BASE_URL="+cfg.baseURL)
	}

	args := []string{"tclaude", "-title", s.Name}
	// tclaude defaults to its own model instead of Claude Code's setting.
	if m := claudeSettingsModel(); m != "" {
		args = append(args, "-model", m)
	}
	if yolo {
		args = append(args, "-allow-bypass", "-mode", "bypassPermissions",
			"-append-system-prompt", safeAutonomousPrompt)
	}
	return syscall.Exec(path, args, env)
}

func claudeSettingsModel() string {
	home, _ := os.UserHomeDir()
	data, err := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if err != nil {
		return ""
	}
	var s struct {
		Model string `json:"model"`
	}
	_ = json.Unmarshal(data, &s)
	return s.Model
}
