package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	pkgconfig "github.com/yurifrl/cly/pkg/config"
)

func setCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set config value",
		Long:  "Set a configuration value in user config file (e.g., theme.style dracula)",
		Args:  cobra.ExactArgs(2),
		RunE:  runSet,
	}
}

func runSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	if err := pkgconfig.Set(key, value); err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}

	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".config/cly/config.yaml")

	fmt.Printf("✓ Set %s = %s\n", key, value)
	fmt.Printf("  Config: %s\n", configPath)
	return nil
}
