package dotfiles

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyJobs_OnceRunsOnlyOnce(t *testing.T) {
	tmpHome := t.TempDir()
	oldHome := os.Getenv("HOME")
	require.NoError(t, os.Setenv("HOME", tmpHome))
	defer os.Setenv("HOME", oldHome)

	oldLaunchctl := launchctlRun
	launchctlRun = func(args ...string) error { return nil }
	defer func() { launchctlRun = oldLaunchctl }()

	cfg := &Config{
		BaseDir: tmpHome,
		Jobs: []Job{{
			Name:    "once-job",
			Run:     JobRunOnce,
			Command: "printf 'run\n' >> " + filepath.Join(tmpHome, "marker.txt"),
		}},
	}

	require.NoError(t, ApplyJobs(cfg, JobApplyOptions{}))
	require.NoError(t, ApplyJobs(cfg, JobApplyOptions{}))

	data, err := os.ReadFile(filepath.Join(tmpHome, "marker.txt"))
	require.NoError(t, err)
	assert.Equal(t, 1, strings.Count(string(data), "run\n"))
}

func TestApplyJobs_ForceRerunsOnce(t *testing.T) {
	tmpHome := t.TempDir()
	oldHome := os.Getenv("HOME")
	require.NoError(t, os.Setenv("HOME", tmpHome))
	defer os.Setenv("HOME", oldHome)

	oldLaunchctl := launchctlRun
	launchctlRun = func(args ...string) error { return nil }
	defer func() { launchctlRun = oldLaunchctl }()

	cfg := &Config{
		BaseDir: tmpHome,
		Jobs: []Job{{
			Name:    "once-job",
			Run:     JobRunOnce,
			Command: "printf 'run\n' >> " + filepath.Join(tmpHome, "marker.txt"),
		}},
	}

	require.NoError(t, ApplyJobs(cfg, JobApplyOptions{}))
	require.NoError(t, ApplyJobs(cfg, JobApplyOptions{Force: true}))

	data, err := os.ReadFile(filepath.Join(tmpHome, "marker.txt"))
	require.NoError(t, err)
	assert.Equal(t, 2, strings.Count(string(data), "run\n"))
}

func TestApplyJobs_StartupWritesScriptAndPlist(t *testing.T) {
	tmpHome := t.TempDir()
	oldHome := os.Getenv("HOME")
	require.NoError(t, os.Setenv("HOME", tmpHome))
	defer os.Setenv("HOME", oldHome)

	var calls [][]string
	oldLaunchctl := launchctlRun
	launchctlRun = func(args ...string) error {
		calls = append(calls, append([]string{}, args...))
		return nil
	}
	defer func() { launchctlRun = oldLaunchctl }()

	oldIsLoaded := isJobLoaded
	isJobLoaded = func(label string) bool { return true }
	defer func() { isJobLoaded = oldIsLoaded }()

	cfg := &Config{
		BaseDir: tmpHome,
		Jobs: []Job{{
			Name:      "claude-mem",
			Run:       JobRunStartup,
			KeepAlive: true,
			Command:   "echo hi",
		}},
	}

	require.NoError(t, ApplyJobs(cfg, JobApplyOptions{}))

	scriptPath := filepath.Join(tmpHome, ".local/share/cly/dotfiles/jobs", "claude-mem.sh")
	plistPath := filepath.Join(tmpHome, "Library/LaunchAgents", "com.yurifrl.dotfiles.claude-mem.plist")

	_, err := os.Stat(scriptPath)
	require.NoError(t, err)
	_, err = os.Stat(plistPath)
	require.NoError(t, err)

	data, err := os.ReadFile(plistPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "<string>com.yurifrl.dotfiles.claude-mem</string>")
	assert.Contains(t, string(data), "<key>KeepAlive</key>")
	require.Len(t, calls, 2)
	assert.Equal(t, []string{"unload", plistPath}, calls[0])
	assert.Equal(t, []string{"load", plistPath}, calls[1])
}

func TestApplyJobs_StartupSkipsUnloadWhenNotLoaded(t *testing.T) {
	tmpHome := t.TempDir()
	oldHome := os.Getenv("HOME")
	require.NoError(t, os.Setenv("HOME", tmpHome))
	defer os.Setenv("HOME", oldHome)

	var calls [][]string
	oldLaunchctl := launchctlRun
	launchctlRun = func(args ...string) error {
		calls = append(calls, append([]string{}, args...))
		return nil
	}
	defer func() { launchctlRun = oldLaunchctl }()

	oldIsLoaded := isJobLoaded
	isJobLoaded = func(label string) bool { return false }
	defer func() { isJobLoaded = oldIsLoaded }()

	cfg := &Config{
		BaseDir: tmpHome,
		Jobs: []Job{{
			Name:    "new-service",
			Run:     JobRunStartup,
			Command: "echo hi",
		}},
	}

	require.NoError(t, ApplyJobs(cfg, JobApplyOptions{}))

	// Only "load", no "unload" since service wasn't loaded
	require.Len(t, calls, 1)
	assert.Equal(t, "load", calls[0][0])
}
