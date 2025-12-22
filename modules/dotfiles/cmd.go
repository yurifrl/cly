package dotfiles

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	pkgconfig "github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/style"
)

var (
	installFlag bool
	configFlag  string
)

func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "dotfiles",
		Short: "Manage dotfile symlinks",
		Long:  "Create and manage symlinks from a declarative config file",
		RunE:  runSync,
	}

	cmd.Flags().BoolVarP(&installFlag, "install", "i", false, "Execute install commands (lines starting with !)")
	cmd.PersistentFlags().StringVarP(&configFlag, "config", "c", "", "Path to config file (default: <dotfiles_dir>/dotfiles.conf)")

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show status of all mappings",
		RunE:  runStatus,
	}

	unlinkCmd := &cobra.Command{
		Use:   "unlink",
		Short: "Remove all managed symlinks",
		RunE:  runUnlink,
	}

	cmd.AddCommand(statusCmd, unlinkCmd)
	parent.AddCommand(cmd)
}

func getConfigPath() (string, error) {
	if configFlag != "" {
		return configFlag, nil
	}

	cfg := pkgconfig.Get()
	dotfilesDir := "~/DotFiles"
	if cfg != nil && cfg.App.DotFilesDir != "" {
		dotfilesDir = cfg.App.DotFilesDir
	}
	dotfilesDir = expandTilde(dotfilesDir)

	return filepath.Join(dotfilesDir, "dotfiles.conf"), nil
}

func runSync(cmd *cobra.Command, args []string) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("config not found: %s\nCreate it or use --config /path/to/dotfiles.conf", configPath)
	}

	cfg, err := ParseConfig(configPath)
	if err != nil {
		return err
	}

	for _, e := range cfg.Errors {
		fmt.Printf("⚠️  %s\n", e)
	}

	for _, m := range cfg.Mappings {
		result := CreateSymlink(m)
		printResult(m, result)
	}

	if len(cfg.InstallCommands) > 0 {
		if installFlag {
			for _, cmdStr := range cfg.InstallCommands {
				fmt.Printf("%s %s\n", style.BlueStyle.Render("⚡ Executing:"), cmdStr)
				if err := executeCommand(cmdStr, cfg.BaseDir); err != nil {
					fmt.Printf("  %s %s\n", style.RedStyle.Render("❌"), err)
				}
			}
		} else {
			fmt.Printf("\n%s %d install command(s) skipped (use -i to execute)\n",
				style.YellowStyle.Render("⏭️ "), len(cfg.InstallCommands))
		}
	}

	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("config not found: %s", configPath)
	}

	cfg, err := ParseConfig(configPath)
	if err != nil {
		return err
	}

	fmt.Printf("Dotfiles: %s\n\n", configPath)

	for _, m := range cfg.Mappings {
		result := CheckStatus(m)
		printStatusResult(m, result)
	}

	if len(cfg.InstallCommands) > 0 {
		fmt.Printf("\nInstall commands: %d (use -i to execute)\n", len(cfg.InstallCommands))
	}

	return nil
}

func runUnlink(cmd *cobra.Command, args []string) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("config not found: %s", configPath)
	}

	cfg, err := ParseConfig(configPath)
	if err != nil {
		return err
	}

	removed := 0
	for _, m := range cfg.Mappings {
		if RemoveSymlink(m) {
			fmt.Printf("%s %s\n", style.YellowStyle.Render("🗑️  Removed:"), shortenPath(m.Destination))
			removed++
		}
	}

	fmt.Printf("\n%s Removed %d symlink(s)\n", style.GreenStyle.Render("✅"), removed)
	return nil
}

func printResult(m Mapping, r LinkResult) {
	src := shortenPath(m.Source)
	dest := shortenPath(m.Destination)

	// Print processing line
	fmt.Printf("%s %s -> %s\n",
		style.BlueStyle.Render("🔗 Processing:"),
		src, dest)

	switch r.State {
	case StateLinked:
		if r.CreatedDir {
			fmt.Printf("  %s %s\n",
				style.BlueStyle.Render("📁 Creating directory:"),
				filepath.Dir(dest))
		}
		if r.RemovedExisting {
			fmt.Printf("  %s %s\n",
				style.YellowStyle.Render("🗑️  Removing existing:"),
				dest)
		}
		fmt.Printf("  %s %s -> %s\n",
			style.GreenStyle.Render("✅ Creating symlink:"),
			src, dest)
	case StateMissing:
		fmt.Printf("  %s Source '%s' does not exist, skipping\n",
			style.YellowStyle.Render("⚠️  Warning:"),
			src)
	case StateConflict:
		fmt.Printf("  %s %s\n",
			style.RedStyle.Render("❌ Error:"),
			r.Error)
	case StateError:
		fmt.Printf("  %s %s\n",
			style.RedStyle.Render("❌ Error:"),
			r.Error)
	}
}

func printStatusResult(m Mapping, r LinkResult) {
	dest := shortenPath(m.Destination)
	src := shortenPath(m.Source)
	switch r.State {
	case StateLinked:
		fmt.Printf("%s %-40s → %s\n", style.GreenStyle.Render("✓"), dest, src)
	case StateUnlinked:
		fmt.Printf("%s %-40s → %s\n", style.SubtleStyle.Render("○"), dest, src)
	case StateMissing:
		fmt.Printf("%s %-40s (source missing)\n", style.YellowStyle.Render("⚠️ "), dest)
	case StateConflict:
		fmt.Printf("%s %-40s (conflict)\n", style.RedStyle.Render("✗"), dest)
	case StateBroken:
		fmt.Printf("%s %-40s (broken symlink)\n", style.RedStyle.Render("✗"), dest)
	}
}

func shortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

func executeCommand(cmdStr, baseDir string) error {
	if strings.HasPrefix(cmdStr, "zellij_plugin ") {
		url := strings.TrimPrefix(cmdStr, "zellij_plugin ")
		url = strings.TrimSpace(url)
		return downloadZellijPlugin(url)
	}

	cmd := exec.Command("bash", "-c", cmdStr)
	cmd.Dir = baseDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
