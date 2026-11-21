package claude

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// The claude-cli-restore Tern plugin ships inside the cly binary, so it
// installs without a source checkout and always matches the cly that
// writes the pane records its host half reads.
//
//go:embed tern-plugin/plugin.toml tern-plugin/host.luau tern-plugin/window.luau
var ternPluginFS embed.FS

const ternPluginID = "claude-cli-restore"

func ternPluginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claude-tern-plugin",
		Short: "Install or remove the claude-cli-restore Tern plugin",
		Long: "The claude-cli-restore Tern plugin, embedded in cly: it resumes `c --cli` claude panes after the Tern " +
			"session daemon restarts and shows the Anthropic icon for c/claude panes.\n\n" +
			"Resuming needs the `cly claude-tern-pane` SessionStart/SessionEnd hook in Claude Code's settings.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error { return cmd.Help() },
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "install",
			Short: "Install the plugin into Tern (replacing an installed or linked copy)",
			Args:  cobra.NoArgs,
			RunE:  runTernPluginInstall,
		},
		&cobra.Command{
			Use:   "uninstall",
			Short: "Remove the plugin from Tern",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				if err := ternPlugin("remove", ternPluginID); err != nil {
					// A linked copy is unlinked, not removed.
					if err2 := ternPlugin("unlink", ternPluginID); err2 != nil {
						return err
					}
				}
				fmt.Printf("✅ removed %s\n", ternPluginID)
				return nil
			},
		},
		&cobra.Command{
			Use:   "extract <dir>",
			Short: "Write the plugin's files into <dir> (to read them or link them by hand)",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				if err := extractTernPlugin(args[0]); err != nil {
					return err
				}
				fmt.Printf("✅ wrote %s to %s\n", ternPluginID, args[0])
				return nil
			},
		},
	)
	return cmd
}

func runTernPluginInstall(cmd *cobra.Command, args []string) error {
	if _, err := exec.LookPath("tern"); err != nil {
		return fmt.Errorf("tern not found on PATH")
	}
	tmp, err := os.MkdirTemp("", "cly-tern-plugin-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	dir := filepath.Join(tmp, ternPluginID)
	if err := extractTernPlugin(dir); err != nil {
		return err
	}
	// `install --force` doesn't replace a linked copy; drop it first.
	_ = ternPlugin("unlink", ternPluginID)
	if err := ternPlugin("install", dir, "--force"); err != nil {
		return err
	}
	fmt.Printf("✅ installed %s\n", ternPluginID)
	if !ternPaneHookConfigured() {
		fmt.Println("⚠️  `cly claude-tern-pane` isn't a hook in ~/.claude/settings.json; panes get the icon but won't resume.")
		fmt.Println(`   Add it under hooks.SessionStart and hooks.SessionEnd: {"type": "command", "command": "cly claude-tern-pane"}`)
	}
	return nil
}

func extractTernPlugin(dir string) error {
	sub, err := fs.Sub(ternPluginFS, "tern-plugin")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return fs.WalkDir(sub, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(sub, path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, path), data, 0o644)
	})
}

func ternPlugin(args ...string) error {
	out, err := exec.Command("tern", append([]string{"plugin"}, args...)...).CombinedOutput()
	if err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return fmt.Errorf("tern plugin %s: %s", args[0], msg)
		}
		return fmt.Errorf("tern plugin %s: %w", args[0], err)
	}
	return nil
}

func ternPaneHookConfigured() bool {
	home, _ := os.UserHomeDir()
	data, err := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	return err == nil && strings.Contains(string(data), "claude-tern-pane")
}
