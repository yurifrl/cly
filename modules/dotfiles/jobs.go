package dotfiles

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	pkgconfig "github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/mut"
	"github.com/yurifrl/cly/pkg/style"
)

type JobRetryConfig struct {
	Enabled      bool
	MaxAttempts  int
	InitialDelay time.Duration
	Multiplier   int
	MaxDelay     time.Duration
	Jitter       bool
	ResetAfter   time.Duration
}

type JobApplyOptions struct {
	Force    bool
	FailFast bool
}

type jobPaths struct {
	ScriptsDir      string
	LaunchAgentsDir string
	LabelPrefix     string
}

type JobStatus struct {
	Name       string
	Run        JobRun
	Every      string
	KeepAlive  bool
	Registered bool
	Completed  bool
}

var (
	jobUserHomeDir = os.UserHomeDir
	launchctlRun   = func(args ...string) error {
		return mut.Exec("launchctl", args...)
	}
)

func ApplyJobs(cfg *Config, opts JobApplyOptions) error {
	if len(cfg.Jobs) == 0 {
		return nil
	}

	retryCfg, err := loadJobRetryConfig()
	if err != nil {
		return err
	}
	paths, err := loadJobPaths()
	if err != nil {
		return err
	}
	if err := mut.MkdirAll(paths.ScriptsDir, 0755); err != nil {
		return fmt.Errorf("create scripts dir: %w", err)
	}
	if err := mut.MkdirAll(paths.LaunchAgentsDir, 0755); err != nil {
		return fmt.Errorf("create launch agents dir: %w", err)
	}

	lockFile, err := lockFilePath()
	if err != nil {
		return err
	}
	lock, err := loadLock(lockFile)
	if err != nil {
		return err
	}
	hashes := lockJobsToMap(lock.Jobs)

	desired := make(map[string]bool)
	var jobErrs []error
	record := func(err error) bool {
		if opts.FailFast {
			return true
		}
		fmt.Printf("  %s %s\n", style.RedStyle.Render("❌"), err)
		jobErrs = append(jobErrs, err)
		return false
	}

	for _, job := range cfg.Jobs {
		desired[job.Name] = true

		scriptPath, err := writeJobScript(paths, job, cfg.BaseDir, retryCfg)
		if err != nil {
			if record(err) {
				return err
			}
			continue
		}

		switch job.Run {
		case JobRunStartup, JobRunInterval:
			plistPath, err := writeLaunchAgent(paths, job, scriptPath)
			if err != nil {
				if record(err) {
					return err
				}
				continue
			}
			unloadJob(paths, job.Name, plistPath)
			if err := launchctlRun("load", plistPath); err != nil {
				wrapped := fmt.Errorf("load launch agent %s: %w", job.Name, err)
				if record(wrapped) {
					return wrapped
				}
				continue
			}
		case JobRunOnce:
			hash := jobDefinitionHash(job, cfg.BaseDir)
			if !opts.Force && hashes[job.Name] == hash {
				continue
			}
			if err := runScript(scriptPath, cfg.BaseDir); err != nil {
				wrapped := fmt.Errorf("run once job %s: %w", job.Name, err)
				diagnoseJobFailure(job, cfg, paths, scriptPath, err)
				if record(wrapped) {
					return wrapped
				}
				continue
			}
			hashes[job.Name] = hash
		case JobRunCache:
			if !opts.Force && cacheCheckPasses(job) {
				fmt.Printf("  %s @cache %s (already installed)\n", style.SubtleStyle.Render("○"), job.Name)
				continue
			}
			fmt.Printf("  %s @cache %s\n", style.BlueStyle.Render("⚙️"), job.Name)
			if err := runScript(scriptPath, cfg.BaseDir); err != nil {
				wrapped := fmt.Errorf("run cache job %s: %w", job.Name, err)
				diagnoseJobFailure(job, cfg, paths, scriptPath, err)
				if record(wrapped) {
					return wrapped
				}
				continue
			}
		}
	}

	cleanupStaleManagedJobs(paths, desired, hashes)
	lock.Jobs = mapToLockJobs(hashes)
	if err := saveLock(lockFile, lock); err != nil {
		return err
	}

	if len(jobErrs) > 0 {
		fmt.Printf("%s %d job(s) failed (continuing without --fail-fast)\n",
			style.YellowStyle.Render("⚠️ "), len(jobErrs))
	}
	return nil
}

func RemoveJobs(cfg *Config) error {
	paths, err := loadJobPaths()
	if err != nil {
		return err
	}

	lockFile, err := lockFilePath()
	if err != nil {
		return err
	}
	lock, err := loadLock(lockFile)
	if err != nil {
		return err
	}
	hashes := lockJobsToMap(lock.Jobs)

	for _, job := range cfg.Jobs {
		delete(hashes, job.Name)
		_ = mut.Remove(jobScriptPath(paths, job.Name))
		if job.Run == JobRunStartup || job.Run == JobRunInterval {
			plistPath := jobPlistPath(paths, job.Name)
			unloadJob(paths, job.Name, plistPath)
			_ = mut.Remove(plistPath)
		}
	}

	lock.Jobs = mapToLockJobs(hashes)
	return saveLock(lockFile, lock)
}

// RemoveJobByName removes a single managed job by name (script + plist + lock entry).
func RemoveJobByName(name string) error {
	paths, err := loadJobPaths()
	if err != nil {
		return err
	}

	lockFile, err := lockFilePath()
	if err != nil {
		return err
	}
	lock, err := loadLock(lockFile)
	if err != nil {
		return err
	}
	hashes := lockJobsToMap(lock.Jobs)

	delete(hashes, name)
	_ = mut.Remove(jobScriptPath(paths, name))
	plistPath := jobPlistPath(paths, name)
	unloadJob(paths, name, plistPath)
	_ = mut.Remove(plistPath)

	lock.Jobs = mapToLockJobs(hashes)
	return saveLock(lockFile, lock)
}

var cacheCheckPasses = func(job Job) bool {
	check := job.Check
	if check == "" {
		check = job.Name
	}
	fields := strings.Fields(check)
	if len(fields) == 0 {
		return false
	}
	var shellCheck string
	if len(fields) == 1 {
		// Single word: `command -v <word>` (PATH lookup only).
		shellCheck = fmt.Sprintf("command -v %s >/dev/null 2>&1", fields[0])
	} else {
		// Multi-word: first ensure the binary exists on PATH, then execute
		// the full expression. Both must succeed (exit 0) to count as installed.
		shellCheck = fmt.Sprintf("command -v %s >/dev/null 2>&1 && %s >/dev/null 2>&1", fields[0], check)
	}
	cmd := exec.Command("sh", "-c", shellCheck)
	return cmd.Run() == nil
}

// diagnoseJobFailure prints a structured diagnostic block after a job fails.
// The subprocess stdout/stderr was already streamed live; this block adds
// metadata (config line, script path, launchd log paths, exit code) so the
// user can jump straight to the right file or tail the right log.
func diagnoseJobFailure(job Job, cfg *Config, paths jobPaths, scriptPath string, err error) {
	sep := strings.Repeat("─", 64)
	fmt.Fprintln(os.Stderr, style.RedStyle.Render(sep))
	fmt.Fprintf(os.Stderr, "%s @%s %q failed\n", style.RedStyle.Render("❌ job:"), job.Run, job.Name)
	fmt.Fprintln(os.Stderr, style.RedStyle.Render(sep))

	kv := func(k, v string) {
		if v == "" {
			return
		}
		fmt.Fprintf(os.Stderr, "  %-14s %s\n", style.SubtleStyle.Render(k), v)
	}

	kv("source:", fmt.Sprintf("dotfiles.conf:%d", job.LineNum))
	kv("command:", job.Command)
	kv("cwd:", cfg.BaseDir)
	kv("script:", scriptPath)
	if job.Run == JobRunCache {
		check := job.Check
		if check == "" {
			check = job.Name
		}
		fields := strings.Fields(check)
		var rendered string
		switch {
		case len(fields) == 1:
			rendered = fmt.Sprintf("command -v %s", fields[0])
		case len(fields) > 1:
			rendered = fmt.Sprintf("command -v %s && %s", fields[0], check)
		}
		kv("cache check:", rendered)
	}
	if job.Run == JobRunInterval {
		kv("every:", job.Every)
	}
	if job.Run == JobRunStartup || job.Run == JobRunInterval {
		kv("plist:", jobPlistPath(paths, job.Name))
		kv("stdout log:", filepath.Join(os.TempDir(), "cly-dotfiles-"+job.Name+".log"))
		kv("stderr log:", filepath.Join(os.TempDir(), "cly-dotfiles-"+job.Name+".error.log"))
	}
	if job.Run == JobRunStartup {
		kv("keepalive:", fmt.Sprintf("%t", job.KeepAlive))
	}

	var exit *exec.ExitError
	if errors.As(err, &exit) {
		kv("exit code:", strconv.Itoa(exit.ExitCode()))
	}
	kv("error:", err.Error())
	fmt.Fprintln(os.Stderr, style.RedStyle.Render(sep))
}

func StatusJobs(cfg *Config) ([]JobStatus, error) {
	paths, err := loadJobPaths()
	if err != nil {
		return nil, err
	}

	lockFile, err := lockFilePath()
	if err != nil {
		return nil, err
	}
	lock, err := loadLock(lockFile)
	if err != nil {
		return nil, err
	}
	hashes := lockJobsToMap(lock.Jobs)

	statuses := make([]JobStatus, 0, len(cfg.Jobs))
	for _, job := range cfg.Jobs {
		status := JobStatus{
			Name:      job.Name,
			Run:       job.Run,
			Every:     job.Every,
			KeepAlive: job.KeepAlive,
		}
		switch job.Run {
		case JobRunOnce:
			status.Completed = hashes[job.Name] == jobDefinitionHash(job, cfg.BaseDir)
		case JobRunCache:
			status.Completed = cacheCheckPasses(job)
		default:
			_, err := os.Stat(jobPlistPath(paths, job.Name))
			status.Registered = err == nil
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

func cleanupStaleManagedJobs(paths jobPaths, desired map[string]bool, hashes map[string]string) {
	entries, err := os.ReadDir(paths.ScriptsDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sh") {
				continue
			}
			name := strings.TrimSuffix(entry.Name(), ".sh")
			if desired[name] {
				continue
			}
			_ = mut.Remove(filepath.Join(paths.ScriptsDir, entry.Name()))
			delete(hashes, name)
		}
	}

	entries, err = os.ReadDir(paths.LaunchAgentsDir)
	if err == nil {
		prefix := paths.LabelPrefix + "."
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".plist") || !strings.HasPrefix(entry.Name(), prefix) {
				continue
			}
			name := strings.TrimSuffix(strings.TrimPrefix(entry.Name(), prefix), ".plist")
			if desired[name] {
				continue
			}
			plistPath := filepath.Join(paths.LaunchAgentsDir, entry.Name())
			unloadJob(paths, name, plistPath)
			_ = mut.Remove(plistPath)
		}
	}
}

func writeJobScript(paths jobPaths, job Job, baseDir string, retryCfg JobRetryConfig) (string, error) {
	path := jobScriptPath(paths, job.Name)
	content := renderJobScript(job, baseDir, retryCfg)
	if err := writeFileIfChanged(path, []byte(content), 0755); err != nil {
		return "", fmt.Errorf("write job script %s: %w", job.Name, err)
	}
	return path, nil
}

func writeLaunchAgent(paths jobPaths, job Job, scriptPath string) (string, error) {
	if job.Run == JobRunInterval {
		if _, err := time.ParseDuration(job.Every); err != nil {
			return "", fmt.Errorf("invalid interval for job %s: %w", job.Name, err)
		}
	}
	path := jobPlistPath(paths, job.Name)
	content := renderLaunchAgent(paths, job, scriptPath)
	if err := writeFileIfChanged(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("write launch agent %s: %w", job.Name, err)
	}
	return path, nil
}

func runScript(scriptPath, baseDir string) error {
	return mut.ExecDir(baseDir, scriptPath)
}

func renderJobScript(job Job, baseDir string, retryCfg JobRetryConfig) string {
	maxAttempts := retryCfg.MaxAttempts
	if !retryCfg.Enabled || maxAttempts < 1 {
		maxAttempts = 1
	}

	keepAlive := 0
	if job.KeepAlive {
		keepAlive = 1
	}
	jitter := 0
	if retryCfg.Jitter {
		jitter = 1
	}

	return fmt.Sprintf(`#!/usr/bin/env bash
set -uo pipefail

export PATH="/opt/homebrew/bin:/usr/local/bin:$HOME/.local/bin:$PATH"
cd %s

COMMAND=$(cat <<'CLY_JOB'
%s
CLY_JOB
)

MAX_ATTEMPTS=%d
INITIAL_DELAY=%d
MULTIPLIER=%d
MAX_DELAY=%d
JITTER=%d
KEEPALIVE=%d
RESET_AFTER=%d

next_delay() {
  local current=$1
  local candidate=$((current * MULTIPLIER))
  if [ "$candidate" -gt "$MAX_DELAY" ]; then
    echo "$MAX_DELAY"
  else
    echo "$candidate"
  fi
}

sleep_for_delay() {
  local current=$1
  local actual=$current
  if [ "$JITTER" -eq 1 ] && [ "$current" -gt 1 ]; then
    local min=$((current / 2))
    local span=$((current - min + 1))
    actual=$((min + RANDOM %% span))
    if [ "$actual" -lt 1 ]; then
      actual=1
    fi
  fi
  sleep "$actual"
}

attempt=1
delay=$INITIAL_DELAY

while true; do
  started_at=$(date +%%s)
  fish -lc "$COMMAND"
  exit_code=$?
  finished_at=$(date +%%s)
  runtime=$((finished_at - started_at))

  if [ "$KEEPALIVE" -ne 1 ] && [ "$exit_code" -eq 0 ]; then
    exit 0
  fi

  if [ "$KEEPALIVE" -eq 1 ] && [ "$runtime" -ge "$RESET_AFTER" ]; then
    attempt=1
    delay=$INITIAL_DELAY
  fi

  if [ "$attempt" -ge "$MAX_ATTEMPTS" ]; then
    exit "$exit_code"
  fi

  sleep_for_delay "$delay"
  attempt=$((attempt + 1))
  delay=$(next_delay "$delay")
done
`, strconv.Quote(baseDir), job.Command, maxAttempts, durationSeconds(retryCfg.InitialDelay), retryCfg.Multiplier, durationSeconds(retryCfg.MaxDelay), jitter, keepAlive, durationSeconds(retryCfg.ResetAfter))
}

func renderLaunchAgent(paths jobPaths, job Job, scriptPath string) string {
	label := jobLabel(paths, job.Name)
	stdoutPath := filepath.Join(os.TempDir(), "cly-dotfiles-"+job.Name+".log")
	stderrPath := filepath.Join(os.TempDir(), "cly-dotfiles-"+job.Name+".error.log")

	var extra strings.Builder
	if job.Run == JobRunStartup {
		extra.WriteString("    <key>RunAtLoad</key>\n    <true/>\n")
		if job.KeepAlive {
			extra.WriteString("    <key>KeepAlive</key>\n    <true/>\n")
		}
	}
	if job.Run == JobRunInterval {
		interval, _ := time.ParseDuration(job.Every)
		extra.WriteString("    <key>RunAtLoad</key>\n    <true/>\n")
		extra.WriteString(fmt.Sprintf("    <key>StartInterval</key>\n    <integer>%d</integer>\n", durationSeconds(interval)))
	}

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
    </array>
%s    <key>StandardOutPath</key>
    <string>%s</string>
    <key>StandardErrorPath</key>
    <string>%s</string>
</dict>
</plist>
`, label, scriptPath, extra.String(), stdoutPath, stderrPath)
}

func jobScriptPath(paths jobPaths, name string) string {
	return filepath.Join(paths.ScriptsDir, name+".sh")
}

func jobPlistPath(paths jobPaths, name string) string {
	return filepath.Join(paths.LaunchAgentsDir, jobLabel(paths, name)+".plist")
}

func jobLabel(paths jobPaths, name string) string {
	return paths.LabelPrefix + "." + name
}

// isJobLoaded checks if a launchd service is currently loaded
var isJobLoaded = func(label string) bool {
	cmd := exec.Command("launchctl", "list", label)
	return cmd.Run() == nil
}

// unloadJob only unloads if the service is actually loaded, preventing
// "Unload failed: 5: Input/output error" noise
func unloadJob(paths jobPaths, name, plistPath string) {
	label := jobLabel(paths, name)
	if isJobLoaded(label) {
		_ = launchctlRun("unload", plistPath)
	}
}

func jobDefinitionHash(job Job, baseDir string) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s\n%s\n%s\n%s\n%t\n%s\n", job.Run, job.Name, job.Command, job.Every, job.KeepAlive, baseDir)
	return hex.EncodeToString(h.Sum(nil))
}

func loadJobPaths() (jobPaths, error) {
	home, err := jobUserHomeDir()
	if err != nil {
		return jobPaths{}, err
	}

	dataDir := pkgconfig.GetString("app.data_dir")
	if dataDir == "" {
		dataDir = "~/.local/share/cly"
	}
	dataDir = expandTilde(dataDir)

	return jobPaths{
		ScriptsDir:      filepath.Join(dataDir, "dotfiles/jobs"),
		LaunchAgentsDir: filepath.Join(home, "Library/LaunchAgents"),
		LabelPrefix:     "com.yurifrl.dotfiles",
	}, nil
}

func loadJobRetryConfig() (JobRetryConfig, error) {
	cfg := JobRetryConfig{
		Enabled:      true,
		MaxAttempts:  5,
		InitialDelay: 2 * time.Second,
		Multiplier:   2,
		MaxDelay:     time.Minute,
		Jitter:       true,
		ResetAfter:   10 * time.Minute,
	}

	cfg.Enabled = dotfilesModuleBool(cfg.Enabled, "jobs", "retries", "enabled")
	cfg.Jitter = dotfilesModuleBool(cfg.Jitter, "jobs", "retries", "jitter")
	cfg.MaxAttempts = dotfilesModuleInt(cfg.MaxAttempts, "jobs", "retries", "max_attempts")
	cfg.Multiplier = dotfilesModuleInt(cfg.Multiplier, "jobs", "retries", "multiplier")

	if value := dotfilesModuleString("", "jobs", "retries", "initial_delay"); value != "" {
		d, err := time.ParseDuration(value)
		if err != nil {
			return JobRetryConfig{}, fmt.Errorf("invalid modules.dotfiles.jobs.retries.initial_delay: %w", err)
		}
		cfg.InitialDelay = d
	}
	if value := dotfilesModuleString("", "jobs", "retries", "max_delay"); value != "" {
		d, err := time.ParseDuration(value)
		if err != nil {
			return JobRetryConfig{}, fmt.Errorf("invalid modules.dotfiles.jobs.retries.max_delay: %w", err)
		}
		cfg.MaxDelay = d
	}
	if value := dotfilesModuleString("", "jobs", "retries", "reset_after"); value != "" {
		d, err := time.ParseDuration(value)
		if err != nil {
			return JobRetryConfig{}, fmt.Errorf("invalid modules.dotfiles.jobs.retries.reset_after: %w", err)
		}
		cfg.ResetAfter = d
	}

	if cfg.MaxAttempts < 1 {
		cfg.MaxAttempts = 1
	}
	if cfg.Multiplier < 1 {
		cfg.Multiplier = 1
	}
	if cfg.InitialDelay < time.Second {
		cfg.InitialDelay = time.Second
	}
	if cfg.MaxDelay < cfg.InitialDelay {
		cfg.MaxDelay = cfg.InitialDelay
	}
	if cfg.ResetAfter < time.Second {
		cfg.ResetAfter = time.Second
	}

	return cfg, nil
}

func durationSeconds(d time.Duration) int {
	seconds := int(d.Seconds())
	if seconds < 1 {
		return 1
	}
	return seconds
}

func writeFileIfChanged(path string, content []byte, mode os.FileMode) error {
	if existing, err := os.ReadFile(path); err == nil && string(existing) == string(content) {
		if err := mut.Chmod(path, mode); err != nil {
			return err
		}
		return nil
	}
	if err := mut.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := mut.WriteFile(path, content, mode); err != nil {
		return err
	}
	return nil
}

func dotfilesModuleValue(path ...string) interface{} {
	cfg := pkgconfig.Get()
	if cfg == nil {
		return nil
	}
	current, ok := cfg.Modules["dotfiles"]
	if !ok {
		return nil
	}
	var value interface{} = current
	for _, part := range path {
		m, ok := value.(map[string]interface{})
		if !ok {
			return nil
		}
		value, ok = m[part]
		if !ok {
			return nil
		}
	}
	return value
}

func dotfilesModuleString(def string, path ...string) string {
	value := dotfilesModuleValue(path...)
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	default:
		return def
	}
}

func dotfilesModuleInt(def int, path ...string) int {
	value := dotfilesModuleValue(path...)
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		parsed, err := strconv.Atoi(v)
		if err == nil {
			return parsed
		}
	}
	return def
}

func dotfilesModuleBool(def bool, path ...string) bool {
	value := dotfilesModuleValue(path...)
	switch v := value.(type) {
	case bool:
		return v
	case string:
		parsed, err := strconv.ParseBool(v)
		if err == nil {
			return parsed
		}
	}
	return def
}
