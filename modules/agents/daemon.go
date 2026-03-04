package agents

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

// DaemonStatus is persisted by the background process for status checks.
type DaemonStatus struct {
	PID        int       `yaml:"pid"`
	StartedAt  time.Time `yaml:"started_at"`
	LastSyncAt time.Time `yaml:"last_sync_at,omitempty"`
	LastRepo   int       `yaml:"last_repo_count"`
	LastWrite  int       `yaml:"last_written"`
	LastSkip   int       `yaml:"last_skipped"`
	LastError  string    `yaml:"last_error,omitempty"`
}

func IsDaemonRunning() (bool, int) {
	pid, err := readPID()
	if err != nil {
		return false, 0
	}
	if err := syscall.Kill(pid, 0); err != nil {
		if errors.Is(err, syscall.ESRCH) {
			_ = os.Remove(PidFilePath())
			return false, 0
		}
		return false, 0
	}
	return true, pid
}

func readPID() (int, error) {
	data, err := os.ReadFile(PidFilePath())
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid pid file: %w", err)
	}
	return pid, nil
}

func writePID(pid int) error {
	if err := os.MkdirAll(GlobalConfigDir(), 0755); err != nil {
		return err
	}
	return os.WriteFile(PidFilePath(), []byte(strconv.Itoa(pid)), 0644)
}

func writeDaemonStatus(s DaemonStatus) error {
	if err := os.MkdirAll(GlobalConfigDir(), 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(StatusFilePath(), data, 0644)
}

func readDaemonStatus() (*DaemonStatus, error) {
	data, err := os.ReadFile(StatusFilePath())
	if err != nil {
		return nil, err
	}
	var s DaemonStatus
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}
