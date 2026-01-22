package mcp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var (
	contextFlag string
	aiFlag      string
	scopeFlag   string
)

func Register(parent *cobra.Command) {
	cmd := NewRootCmd()
	parent.AddCommand(cmd)
}

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "MCP server manager for AI tools",
		Long:  "Manage MCP servers for Claude Code, Cursor IDE, and Claude Desktop",
		RunE:  runTUI,
	}

	cmd.PersistentFlags().StringVar(&contextFlag, "context", "", "Context: <ai>:<scope> (e.g., claude:user)")

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newSwitchCmd())
	cmd.AddCommand(newAddCmd())
	cmd.AddCommand(newRemoveCmd())
	cmd.AddCommand(newDoctorCmd())
	cmd.AddCommand(newContextCmd())
	cmd.AddCommand(newApplyCmd())
	cmd.AddCommand(newCompletionCmd())

	return cmd
}

func parseContextFlag() {
	if contextFlag != "" {
		parts := strings.Split(contextFlag, ":")
		if len(parts) == 2 {
			aiFlag = parts[0]
			scopeFlag = parts[1]
		}
	}
}

func runTUI(cmd *cobra.Command, args []string) error {
	parseContextFlag()
	return launchTUI(aiFlag, scopeFlag)
}

func newApplyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "apply",
		Short: "Launch interactive TUI",
		RunE:  runTUI,
	}
}

func newListCmd() *cobra.Command {
	var long, all bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List MCPs with status",
		RunE: func(cmd *cobra.Command, args []string) error {
			parseContextFlag()
			return runList(long, all)
		},
	}

	cmd.Flags().BoolVarP(&long, "long", "l", false, "Detailed list with tags and presets")
	cmd.Flags().BoolVarP(&all, "all", "a", false, "Show all MCPs (installed + available)")

	return cmd
}

func newSwitchCmd() *cobra.Command {
	var forceOn, forceOff, switchAll bool
	var preset, tag string

	cmd := &cobra.Command{
		Use:   "switch [mcp...]",
		Short: "Toggle MCPs on/off",
		RunE: func(cmd *cobra.Command, args []string) error {
			parseContextFlag()
			return runSwitch(args, forceOn, forceOff, switchAll, preset, tag)
		},
	}

	cmd.Flags().BoolVar(&forceOn, "on", false, "Force enable")
	cmd.Flags().BoolVar(&forceOff, "off", false, "Force disable")
	cmd.Flags().BoolVar(&switchAll, "all", false, "Apply to all MCPs in catalog")
	cmd.Flags().StringVarP(&preset, "preset", "p", "", "Switch all MCPs in preset")
	cmd.Flags().StringVarP(&tag, "tag", "t", "", "Switch all MCPs with tag")

	return cmd
}

func newAddCmd() *cobra.Command {
	var transport, tags, desc, targetFile string
	var envVars, headers []string

	cmd := &cobra.Command{
		Use:   "add <name> <commandOrUrl> [args...]",
		Short: "Add MCP to sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("name required")
			}
			return runAdd(args, transport, tags, desc, targetFile, envVars, headers)
		},
	}

	cmd.Flags().StringVarP(&transport, "transport", "t", "stdio", "Transport type (stdio, sse, http)")
	cmd.Flags().StringSliceVarP(&envVars, "env", "e", nil, "Environment variables (KEY=value)")
	cmd.Flags().StringSliceVarP(&headers, "header", "H", nil, "Headers (KEY: value)")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tags")
	cmd.Flags().StringVar(&desc, "description", "", "MCP description")
	cmd.Flags().StringVar(&targetFile, "file", "", "Target source file")

	return cmd
}

func newRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove MCP from sources",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(args[0])
		},
	}
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Run system health check",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor()
		},
	}
}

func newContextCmd() *cobra.Command {
	var showOnly, whichOnly bool

	cmd := &cobra.Command{
		Use:   "context [ai:scope]",
		Short: "Set or show default context",
		RunE: func(cmd *cobra.Command, args []string) error {
			if whichOnly {
				return showContextPath()
			}
			if showOnly || len(args) == 0 {
				return showContext()
			}
			return setContext(args[0])
		},
	}

	cmd.Flags().BoolVar(&showOnly, "show", false, "Show current default context")
	cmd.Flags().BoolVar(&whichOnly, "which", false, "Show config file path")

	return cmd
}

func newCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "completion <bash|zsh|fish>",
		Short: "Generate shell completion",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return generateCompletion(args[0])
		},
	}
}

func launchTUI(ai, scope string) error {
	globalCfg, err := LoadGlobalConfig()
	if err != nil {
		return fmt.Errorf("loading global config: %w", err)
	}

	projectCfg, err := LoadProjectConfig()
	if err != nil {
		return fmt.Errorf("loading project config: %w", err)
	}

	detector := NewDetector(globalCfg, projectCfg, ai, scope)
	ctx, err := detector.DetectContext()
	if err != nil {
		return fmt.Errorf("detecting context: %w", err)
	}
	contextSource := detector.GetContextSource()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}
	libPath := filepath.Join(homeDir, ".config", "mcpcli")
	catalog, err := LoadCatalogWithSources(libPath, globalCfg.SourcePaths)
	if err != nil {
		return fmt.Errorf("loading MCP sources: %w", err)
	}

	adapter, err := getAdapter(ctx.AI)
	if err != nil {
		return err
	}

	model := NewModel(catalog, globalCfg, ctx, contextSource, adapter)
	p := tea.NewProgram(model, tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("running TUI: %w", err)
	}

	return nil
}

func getAdapter(ai string) (Adapter, error) {
	switch ai {
	case "claude":
		return &ClaudeAdapter{}, nil
	case "cursor":
		return &CursorAdapter{}, nil
	case "desktop":
		return &DesktopAdapter{}, nil
	default:
		return nil, fmt.Errorf("unsupported AI tool: %s (available: claude, cursor, desktop)", ai)
	}
}

func runList(long, all bool) error {
	globalCfg, err := LoadGlobalConfig()
	if err != nil {
		return fmt.Errorf("loading global config: %w", err)
	}

	homeDir, _ := os.UserHomeDir()
	libPath := filepath.Join(homeDir, ".config", "mcpcli")
	catalog, err := LoadCatalogWithSources(libPath, globalCfg.SourcePaths)
	if err != nil {
		return fmt.Errorf("loading catalog: %w", err)
	}

	projectCfg, _ := LoadProjectConfig()
	detector := NewDetector(globalCfg, projectCfg, aiFlag, scopeFlag)
	ctx, err := detector.DetectContext()
	if err != nil {
		return fmt.Errorf("detecting context: %w", err)
	}

	adapter, err := getAdapter(ctx.AI)
	if err != nil {
		return err
	}

	toolCfg, err := adapter.ReadConfig(ctx.Scope)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	installed := make(map[string]bool)
	for name := range toolCfg.MCPServers {
		installed[name] = true
	}

	mcpPresets := make(map[string][]string)
	for presetName, mcpNames := range globalCfg.Presets {
		for _, mcpName := range mcpNames {
			mcpPresets[mcpName] = append(mcpPresets[mcpName], presetName)
		}
	}

	configPath, _ := adapter.GetConfigPath(ctx.Scope)
	fmt.Printf("%s:%s (%s)\n\n", ctx.AI, ctx.Scope, configPath)

	allMCPs := catalog.GetAll()

	var mcpsToShow []MCP
	for _, mcp := range allMCPs {
		if all || installed[mcp.Name] {
			mcpsToShow = append(mcpsToShow, mcp)
		}
	}

	if len(mcpsToShow) == 0 {
		fmt.Println("No MCPs installed.")
		fmt.Println("  Use 'mcp list -a' to see all available MCPs")
		return nil
	}

	if long {
		for _, mcp := range mcpsToShow {
			status := "o"
			if installed[mcp.Name] {
				status = "*"
			}
			fmt.Printf("%s %s\n", status, mcp.Name)
			if mcp.Description != "" {
				fmt.Printf("    %s\n", mcp.Description)
			}
			if len(mcp.Tags) > 0 {
				fmt.Printf("    tags: %s\n", strings.Join(mcp.Tags, ", "))
			}
			if presets, ok := mcpPresets[mcp.Name]; ok {
				fmt.Printf("    presets: %s\n", strings.Join(presets, ", "))
			}
			fmt.Println()
		}
	} else if all {
		var enabledMCPs, disabledMCPs []string
		for _, mcp := range mcpsToShow {
			if installed[mcp.Name] {
				enabledMCPs = append(enabledMCPs, mcp.Name)
			} else {
				disabledMCPs = append(disabledMCPs, mcp.Name)
			}
		}

		if len(enabledMCPs) > 0 {
			fmt.Printf("* Enabled (%d):\n", len(enabledMCPs))
			for _, name := range enabledMCPs {
				fmt.Printf("  %s\n", name)
			}
		}

		if len(disabledMCPs) > 0 {
			if len(enabledMCPs) > 0 {
				fmt.Println()
			}
			fmt.Printf("o Available (%d):\n", len(disabledMCPs))
			for _, name := range disabledMCPs {
				fmt.Printf("  %s\n", name)
			}
		}
	} else {
		for _, mcp := range mcpsToShow {
			fmt.Printf("  %s\n", mcp.Name)
		}
	}

	return nil
}

func runSwitch(args []string, forceOn, forceOff, switchAll bool, preset, tag string) error {
	if forceOn && forceOff {
		return fmt.Errorf("cannot use both --on and --off")
	}

	if switchAll {
		if preset != "" || tag != "" {
			return fmt.Errorf("cannot use --all with --preset or --tag")
		}
		if !forceOn && !forceOff {
			return fmt.Errorf("--all requires --on or --off")
		}
	}

	globalCfg, err := LoadGlobalConfig()
	if err != nil {
		return fmt.Errorf("loading global config: %w", err)
	}

	homeDir, _ := os.UserHomeDir()
	libPath := filepath.Join(homeDir, ".config", "mcpcli")
	catalog, err := LoadCatalogWithSources(libPath, globalCfg.SourcePaths)
	if err != nil {
		return fmt.Errorf("loading catalog: %w", err)
	}

	var mcpNames []string

	if switchAll {
		allMCPs := catalog.GetAll()
		if len(allMCPs) == 0 {
			return fmt.Errorf("no MCPs found in catalog")
		}
		for _, mcp := range allMCPs {
			mcpNames = append(mcpNames, mcp.Name)
		}
	}

	if preset != "" && !switchAll {
		presetMCPs, ok := globalCfg.Presets[preset]
		if !ok {
			return fmt.Errorf("preset not found: %s", preset)
		}
		mcpNames = append(mcpNames, presetMCPs...)
	}

	if tag != "" && !switchAll {
		taggedMCPs := catalog.Filter("", []string{tag})
		if len(taggedMCPs) == 0 {
			return fmt.Errorf("no MCPs found with tag: %s", tag)
		}
		for _, mcp := range taggedMCPs {
			mcpNames = append(mcpNames, mcp.Name)
		}
	}

	if !switchAll {
		mcpNames = append(mcpNames, args...)
	}

	if len(mcpNames) == 0 {
		return fmt.Errorf("no MCPs specified")
	}

	for _, name := range mcpNames {
		if _, ok := catalog.Get(name); !ok {
			return fmt.Errorf("MCP not found in catalog: %s", name)
		}
	}

	projectCfg, _ := LoadProjectConfig()
	detector := NewDetector(globalCfg, projectCfg, aiFlag, scopeFlag)
	ctx, err := detector.DetectContext()
	if err != nil {
		return fmt.Errorf("detecting context: %w", err)
	}

	adapter, err := getAdapter(ctx.AI)
	if err != nil {
		return err
	}

	toolCfg, err := adapter.ReadConfig(ctx.Scope)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	installed := make(map[string]bool)
	for name := range toolCfg.MCPServers {
		installed[name] = true
	}

	var enabled, disabled []string

	for _, name := range mcpNames {
		isInstalled := installed[name]

		var shouldEnable bool
		if forceOn {
			shouldEnable = true
		} else if forceOff {
			shouldEnable = false
		} else {
			shouldEnable = !isInstalled
		}

		if shouldEnable && !isInstalled {
			installed[name] = true
			enabled = append(enabled, name)
		} else if !shouldEnable && isInstalled {
			delete(installed, name)
			disabled = append(disabled, name)
		}
	}

	var finalMCPs []MCP
	for name := range installed {
		mcp, _ := catalog.Get(name)
		finalMCPs = append(finalMCPs, mcp)
	}

	if err := adapter.WriteConfig(ctx.Scope, finalMCPs); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	configPath, _ := adapter.GetConfigPath(ctx.Scope)
	fmt.Printf("%s:%s (%s)\n", ctx.AI, ctx.Scope, configPath)

	if len(enabled) > 0 {
		fmt.Printf("+ Enabled: %s\n", strings.Join(enabled, ", "))
	}
	if len(disabled) > 0 {
		fmt.Printf("- Disabled: %s\n", strings.Join(disabled, ", "))
	}
	if len(enabled) == 0 && len(disabled) == 0 {
		fmt.Println("No changes (already in desired state)")
	}

	return nil
}

func runAdd(args []string, transport, tags, desc, targetFile string, envVars, headers []string) error {
	name := args[0]

	var commandOrURL string
	var cmdArgs []string
	if len(args) > 1 {
		commandOrURL = args[1]
		cmdArgs = args[2:]
	}

	envMap := make(map[string]string)
	for _, e := range envVars {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	headerMap := make(map[string]string)
	for _, h := range headers {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			headerMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	var tagList []string
	if tags != "" {
		tagList = strings.Split(tags, ",")
		for i := range tagList {
			tagList[i] = strings.TrimSpace(tagList[i])
		}
	}

	opts := AddMCPOptions{
		Name:        name,
		Type:        transport,
		Description: desc,
		Tags:        tagList,
		TargetFile:  targetFile,
		Env:         envMap,
		Headers:     headerMap,
	}

	if transport == "stdio" {
		opts.Command = commandOrURL
		opts.Args = cmdArgs
	} else {
		opts.URL = commandOrURL
	}

	globalCfg, _ := LoadGlobalConfig()
	homeDir, _ := os.UserHomeDir()
	libPath := filepath.Join(homeDir, ".config", "mcpcli")
	catalog, err := LoadCatalogWithSources(libPath, globalCfg.SourcePaths)
	if err != nil {
		return fmt.Errorf("loading source catalog: %w", err)
	}

	if err := catalog.AddMCP(opts); err != nil {
		return fmt.Errorf("adding MCP: %w", err)
	}

	fmt.Printf("+ Added %s to sources\n", name)
	return nil
}

func runRemove(name string) error {
	globalCfg, _ := LoadGlobalConfig()
	homeDir, _ := os.UserHomeDir()
	libPath := filepath.Join(homeDir, ".config", "mcpcli")
	catalog, err := LoadCatalogWithSources(libPath, globalCfg.SourcePaths)
	if err != nil {
		return fmt.Errorf("loading source catalog: %w", err)
	}

	if err := catalog.RemoveMCP(name); err != nil {
		return fmt.Errorf("removing MCP: %w", err)
	}

	fmt.Printf("- Removed %s from sources\n", name)
	return nil
}

func runDoctor() error {
	fmt.Println("MCP Health Check")

	checks := []struct {
		name string
		fn   func() (bool, string)
	}{
		{"Config files", checkConfigFiles},
		{"Source catalog", checkSourceCatalog},
		{"AI tools installed", checkAITools},
		{"File permissions", checkPermissions},
	}

	passed := 0
	failed := 0
	warned := 0

	for _, check := range checks {
		ok, msg := check.fn()
		status := "+"
		if strings.HasPrefix(msg, "WARNING:") {
			status = "!"
			warned++
		} else if !ok {
			status = "x"
			failed++
		} else {
			passed++
		}
		fmt.Printf("%s %s: %s\n", status, check.name, strings.TrimPrefix(msg, "WARNING: "))
	}

	fmt.Printf("\n%d passed", passed)
	if warned > 0 {
		fmt.Printf(", %d warnings", warned)
	}
	if failed > 0 {
		fmt.Printf(", %d failed", failed)
	}
	fmt.Println()

	return nil
}

func checkConfigFiles() (bool, string) {
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".config", "mcpcli", "config.yaml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return false, "Global config missing"
	}

	cfg, err := LoadGlobalConfig()
	if err != nil {
		return false, fmt.Sprintf("Can't parse: %v", err)
	}

	return true, fmt.Sprintf("Valid (default: %s/%s)", cfg.Defaults.AI, cfg.Defaults.Scope)
}

func checkSourceCatalog() (bool, string) {
	globalCfg, _ := LoadGlobalConfig()
	homeDir, _ := os.UserHomeDir()
	mcpsPath := filepath.Join(homeDir, ".config", "mcpcli")

	catalog, err := LoadCatalogWithSources(mcpsPath, globalCfg.SourcePaths)
	if err != nil {
		return false, fmt.Sprintf("Can't load: %v", err)
	}

	count := len(catalog.GetAll())
	if count == 0 {
		return false, "No MCPs in catalog"
	}

	return true, fmt.Sprintf("%d MCPs available", count)
}

func checkAITools() (bool, string) {
	tools := []struct {
		name    string
		adapter Adapter
	}{
		{"Claude Code", &ClaudeAdapter{}},
		{"Cursor IDE", &CursorAdapter{}},
		{"Claude Desktop", &DesktopAdapter{}},
	}

	var installed []string
	for _, tool := range tools {
		if tool.adapter.IsInstalled() {
			installed = append(installed, tool.name)
		}
	}

	if len(installed) == 0 {
		return true, "WARNING: No AI tools detected"
	}

	return true, fmt.Sprintf("%d detected: %v", len(installed), installed)
}

func checkPermissions() (bool, string) {
	homeDir, _ := os.UserHomeDir()
	paths := []string{
		filepath.Join(homeDir, ".config", "mcpcli"),
		filepath.Join(homeDir, ".config", "mcpcli", "mcps"),
	}

	for _, path := range paths {
		if info, err := os.Stat(path); err == nil {
			if info.Mode().Perm()&0200 == 0 {
				return false, fmt.Sprintf("%s not writable", path)
			}
		}
	}

	return true, "All directories writable"
}

func showContext() error {
	cfg, _ := LoadGlobalConfig()
	fmt.Printf("Default context: %s:%s\n", cfg.Defaults.AI, cfg.Defaults.Scope)
	return nil
}

func showContextPath() error {
	cfg, err := LoadGlobalConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	adapter, err := getAdapter(cfg.Defaults.AI)
	if err != nil {
		return err
	}

	path, err := adapter.GetConfigPath(cfg.Defaults.Scope)
	if err != nil {
		return fmt.Errorf("getting config path: %w", err)
	}

	fmt.Println(path)
	return nil
}

func setContext(contextStr string) error {
	parts := strings.Split(contextStr, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid format, use: ai:scope (e.g., cursor:project)")
	}

	ai, scope := parts[0], parts[1]

	validAI := map[string]bool{"claude": true, "cursor": true, "desktop": true}
	validScope := map[string]bool{"user": true, "project": true, "local": true}

	if !validAI[ai] {
		return fmt.Errorf("invalid AI: %s (available: claude, cursor, desktop)", ai)
	}
	if !validScope[scope] {
		return fmt.Errorf("invalid scope: %s (available: user, project, local)", scope)
	}

	cfg, err := LoadGlobalConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	cfg.Defaults.AI = ai
	cfg.Defaults.Scope = scope

	if err := SaveGlobalConfig(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("+ Default context set to: %s (%s)\n", ai, scope)

	adapter, _ := getAdapter(ai)
	if adapter != nil {
		path, _ := adapter.GetConfigPath(scope)
		fmt.Printf("Config file: %s\n", path)
	}

	return nil
}

func generateCompletion(shell string) error {
	switch shell {
	case "bash":
		printBashCompletion()
	case "zsh":
		printZshCompletion()
	case "fish":
		printFishCompletion()
	default:
		return fmt.Errorf("unknown shell: %s (available: bash, zsh, fish)", shell)
	}
	return nil
}

func printBashCompletion() {
	fmt.Println(`_mcp() {
    local cur prev
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    case "${COMP_WORDS[1]}" in
        switch)
            COMPREPLY=($(compgen -W "--on --off --all -p --preset -t --tag" -- "${cur}"))
            ;;
        list)
            COMPREPLY=($(compgen -W "-l --long -a --all" -- "${cur}"))
            ;;
        completion)
            COMPREPLY=($(compgen -W "bash zsh fish" -- "${cur}"))
            ;;
        *)
            COMPREPLY=($(compgen -W "add remove switch list apply completion doctor context" -- "${cur}"))
            ;;
    esac
}
complete -F _mcp mcp`)
}

func printZshCompletion() {
	fmt.Println(`#compdef mcp
_mcp() {
    local -a commands
    commands=(
        'add:Add MCP to sources'
        'remove:Remove MCP from sources'
        'switch:Toggle MCPs on/off'
        'list:List MCPs with status'
        'apply:Launch interactive TUI'
        'completion:Generate shell completion'
        'doctor:Run system health check'
        'context:Set or show default context'
    )
    _describe 'command' commands
}
_mcp`)
}

func printFishCompletion() {
	fmt.Println(`complete -c mcp -f
complete -c mcp -n '__fish_use_subcommand' -a 'add' -d 'Add MCP to sources'
complete -c mcp -n '__fish_use_subcommand' -a 'remove' -d 'Remove MCP from sources'
complete -c mcp -n '__fish_use_subcommand' -a 'switch' -d 'Toggle MCPs on/off'
complete -c mcp -n '__fish_use_subcommand' -a 'list' -d 'List MCPs with status'
complete -c mcp -n '__fish_use_subcommand' -a 'apply' -d 'Launch interactive TUI'
complete -c mcp -n '__fish_use_subcommand' -a 'completion' -d 'Generate shell completion'
complete -c mcp -n '__fish_use_subcommand' -a 'doctor' -d 'Run system health check'
complete -c mcp -n '__fish_use_subcommand' -a 'context' -d 'Set or show default context'
complete -c mcp -n '__fish_seen_subcommand_from switch' -l on -d 'Force enable'
complete -c mcp -n '__fish_seen_subcommand_from switch' -l off -d 'Force disable'
complete -c mcp -n '__fish_seen_subcommand_from list' -s l -l long -d 'Detailed list'
complete -c mcp -n '__fish_seen_subcommand_from list' -s a -l all -d 'Show all MCPs'
complete -c mcp -n '__fish_seen_subcommand_from completion' -xa 'bash zsh fish'`)
}

// Ensure commands are available (for doctor check)
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
