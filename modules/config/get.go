package config

import (
	"fmt"

	"github.com/spf13/cobra"
	pkgconfig "github.com/yurifrl/cly/pkg/config"
)

func getCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get config value",
		Long:  "Get a specific configuration value by key (e.g., app.name, theme.style)",
		Args:  cobra.ExactArgs(1),
		RunE:  runGet,
	}
}

func runGet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := pkgconfig.GetString(key)

	if value == "" {
		return fmt.Errorf("key not found: %s", key)
	}

	fmt.Println(value)
	return nil
}
