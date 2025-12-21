package backup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	pkgconfig "github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/style"
)

var (
	gcpFlag    bool
	outputPath string
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Backup operations",
		Long:  "Backup operations for various directories and services",
	}

	workdirCmd := &cobra.Command{
		Use:   "workdir",
		Short: "Backup ~/Workdir",
		Long:  "Backup ~/Workdir to Google Cloud Storage",
		RunE:  runWorkdirBackup,
	}

	downloadCmd := &cobra.Command{
		Use:   "download",
		Short: "Download backup as tar.gz",
		Long:  "Download everything from GCS backup as a compressed tar.gz archive",
		RunE:  runDownload,
	}

	workdirCmd.Flags().BoolVar(&gcpFlag, "gcp", true, "Backup to GCP (default)")
	downloadCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path (default: workdir-backup-YYYYMMDD-HHMMSS.tar.gz)")
	downloadCmd.Flags().BoolVar(&gcpFlag, "gcp", true, "Download from GCP (default)")

	cmd.AddCommand(workdirCmd, downloadCmd)
	parent.AddCommand(cmd)
}

func runWorkdirBackup(cmd *cobra.Command, args []string) error {
	if !gcpFlag {
		return fmt.Errorf("only GCP backup is currently supported")
	}

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

	fmt.Printf("%s Backing up %s to %s...\n", style.BlueStyle.Render("🔄"), workdir, bucketPath)

	if err := syncToGCS(workdir, bucketPath); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	fmt.Printf("%s Backup completed successfully!\n", style.GreenStyle.Render("✅"))
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

	fmt.Printf("%s Using %d parallel processes for faster sync...\n",
		style.BlueStyle.Render("⚡"),
		parallelProcesses)
	fmt.Printf("%s Excluding artifacts and dependencies, keeping git history\n",
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
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("%s Some errors occurred during sync, but continuing...\n",
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
		".*\\.git/objects/.*",
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
	if !gcpFlag {
		return fmt.Errorf("only GCP download is currently supported")
	}

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
