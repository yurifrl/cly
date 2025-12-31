package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize user config file",
		Long:  "Create ~/.config/cly/config.yaml with default values",
		RunE:  runInit,
	}
}

func runInit(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Check env var for config dir override
	configDir := os.Getenv("CLY_APP_CONFIG_DIR")
	if configDir == "" {
		configDir = filepath.Join(homeDir, ".config/cly")
	}
	configPath := filepath.Join(configDir, "config.yaml")

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Config already exists: %s\n", configPath)
		fmt.Print("Overwrite? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Create directory
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Copy default config
	defaultContent := `app:
  name: cly
  debug: false
  config_dir: ~/.config/cly
  data_dir: ~/.local/share/cly
  dotfiles_dir: ~/DotFiles

theme:
  style: charm # Options: charm | dracula | catppuccin

modules:
  bundle:
    go_file: ~/.config/cly/bundles/Gofile
    js_file: ~/.config/cly/bundles/packages.json
    python_file: ~/.config/cly/bundles/Pythonfile
    brew_file: ~/.config/cly/bundles/Brewfile
    rust_file: ~/.config/cly/bundles/Rustfile
  notify:
    enabled: true
    sound: false
    use_zellij_status: true
    use_zellij_notify: true
    icon: ""
    hooks:
      notification:
        enabled: true
        title: "🔔 Claude Task"
        message: "Starting - New task [${ZELLIJ_SESSION_NAME}] ${CLAUDE_SESSION_NAME}"
        sound: "Glass"
        zellij_status: "🔔 Task notification"
        zellij_event: "notification"
      stop:
        enabled: true
        title: "✅ Claude Complete"
        message: "Finished - Task completed [${ZELLIJ_SESSION_NAME}] ${CLAUDE_SESSION_NAME}"
        sound: "Blow"
        zellij_status: "✅ Task completed"
        zellij_event: "stop"
  uuid:
    default_version: v4 # Options: v4 | v7
  demo:
    show_count: true
  dotfiles:
    zellij_plugins_dir: ~/.config/zellij/plugins
  backup:
    # GCS bucket for backups (configure in config.local.yaml)
    gcs_bucket: ""
    show_skipped: false
  scraper:
    aliexpress:
      browser:
        headless: false
        debug_port: 9222
        user_data_dir: ~/.cly/scraper/chrome
        timeout: 60s
        wait_time: 15s
      reviews_count: 20
      filter_reviews_by: "all"
      output_mode: "single"
      output_dir: "./scraped"
      delay_between_products: 5s
      max_retries: 3
`

	if err := os.WriteFile(configPath, []byte(defaultContent), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("✓ Config initialized: %s\n", configPath)
	return nil
}
