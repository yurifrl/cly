package envs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	var configPath, cachePath, profile, opBinary string
	var reload, plain bool

	cmd := &cobra.Command{
		Use:   "envs",
		Short: "Load environment variables from 1Password",
		Long:  "A standalone 1Password environment loader with parallel fetches and a live terminal progress view.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context(), configPath, cachePath, profile, opBinary, reload, plain)
		},
	}
	cmd.Flags().StringVar(&configPath, "config", defaultConfigPath(), "Path to environment loader JSON config")
	cmd.Flags().StringVar(&cachePath, "cache", defaultCacheDir(), "Directory for generated shell caches")
	cmd.Flags().StringVar(&profile, "profile", "", "Environment profile: all, work, or personal")
	cmd.Flags().StringVar(&opBinary, "op", "op", "Path to the 1Password CLI")
	cmd.Flags().BoolVar(&reload, "reload", false, "Refresh the 1Password cache")
	cmd.Flags().BoolVar(&plain, "plain", false, "Disable the interactive terminal interface")
	parent.AddCommand(cmd)
}

func run(ctx context.Context, configPath, cachePath, profile, opBinary string, reload, plain bool) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	config, err := ParseConfig(data)
	if err != nil {
		return err
	}
	if profile == "" {
		profile = config.DefaultProfile
	}
	if profile != "all" && profile != "work" && profile != "personal" {
		return fmt.Errorf("invalid profile %q: expected all, work, or personal", profile)
	}
	if err := os.MkdirAll(cachePath, 0o700); err != nil {
		return fmt.Errorf("create cache: %w", err)
	}
	cacheFile := filepath.Join(cachePath, "envs-"+profile+".fish")
	if !reload {
		if data, err := os.ReadFile(cacheFile); err == nil {
			fmt.Print(string(data))
			return nil
		}
	}

	tokens, err := signIn(ctx, opBinary, config.Secrets)
	if err != nil {
		return err
	}

	model := newModel(ctx, config, profile, opBinary, cacheFile, tokens)
	if plain || !isTerminal() {
		return model.runPlain()
	}
	program := tea.NewProgram(model)
	_, err = program.Run()
	return err
}

func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "1pass-load-envs.json"
	}
	return filepath.Join(home, ".config", "1pass-load-envs.json")
}

func defaultCacheDir() string {
	return "/tmp/1pass-load-envs"
}

func isTerminal() bool {
	info, err := os.Stdout.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
