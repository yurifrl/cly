package envs

import (
	"context"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
)

func Register(parent *cobra.Command) {
	var configPath, profile, opBinary string
	var launchctl, fish bool

	cmd := &cobra.Command{
		Use:   "envs",
		Short: "Load environment variables from 1Password",
		Long:  "Fetches secrets from 1Password in parallel and outputs environment variables to stdout.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context(), configPath, profile, opBinary, launchctl, fish)
		},
	}
	cmd.Flags().StringVar(&configPath, "config", defaultConfigPath(), "Path to environment loader JSON config")
	cmd.Flags().StringVar(&profile, "profile", "", "Environment profile: all, work, or personal")
	cmd.Flags().StringVar(&opBinary, "op", "op", "Path to the 1Password CLI")
	cmd.Flags().BoolVar(&launchctl, "launchctl", false, "Inject vars via launchctl setenv (available to all GUI apps)")
	cmd.Flags().BoolVar(&fish, "fish", false, "Output fish-compatible set -gx format")
	registerInstallApp(cmd)
	parent.AddCommand(cmd)
}

func run(ctx context.Context, configPath, profile, opBinary string, launchctl, fish bool) error {
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

	tokens, err := signIn(ctx, opBinary, config.Secrets)
	if err != nil {
		return err
	}

	mdl := newModel(ctx, config, profile, opBinary, tokens, launchctl, fish)
	if !isTerminal() {
		return mdl.runPlain()
	}
	program := tea.NewProgram(mdl)
	finalModel, err := program.Run()
	if err != nil {
		return err
	}
	m := finalModel.(model)
	writeOutput(os.Stdout, m.fields, m.fish)
	return nil
}

func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "1pass-load-envs.json"
	}
	return fmt.Sprintf("%s/.config/1pass-load-envs.json", home)
}

func isTerminal() bool {
	info, err := os.Stdout.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
