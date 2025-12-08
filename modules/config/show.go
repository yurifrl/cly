package config

import (
	"fmt"

	"github.com/spf13/cobra"
	pkgconfig "github.com/yurifrl/cly/pkg/config"
	"gopkg.in/yaml.v3"
)

func showCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Long:  "Display the merged configuration (defaults + user config + env vars)",
		RunE:  runShow,
	}
}

func runShow(cmd *cobra.Command, args []string) error {
	cfg := pkgconfig.Get()

	yamlBytes, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	fmt.Println(string(yamlBytes))
	return nil
}
