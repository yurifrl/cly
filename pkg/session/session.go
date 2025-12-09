package session

import (
	"errors"
	"os"
	"os/exec"
	"regexp"
	"syscall"
)

type Session struct {
	Name string
}

var validNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func Initialize(name string) (*Session, error) {
	if name == "" {
		name = os.Getenv("CLAUDE_SESSION_NAME")
	}
	if name == "" {
		name = GenerateName()
	}

	if err := ValidateName(name); err != nil {
		return nil, err
	}

	return &Session{Name: name}, nil
}

func ValidateName(name string) error {
	if name == "" {
		return errors.New("session name cannot be empty")
	}
	if !validNamePattern.MatchString(name) {
		return errors.New("session name must contain only alphanumeric characters, hyphens, or underscores")
	}
	return nil
}

func IsInZellij() bool {
	_, exists := os.LookupEnv("ZELLIJ")
	return exists
}

func (s *Session) RenameZellijTab() error {
	if !IsInZellij() {
		return nil
	}
	cmd := exec.Command("zellij", "action", "rename-tab", s.Name)
	return cmd.Run()
}

func (s *Session) ExecClaude(args []string) error {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return errors.New("claude not found in PATH")
	}

	env := os.Environ()
	env = append(env, "CLAUDE_SESSION_NAME="+s.Name)

	execArgs := append([]string{"claude"}, args...)
	return syscall.Exec(claudePath, execArgs, env)
}
