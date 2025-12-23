package backup

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	pkgconfig "github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/style"
)

var (
	outputPath   string
	uploadFlag   bool
	downloadFlag bool
)

type operationType int

const (
	opUpload operationType = iota
	opSkipped
	opError
	opProgress
	opInfo
)

type syncStats struct {
	uploaded int
	skipped  int
	failed   int
	mu       sync.Mutex
}

func (s *syncStats) increment(op operationType) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch op {
	case opUpload:
		s.uploaded++
	case opSkipped:
		s.skipped++
	case opError:
		s.failed++
	}
}

func categorizeGsutilLine(line string) (operationType, bool, bool) {
	// Returns: (opType, shouldPrint, shouldCount)
	line = strings.TrimSpace(line)

	// Actual file copying/syncing - count this
	if (strings.Contains(line, "Copying") || strings.Contains(line, "Uploading")) && strings.Contains(line, "gs://") {
		return opUpload, true, true
	}

	// Also catch lines that show file operations
	if strings.HasPrefix(line, "Operation completed") {
		return opInfo, true, false
	}

	// Skipped files (both files and directories)
	if strings.Contains(line, "Skipping symbolic link") ||
	   strings.Contains(line, "Skipping symlink") {
		return opSkipped, pkgconfig.GetBool("modules.backup.show_skipped"), true
	}

	// Progress indicators - show but don't count
	if (strings.Contains(line, "[") && strings.Contains(line, "]")) ||
		strings.HasPrefix(line, "At source listing") ||
		strings.HasPrefix(line, "At destination listing") {
		return opProgress, true, false
	}

	// Errors - count
	if strings.Contains(line, "CommandException") ||
		strings.Contains(line, "Error") ||
		strings.Contains(line, "failed") {
		return opError, true, true
	}

	// Building state, warnings - show but don't count
	if strings.HasPrefix(line, "Building synchronization") ||
		strings.HasPrefix(line, "Starting synchronization") ||
		strings.HasPrefix(line, "WARNING:") ||
		strings.HasPrefix(line, "If you experience problems") ||
		strings.HasPrefix(line, "Updates are available") {
		return opInfo, true, false
	}

	// Default - show but don't count
	return opInfo, len(line) > 0, false
}

func printStyledLine(line string, opType operationType) {
	switch opType {
	case opUpload:
		fmt.Printf("%s %s\n", style.BlueStyle.Render("📤"), line)
	case opSkipped:
		if pkgconfig.GetBool("modules.backup.show_skipped") {
			fmt.Printf("%s %s\n", style.YellowStyle.Render("⏭️ "), line)
		}
	case opError:
		fmt.Printf("%s %s\n", style.RedStyle.Render("❌"), line)
	case opProgress:
		fmt.Printf("%s %s\n", style.GreenStyle.Render("⚡"), line)
	case opInfo:
		fmt.Println(line)
	}
}

func processOutputStream(reader io.Reader, logFile *os.File, stats *syncStats, wg *sync.WaitGroup) {
	defer wg.Done()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		// Always write to log file for record keeping
		if logFile != nil {
			fmt.Fprintln(logFile, line)
		}

		opType, shouldPrint, shouldCount := categorizeGsutilLine(line)

		// Update stats if needed
		if shouldCount {
			stats.increment(opType)
		}

		// Print with styling if appropriate
		if shouldPrint {
			printStyledLine(line, opType)
		}
	}

	if err := scanner.Err(); err != nil {
		// Only show error if it's not a normal EOF/close
		if !strings.Contains(err.Error(), "file already closed") {
			fmt.Printf("%s Error reading output: %s\n",
				style.RedStyle.Render("❌"), err)
		}
	}
}

func printSyncSummary(stats *syncStats, logPath string) {
	stats.mu.Lock()
	defer stats.mu.Unlock()

	fmt.Println()

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("212")).
		Padding(0, 1)

	rowStyle := lipgloss.NewStyle().Padding(0, 1)

	header := headerStyle.Render("📊 Sync Summary")
	fmt.Println(header)
	fmt.Println()

	// Stats rows
	if stats.uploaded > 0 {
		fmt.Printf("%s %s: %d\n",
			style.GreenStyle.Render("✓"),
			rowStyle.Render("Uploaded"),
			stats.uploaded)
	}

	fmt.Printf("%s %s: %d\n",
		style.YellowStyle.Render("⏭"),
		rowStyle.Render("Skipped"),
		stats.skipped)

	if stats.failed > 0 {
		fmt.Printf("%s %s: %d\n",
			style.RedStyle.Render("✗"),
			rowStyle.Render("Failed"),
			stats.failed)
	}

	fmt.Println()

	// Log file location
	if logPath != "" {
		fmt.Printf("%s Full log saved to: %s\n",
			style.SubtleStyle.Render("📝"),
			logPath)
		fmt.Println()
	}
}

func Register(parent *cobra.Command) {
	backupCmd := &cobra.Command{
		Use:     "backup",
		Aliases: []string{"bkp"},
		Short:   "Backup ~/Workdir to/from GCS",
		Long:    "Upload ~/Workdir to GCS (default) or download as tar.gz",
		RunE:    runBackup,
	}
	backupCmd.Flags().BoolVar(&uploadFlag, "upload", false, "Upload to GCS (default behavior)")
	backupCmd.Flags().BoolVar(&downloadFlag, "download", false, "Download backup as tar.gz")
	backupCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path for download (default: workdir-backup-YYYYMMDD-HHMMSS.tar.gz)")

	parent.AddCommand(backupCmd)
}

func runBackup(cmd *cobra.Command, args []string) error {
	if downloadFlag {
		return runDownload(cmd, args)
	}
	return runWorkdirSync(cmd, args)
}

func runWorkdirSync(cmd *cobra.Command, args []string) error {
	bucket := getBucket()
	if bucket == "" {
		return fmt.Errorf("GCS bucket not configured. Set it in your config.yaml :\n\nmodules:\n  backup:\n    gcs_bucket: your-bucket-name")
	}

	if !isAuthenticated() {
		fmt.Println(style.YellowStyle.Render("⚠️  No active gcloud authentication found. Initiating login..."))
		if err := login(); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	if !isAuthenticated() {
		return fmt.Errorf("authentication failed or canceled")
	}

	account, err := getActiveAccount()
	if err != nil {
		return err
	}
	fmt.Printf("%s Authenticated as %s\n", style.GreenStyle.Render("✓"), account)

	bucketPath := fmt.Sprintf("gs://%s/", bucket)
	workdir := filepath.Join(os.Getenv("HOME"), "Workdir")

	// Create workdir if it doesn't exist
	if _, err := os.Stat(workdir); os.IsNotExist(err) {
		if err := os.MkdirAll(workdir, 0755); err != nil {
			return fmt.Errorf("failed to create workdir: %w", err)
		}
		fmt.Printf("%s Created directory: %s\n", style.BlueStyle.Render("📁"), workdir)
	}

	// Confirm before starting sync
	fmt.Printf("\n%s About to sync:\n", style.YellowStyle.Render("⚠️ "))
	fmt.Printf("  Source: %s\n", style.BlueStyle.Render(workdir))
	fmt.Printf("  Target: %s\n", style.BlueStyle.Render(bucketPath))
	fmt.Printf("  Note: Will include .git directories\n")
	fmt.Printf("  Note: Will NOT delete remote files\n\n")
	fmt.Print("Continue? (y/N): ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		fmt.Println(style.YellowStyle.Render("Sync cancelled"))
		return nil
	}

	fmt.Printf("\n%s Syncing %s to %s...\n", style.BlueStyle.Render("🔄"), workdir, bucketPath)

	if err := syncToGCS(workdir, bucketPath); err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	fmt.Printf("%s Sync completed successfully!\n", style.GreenStyle.Render("✅"))
	return nil
}

func isAuthenticated() bool {
	cmd := exec.Command("gcloud", "auth", "list", "--filter=status:ACTIVE", "--format=value(account)")
	output, err := cmd.Output()
	return err == nil && len(strings.TrimSpace(string(output))) > 0
}

func login() error {
	cmd := exec.Command("gcloud", "auth", "login")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func getActiveAccount() (string, error) {
	cmd := exec.Command("gcloud", "auth", "list", "--filter=status:ACTIVE", "--format=value(account)")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func syncToGCS(workdir, bucketPath string) error {
	excludePattern := buildExcludePattern()
	parallelProcesses := calculateParallelProcesses()

	// Create log file in tmp directory
	timestamp := time.Now().Format("20060102-150405")
	logPath := filepath.Join(os.TempDir(), fmt.Sprintf("cly-backup-%s.log", timestamp))
	logFile, err := os.Create(logPath)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer logFile.Close()

	fmt.Printf("%s Using %d parallel processes for faster sync...\n",
		style.BlueStyle.Render("⚡"),
		parallelProcesses)
	fmt.Printf("%s Excluding artifacts and dependencies, including .git directories\n",
		style.BlueStyle.Render("📋"))

	args := []string{
		"-m",
		"rsync",
		"-r",
		"-c",
		"-e",
		"-P",
		"-x", excludePattern,
		workdir,
		bucketPath,
	}

	cmd := exec.Command("gsutil", args...)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GSUTIL_PARALLEL_PROCESS_COUNT=%d", parallelProcesses),
		"GSUTIL_COPY_TIMEOUT_SEC=1",
		"GSUTIL_PARALLEL_COMPOSITE_UPLOAD_THRESHOLD=150M",
	)

	// Create pipes for capturing output
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Initialize stats
	stats := &syncStats{}

	// Start command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start gsutil: %w", err)
	}

	// WaitGroup to ensure all output is processed
	var wg sync.WaitGroup
	wg.Add(2)

	// Process stdout
	go processOutputStream(stdoutPipe, logFile, stats, &wg)

	// Process stderr
	go processOutputStream(stderrPipe, logFile, stats, &wg)

	// Wait for all output to be processed first
	wg.Wait()

	// Then wait for command to complete
	cmdErr := cmd.Wait()

	// Print summary
	printSyncSummary(stats, logPath)

	if cmdErr != nil {
		fmt.Printf("%s Some errors occurred during sync\n",
			style.YellowStyle.Render("⚠️ "))
	}

	return nil
}

func buildExcludePattern() string {
	patterns := []string{
		".*node_modules/.*",
		".*__pycache__/.*",
		".*\\.pyc$",
		".*\\.DS_Store$",
		".*\\._\\.DS_Store$",
		".*\\.terraform/.*",
		".*\\.vscode/.*",
		".*\\.npm/.*",
		".*\\.yarn/.*",
		".*venv/.*",
		".*\\.venv/.*",
		".*env/.*",
		".*\\.env/.*",
		".*dist/.*",
		".*build/.*",
		".*target/.*",
		".*bin/.*",
		".*obj/.*",
		".*out/.*",
		".*\\.\\#.*",
		".*~$",
		".*\\.bak$",
		".*\\.bkp$",
		".*\\.log$",
		".*\\.pyo$",
		".*\\.pyd$",
		".*\\.terragrunt-cache/.*",
		".*\\.idea/.*",
		".*\\.cache/.*",
		".*\\.pytest_cache/.*",
		".*\\.mypy_cache/.*",
		".*\\.ruff_cache/.*",
		".*coverage/.*",
		".*\\.coverage$",
		".*htmlcov/.*",
		".*\\.tox/.*",
		".*\\.eggs/.*",
		".*\\.egg-info/.*",
		".*\\.direnv/.*",
		".*tmp/.*",
		".*temp/.*",
		".*\\.tmp$",
		".*\\.temp$",
		".*\\.swp$",
		".*\\.swo$",
		".*\\.swn$",
		".*\\.class$",
		".*\\.o$",
		".*\\.so$",
		".*\\.dylib$",
		".*\\.dll$",
		".*\\.exe$",
	}
	return strings.Join(patterns, "|")
}

func calculateParallelProcesses() int {
	numCores := runtime.NumCPU()
	processes := numCores * 2
	if processes < 4 {
		processes = 4
	}
	if processes > 32 {
		processes = 32
	}
	return processes
}

func getBucket() string {
	bucket := pkgconfig.GetString("modules.backup.gcs_bucket")
	if bucket == "" {
		bucket = os.Getenv("CLY_BACKUP_GCS_BUCKET")
	}
	return bucket
}

func runDownload(cmd *cobra.Command, args []string) error {
	bucket := getBucket()
	if bucket == "" {
		return fmt.Errorf("GCS bucket not configured. Set it in your config.yaml :\n\nmodules:\n  backup:\n    gcs_bucket: your-bucket-name")
	}

	if !isAuthenticated() {
		fmt.Println(style.YellowStyle.Render("⚠️  No active gcloud authentication found. Initiating login..."))
		if err := login(); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	if !isAuthenticated() {
		return fmt.Errorf("authentication failed or canceled")
	}

	account, err := getActiveAccount()
	if err != nil {
		return err
	}
	fmt.Printf("%s Authenticated as %s\n", style.GreenStyle.Render("✓"), account)

	output := outputPath
	if output == "" {
		output = fmt.Sprintf("workdir-backup-%s.tar.gz",
			exec.Command("date", "+%Y%m%d-%H%M%S").String())
		if dateOut, err := exec.Command("date", "+%Y%m%d-%H%M%S").Output(); err == nil {
			output = fmt.Sprintf("workdir-backup-%s.tar.gz", strings.TrimSpace(string(dateOut)))
		}
	}

	bucketPath := fmt.Sprintf("gs://%s/*", bucket)
	tmpDir, err := os.MkdirTemp("", "workdir-download-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	fmt.Printf("%s Downloading from %s...\n", style.BlueStyle.Render("⬇️"), bucketPath)

	downloadCmd := exec.Command("gsutil", "-m", "cp", "-r", bucketPath, tmpDir)
	downloadCmd.Stdout = os.Stdout
	downloadCmd.Stderr = os.Stderr
	if err := downloadCmd.Run(); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	fmt.Printf("%s Creating archive %s...\n", style.BlueStyle.Render("📦"), output)

	tarCmd := exec.Command("tar", "-czf", output, "-C", tmpDir, ".")
	tarCmd.Stdout = os.Stdout
	tarCmd.Stderr = os.Stderr
	if err := tarCmd.Run(); err != nil {
		return fmt.Errorf("archive creation failed: %w", err)
	}

	fileInfo, err := os.Stat(output)
	if err == nil {
		sizeInMB := float64(fileInfo.Size()) / (1024 * 1024)
		fmt.Printf("%s Download completed: %s (%.2f MB)\n",
			style.GreenStyle.Render("✅"),
			output,
			sizeInMB)
	} else {
		fmt.Printf("%s Download completed: %s\n", style.GreenStyle.Render("✅"), output)
	}

	return nil
}
