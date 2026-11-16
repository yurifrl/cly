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
	headroom   bool
	baseURL    string
}

// WithAnthropicBaseURL sets ANTHROPIC_BASE_URL on the child process, for
// launchers (tern acp, tclaude) that spawn claude themselves.
func WithAnthropicBaseURL(url string) ExecOption {
	return func(c *execConfig) {
		c.baseURL = url
	}
}

// WithHeadroom launches claude through `headroom wrap claude` when the
// headroom binary is on PATH; otherwise claude is exec'd directly.
func WithHeadroom(on bool) ExecOption {
	return func(c *execConfig) {
		c.headroom = on
	}
}

// claudeExec resolves the binary and argv used to launch claude. With
// headroom enabled and installed it is `headroom wrap claude [-- args]`;
// a missing headroom silently falls back to plain claude.
func claudeExec(args []string, cfg execConfig) (string, []string, error) {
	if cfg.headroom {
		if hr, err := exec.LookPath("headroom"); err == nil {
			argv := []string{"headroom", "wrap", "claude"}
			if len(args) > 0 {
				argv = append(argv, "--")
				argv = append(argv, args...)
			}
			return hr, argv, nil
		}
	}
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return "", nil, errors.New("claude not found in PATH")
	}
	return claudePath, append([]string{"claude"}, args...), nil
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

	path, argv, err := claudeExec(args, cfg)
	if err != nil {
		return err
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

	return syscall.Exec(path, argv, env)
}

// ExecClaude execs claude with the given args (no session env).
func ExecClaude(args []string, opts ...ExecOption) error {
	var cfg execConfig
	for _, o := range opts {
		o(&cfg)
	}
	path, argv, err := claudeExec(args, cfg)
	if err != nil {
		return err
	}
	return syscall.Exec(path, argv, os.Environ())
}

// BuildAnonymousArgs appends the flags required to run claude without
// project-level setting sources.
func BuildAnonymousArgs(args []string) []string {
	return append(args, "--setting-sources", "user")
}

// ExecClaudeAnonymous execs claude in a fresh temp directory with
// project setting sources disabled.
func ExecClaudeAnonymous(args []string, opts ...ExecOption) error {
	var cfg execConfig
	for _, o := range opts {
		o(&cfg)
	}
	path, argv, err := claudeExec(BuildAnonymousArgs(args), cfg)
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "claude-anon-")
	if err != nil {
		return err
	}
	if err := os.Chdir(tmpDir); err != nil {
		return err
	}

	return syscall.Exec(path, argv, os.Environ())
}
