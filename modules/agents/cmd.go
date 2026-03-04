package agents

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var (
	flagDryRun bool
	flagIDE    string

	flagFollow bool
	flagTail   int
)

const daemonSyncInterval = 3 * time.Second

// Register adds the agents command tree to the parent.
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "agents",
		Short: "Sync .agents configs to IDE directories",
		Long:  "Synchronize repository .agents configurations to IDE-specific directories (.claude, .opencode, .crush).",
	}

	cmd.PersistentFlags().BoolVar(&flagDryRun, "dry-run", false, "Show what would be synced without writing")
	cmd.PersistentFlags().StringVarP(&flagIDE, "ide", "i", "", "Sync only this IDE (claude, opencode, crush)")

	syncCmd := &cobra.Command{
		Use:   "sync [repo]",
		Short: "Run one sync and exit",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runSync,
	}

	startCmd := &cobra.Command{
		Use:   "start [repo]",
		Short: "Start background sync daemon",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runStart,
	}

	addCmd := &cobra.Command{
		Use:   "add [repo]",
		Short: "Add repository to global sync list",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runAdd,
	}

	logsCmd := &cobra.Command{
		Use:   "logs",
		Short: "Show daemon logs",
		RunE:  runLogs,
	}
	logsCmd.Flags().BoolVarP(&flagFollow, "follow", "f", false, "Follow log output")
	logsCmd.Flags().IntVar(&flagTail, "tail", 100, "Number of lines to show")

	internalDaemonCmd := &cobra.Command{
		Use:    "__daemon",
		Hidden: true,
		RunE:   runDaemon,
	}

	cmd.AddCommand(
		syncCmd,
		startCmd,
		addCmd,
		logsCmd,
		&cobra.Command{
			Use:   "status",
			Short: "Show daemon status",
			RunE:  runStatus,
		},
		&cobra.Command{
			Use:   "stop",
			Short: "Stop running daemon",
			RunE:  runStop,
		},
		internalDaemonCmd,
	)

	parent.AddCommand(cmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	repo, err := resolveRepoArg(args)
	if err != nil {
		return err
	}

	cfg, err := LoadGlobalConfig()
	if err != nil {
		return err
	}

	ides := chooseIDEs(cfg)
	stats, err := syncRepo(repo, ides, flagDryRun)
	if err != nil {
		return err
	}

	logSuccess("sync completed",
		lf("repo", repo),
		lf("written", stats.Written),
		lf("skipped", stats.Skipped),
		lf("errors", stats.Errors),
		lf("dry_run", flagDryRun),
	)
	if stats.Errors > 0 {
		logWarn("sync completed with errors", lf("repo", repo), lf("errors", stats.Errors))
	}
	return nil
}

func runAdd(cmd *cobra.Command, args []string) error {
	repo, err := resolveRepoArg(args)
	if err != nil {
		return err
	}

	cfg, err := LoadGlobalConfig()
	if err != nil {
		return err
	}

	added := AddRepo(cfg, repo)
	if err := SaveGlobalConfig(cfg); err != nil {
		return err
	}

	if added {
		logSuccess("repo added", lf("repo", repo))
	} else {
		logInfo("repo already tracked", lf("repo", repo))
	}

	running, _ := IsDaemonRunning()
	if running {
		logInfo("daemon running; repo will sync on next cycle", lf("repo", repo))
	}
	return nil
}

func runStart(cmd *cobra.Command, args []string) error {
	repo, err := resolveRepoArg(args)
	if err != nil {
		return err
	}

	cfg, err := LoadGlobalConfig()
	if err != nil {
		return err
	}

	added := AddRepo(cfg, repo)
	if err := SaveGlobalConfig(cfg); err != nil {
		return err
	}

	running, pid := IsDaemonRunning()
	if running {
		if added {
			logInfo("daemon already running; repo added", lf("pid", pid), lf("repo", repo))
		} else {
			logInfo("daemon already running", lf("pid", pid))
		}
		return nil
	}

	if err := os.MkdirAll(GlobalConfigDir(), 0755); err != nil {
		return err
	}

	logFile, err := os.OpenFile(LogFilePath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer logFile.Close()

	daemonCmd := exec.Command(os.Args[0], "agents", "__daemon")
	daemonCmd.Stdout = logFile
	daemonCmd.Stderr = logFile
	daemonCmd.Stdin = nil
	daemonCmd.Env = os.Environ()
	daemonCmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := daemonCmd.Start(); err != nil {
		return err
	}

	for i := 0; i < 30; i++ {
		time.Sleep(100 * time.Millisecond)
		running, pid = IsDaemonRunning()
		if running {
			logSuccess("daemon started", lf("pid", pid), lf("log_file", LogFilePath()))
			return nil
		}
	}

	return errors.New("daemon failed to start")
}

func runDaemon(cmd *cobra.Command, args []string) error {
	running, pid := IsDaemonRunning()
	if running && pid != os.Getpid() {
		return fmt.Errorf("daemon already running (pid %d)", pid)
	}

	if err := writePID(os.Getpid()); err != nil {
		return err
	}
	defer os.Remove(PidFilePath())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	status := DaemonStatus{
		PID:       os.Getpid(),
		StartedAt: time.Now(),
	}
	_ = writeDaemonStatus(status)

	syncAndUpdate := func() {
		stats, err := syncAllConfiguredRepos(false)
		status.LastSyncAt = time.Now()
		status.LastRepo = stats.Repos
		status.LastWrite = stats.Written
		status.LastSkip = stats.Skipped
		if err != nil {
			status.LastError = err.Error()
			logError("sync cycle failed", lf("error", err))
		} else {
			status.LastError = ""
		}
		_ = writeDaemonStatus(status)
		if stats.Written > 0 || stats.Errors > 0 {
			logInfo("sync cycle", lf("repos", stats.Repos), lf("written", stats.Written), lf("skipped", stats.Skipped), lf("errors", stats.Errors))
		}
	}

	syncAndUpdate()

	ticker := time.NewTicker(daemonSyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			syncAndUpdate()
		}
	}
}

func runStatus(cmd *cobra.Command, args []string) error {
	running, pid := IsDaemonRunning()
	if running {
		logInfo("daemon status", lf("running", "yes"), lf("pid", pid))
	} else {
		logInfo("daemon status", lf("running", "no"))
	}

	cfg, err := LoadGlobalConfig()
	if err == nil {
		logInfo("tracked repositories", lf("count", len(cfg.Repos)))
	}

	status, err := readDaemonStatus()
	if err == nil {
		logInfo("daemon lifecycle", lf("started_at", status.StartedAt.Format(time.RFC3339)))
		if !status.LastSyncAt.IsZero() {
			logInfo("last sync", lf("at", status.LastSyncAt.Format(time.RFC3339)))
			logInfo("last cycle", lf("repos", status.LastRepo), lf("written", status.LastWrite), lf("skipped", status.LastSkip))
		}
		if status.LastError != "" {
			logError("last daemon error", lf("error", status.LastError))
		}
	}

	return nil
}

func runStop(cmd *cobra.Command, args []string) error {
	running, pid := IsDaemonRunning()
	if !running {
		return fmt.Errorf("daemon not running")
	}

	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		return err
	}

	for i := 0; i < 30; i++ {
		time.Sleep(100 * time.Millisecond)
		running, _ := IsDaemonRunning()
		if !running {
			logSuccess("daemon stopped")
			return nil
		}
	}

	return fmt.Errorf("daemon did not stop in time (pid %d)", pid)
}

func runLogs(cmd *cobra.Command, args []string) error {
	if flagTail < 0 {
		return fmt.Errorf("--tail must be >= 0")
	}

	if err := printLogTail(LogFilePath(), flagTail); err != nil {
		return err
	}

	if !flagFollow {
		return nil
	}

	return followLog(LogFilePath())
}

func printLogTail(path string, tail int) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("log file not found: %s", path)
		}
		return err
	}

	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	start := 0
	if tail > 0 && len(lines) > tail {
		start = len(lines) - tail
	}

	for _, line := range lines[start:] {
		fmt.Println(line)
	}
	return nil
}

func followLog(path string) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	pos, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}

	reader := bufio.NewReader(f)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			stat, err := os.Stat(path)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return err
			}
			if stat.Size() < pos {
				if _, err := f.Seek(0, io.SeekStart); err != nil {
					return err
				}
				reader.Reset(f)
				pos = 0
			}
			if stat.Size() == pos {
				continue
			}
			if _, err := f.Seek(pos, io.SeekStart); err != nil {
				return err
			}
			reader.Reset(f)
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					if errors.Is(err, io.EOF) {
						break
					}
					return err
				}
				fmt.Print(line)
			}
			currentPos, err := f.Seek(0, io.SeekCurrent)
			if err != nil {
				return err
			}
			pos = currentPos
		}
	}
}

type syncSummary struct {
	Repos   int
	Written int
	Skipped int
	Errors  int
}

func syncAllConfiguredRepos(dryRun bool) (syncSummary, error) {
	cfg, err := LoadGlobalConfig()
	if err != nil {
		return syncSummary{}, err
	}

	ides := chooseIDEs(cfg)
	totals := syncSummary{}

	for _, repo := range cfg.Repos {
		repoStats, err := syncRepo(repo, ides, dryRun)
		if err != nil {
			totals.Errors++
			logError("repo sync failed", lf("repo", repo), lf("error", err))
			continue
		}
		totals.Repos++
		totals.Written += repoStats.Written
		totals.Skipped += repoStats.Skipped
		totals.Errors += repoStats.Errors
	}

	return totals, nil
}

func syncRepo(repo string, ides []string, dryRun bool) (syncSummary, error) {
	totals := syncSummary{Repos: 1}
	sourceDir := filepath.Join(repo, ".agents")

	if st, err := os.Stat(sourceDir); err != nil || !st.IsDir() {
		return syncSummary{}, fmt.Errorf("missing %s", sourceDir)
	}

	for _, ideName := range ides {
		ide := GetIDEDef(ideName)
		if ide == nil {
			totals.Errors++
			continue
		}

		targetBase := filepath.Join(repo, ide.LocalDir)
		plan, err := Discover(sourceDir, ide, targetBase)
		if err != nil {
			totals.Errors++
			continue
		}
		if len(plan.Items) == 0 {
			continue
		}

		result, err := Reconcile(plan, dryRun)
		if err != nil {
			totals.Errors++
			continue
		}
		totals.Written += result.Written
		totals.Skipped += result.Skipped
		totals.Errors += len(result.Errors)
	}

	return totals, nil
}

func chooseIDEs(cfg *Config) []string {
	ides := cfg.IDEs
	if flagIDE != "" {
		ides = []string{flagIDE}
	}
	return ides
}

func resolveRepoArg(args []string) (string, error) {
	path := "."
	if len(args) == 1 {
		path = args[0]
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)

	st, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !st.IsDir() {
		return "", fmt.Errorf("not a directory: %s", abs)
	}
	return abs, nil
}
