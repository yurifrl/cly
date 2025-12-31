package statusline

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
	pkgconfig "github.com/yurifrl/cly/pkg/config"
)

// Register adds the statusline command and subcommands.
func Register(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "statusline",
		Short: "Claude Code status line",
		Long:  "Outputs status line for Claude Code. Reads JSON from stdin.",
		RunE:  runMain,
	}

	contextCmd := &cobra.Command{
		Use:   "context",
		Short: "Output context window usage",
		RunE:  runContext,
	}

	modelCmd := &cobra.Command{
		Use:   "model",
		Short: "Output model name",
		RunE:  runModel,
	}

	costCmd := &cobra.Command{
		Use:   "cost",
		Short: "Output session cost",
		RunE:  runCost,
	}

	cmd.AddCommand(contextCmd, modelCmd, costCmd)
	parent.AddCommand(cmd)
}

func readInput() (*StatusJSON, error) {
	var input StatusJSON
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		return nil, err
	}
	return &input, nil
}

func runMain(cmd *cobra.Command, args []string) error {
	input, err := readInput()
	if err != nil {
		return err
	}
	cfg := GetConfig()
	out := RenderStatusline(input, cfg)
	fmt.Print(out)
	return nil
}

func runContext(cmd *cobra.Command, args []string) error {
	input, err := readInput()
	if err != nil {
		return err
	}
	fmt.Print(RenderContext(input))
	return nil
}

func runModel(cmd *cobra.Command, args []string) error {
	input, err := readInput()
	if err != nil {
		return err
	}
	fmt.Print(RenderModel(input))
	return nil
}

func runCost(cmd *cobra.Command, args []string) error {
	input, err := readInput()
	if err != nil {
		return err
	}
	fmt.Print(RenderCost(input))
	return nil
}

// RenderModel outputs model name.
func RenderModel(input *StatusJSON) string {
	if input.Model == nil || input.Model.DisplayName == "" {
		return ""
	}
	return fmt.Sprintf("[%s]", input.Model.DisplayName)
}

// RenderCost outputs session cost.
func RenderCost(input *StatusJSON) string {
	if input.Cost == nil {
		return ""
	}
	return fmt.Sprintf("💰 $%.2f", input.Cost.TotalCostUSD)
}

// RenderCustom executes custom command with timeout.
func RenderCustom(input *StatusJSON, cfg CustomConfig) string {
	if !cfg.Enabled || cfg.Command == "" {
		return ""
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 500
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)
	defer cancel()

	// Replace $cwd with actual directory
	command := cfg.Command
	if input.Workspace != nil && input.Workspace.CurrentDir != "" {
		command = strings.ReplaceAll(command, "$cwd", input.Workspace.CurrentDir)
	}

	cmd := exec.CommandContext(ctx, "bash", "-c", command)

	// Pipe input JSON to command
	inputJSON, _ := json.Marshal(input)
	cmd.Stdin = strings.NewReader(string(inputJSON))

	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// ParseFormat extracts variable names from format string.
func ParseFormat(format string) []string {
	re := regexp.MustCompile(`\$(\w+)`)
	matches := re.FindAllStringSubmatch(format, -1)
	var parts []string
	for _, m := range matches {
		if len(m) > 1 {
			parts = append(parts, m[1])
		}
	}
	return parts
}

// RenderStatusline renders full status line from config.
func RenderStatusline(input *StatusJSON, cfg Config) string {
	parts := ParseFormat(cfg.Format)
	var outputs []string

	for _, part := range parts {
		var out string
		switch part {
		case "context":
			if cfg.Context.Enabled {
				out = RenderContext(input)
			}
		case "model":
			if cfg.Model.Enabled {
				out = RenderModel(input)
			}
		case "cost":
			if cfg.Cost.Enabled {
				out = RenderCost(input)
			}
		case "custom":
			if cfg.Custom.Enabled {
				out = RenderCustom(input, cfg.Custom)
			}
		}
		if out != "" {
			outputs = append(outputs, out)
		}
	}

	if len(outputs) == 0 {
		return ""
	}

	// Join with separator (extract from format or default)
	separator := " │ "
	return strings.Join(outputs, separator)
}

// GetConfig returns statusline config from pkg/config.
func GetConfig() Config {
	cfg := pkgconfig.Get()
	if cfg == nil {
		return DefaultConfig()
	}
	slCfg := cfg.GetStatusline()
	return Config{
		Format: slCfg.Format,
		Context: ContextConfig{Enabled: slCfg.Context.Enabled},
		Model:   ModelConfig{Enabled: slCfg.Model.Enabled},
		Cost:    CostConfig{Enabled: slCfg.Cost.Enabled},
		Custom: CustomConfig{
			Enabled: slCfg.Custom.Enabled,
			Command: slCfg.Custom.Command,
			Timeout: slCfg.Custom.Timeout,
		},
	}
}
