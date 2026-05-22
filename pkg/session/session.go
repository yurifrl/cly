package session

import (
	"errors"
	"os"
	"os/exec"
	"regexp"
	"syscall"

	"github.com/yurifrl/cly/pkg/envs"
)

// Session represents a named session passed through to downstream
// agent CLIs. The name is propagated via env vars owned by pkg/envs.
type Session struct {
	Name string
}

var validNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// Initialize resolves a session name from (in order):
//
//   1. The explicit name argument.
//   2. envs.SessionName() — canonical $CLY_SESSION_NAME, falling back
//      to legacy $CLAUDE_SESSION_NAME.
//   3. An auto-generated AdjectiveAnimal name.
//
// The resolved name is validated before the Session is returned.
func Initialize(name string) (*Session, error) {
	if name == "" {
		name = envs.SessionName().Or("")
	}
	if name == "" {
		name = GenerateName()
	}

	if err := ValidateName(name); err != nil {
		return nil, err
	}

	return &Session{Name: name}, nil
}

// ValidateName enforces the session-name character class. Empty names
// and names containing characters outside [a-zA-Z0-9_-] are rejected.
func ValidateName(name string) error {
	if name == "" {
		return errors.New("session name cannot be empty")
	}
	if !validNamePattern.MatchString(name) {
		return errors.New("session name must contain only alphanumeric characters, hyphens, or underscores")
	}
	return nil
}

// IsInZellij reports whether the process is running inside a Zellij
// session. Thin alias preserved for callers; new code should use
// envs.InZellij directly.
func IsInZellij() bool {
	return envs.InZellij()
}

// RenameZellijTab renames the surrounding Zellij tab to the session
// name. No-op when not running in Zellij.
func (s *Session) RenameZellijTab() error {
	if !envs.InZellij() {
		return nil
	}
	cmd := exec.Command("zellij", "action", "rename-tab", s.Name)
	return cmd.Run()
}

// ExecOption configures how claude is exec'd.
type ExecOption func(*execConfig)

type execConfig struct {
	taskListID string
}

// WithTaskListID sets the CLAUDE_CODE_TASK_LIST_ID env var on the
// child process.
func WithTaskListID(id string) ExecOption {
	return func(c *execConfig) {
		c.taskListID = id
	}
}

// ExecClaude replaces the current process with `claude`, propagating
// the session name through pkg/envs (writes both canonical and legacy
// keys for backward compatibility) and any configured options.
func (s *Session) ExecClaude(args []string, opts ...ExecOption) error {
	var cfg execConfig
	for _, o := range opts {
		o(&cfg)
	}

	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return errors.New("claude not found in PATH")
	}

	// Write to the active source so os.Environ() picks up both the
	// canonical and legacy session env vars in a single place.
	if err := envs.SetSessionName(s.Name); err != nil {
		return err
	}

	env := os.Environ()
	if cfg.taskListID != "" {
		env = append(env, "CLAUDE_CODE_TASK_LIST_ID="+cfg.taskListID)
	}

	execArgs := append([]string{"claude"}, args...)
	return syscall.Exec(claudePath, execArgs, env)
}

// ExecClaude execs claude with the given args (no session env).
func ExecClaude(args []string) error {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return errors.New("claude not found in PATH")
	}
	execArgs := append([]string{"claude"}, args...)
	return syscall.Exec(claudePath, execArgs, os.Environ())
}

// BuildAnonymousArgs appends the flags required to run claude without
// project-level setting sources.
func BuildAnonymousArgs(args []string) []string {
	return append(args, "--setting-sources", "user")
}

// ExecClaudeAnonymous execs claude in a fresh temp directory with
// project setting sources disabled.
func ExecClaudeAnonymous(args []string) error {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return errors.New("claude not found in PATH")
	}

	tmpDir, err := os.MkdirTemp("", "claude-anon-")
	if err != nil {
		return err
	}
	if err := os.Chdir(tmpDir); err != nil {
		return err
	}

	execArgs := append([]string{"claude"}, BuildAnonymousArgs(args)...)
	return syscall.Exec(claudePath, execArgs, os.Environ())
}
