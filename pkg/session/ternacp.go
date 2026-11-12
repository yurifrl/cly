package session

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/yurifrl/cly/pkg/envs"
)

// ExecTernACP replaces the process with `tern acp claude`, Tern's native
// agent view over @agentclientprotocol/claude-agent-acp. It only returns
// on failure (not inside a Tern pane, or tern not in PATH), so callers
// fall back to ExecClaude.
//
// yolo opens in bypassPermissions mode and sets CLY_YOLO=1. ACP can't
// append a system prompt, so the safe-autonomous prompt is injected by a
// SessionStart hook (`cly claude-yolo-context`) gated on CLY_YOLO.
func (s *Session) ExecTernACP(yolo bool, extra []string, opts ...ExecOption) error {
	if os.Getenv("TERN_PANE") == "" {
		return exec.ErrNotFound
	}
	path, err := exec.LookPath("tern")
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

	args := []string{"tern", "acp", "claude"}
	args = append(args, extra...)
	if yolo {
		env = append(env, "CLY_YOLO=1")
		args = append(args, "--mode", "bypassPermissions")
	}
	return syscall.Exec(path, args, env)
}
