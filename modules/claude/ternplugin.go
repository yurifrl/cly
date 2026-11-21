package claude

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Install the plugin, or update it when cly carries a newer version",
		Long: "Extracts the plugin to " + "~/.local/share/cly/tern-plugins/" + ternPluginID + " and links it into Tern. " +
			"Does nothing when that version is already linked; --force rewrites it anyway.",
		Args: cobra.NoArgs,
		RunE: runTernPluginInstall,
	}
	installCmd.Flags().Bool("force", false, "reinstall even when the version is current")
	cmd.AddCommand(
		installCmd,
		&cobra.Command{
			Use:   "uninstall",
			Short: "Remove the plugin from Tern",
			Args:  cobra.NoArgs,
			RunE:  runTernPluginUninstall,
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
	force, _ := cmd.Flags().GetBool("force")
	if _, err := exec.LookPath("tern"); err != nil {
		return fmt.Errorf("tern not found on PATH")
	}
	want, err := ternPluginVersion(ternPluginFS, "tern-plugin/plugin.toml")
	if err != nil {
		return err
	}
	dir := ternPluginHome()
	have, _ := ternPluginVersion(os.DirFS(dir), "plugin.toml")
	linked := ternPluginLinkedTo() == dir
	if have == want && linked && !force {
		fmt.Printf("✅ %s %s is up to date\n", ternPluginID, want)
		return nil
	}

	// Rewrite the folder so files dropped from a newer version go away.
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	if err := extractTernPlugin(dir); err != nil {
		return err
	}
	if linked {
		err = ternPlugin("reload", ternPluginID)
	} else {
		// Replace an installed copy or a link elsewhere (a dev checkout).
		_ = ternPlugin("unlink", ternPluginID)
		_ = ternPlugin("remove", ternPluginID)
		err = ternPlugin("link", dir)
	}
	if err != nil {
		return err
	}
	switch {
	case have == "" || !linked:
		fmt.Printf("✅ installed %s %s\n", ternPluginID, want)
	case have == want:
		fmt.Printf("✅ reinstalled %s %s\n", ternPluginID, want)
	default:
		fmt.Printf("✅ updated %s %s → %s\n", ternPluginID, have, want)
	}
	if !ternPaneHookConfigured() {
		fmt.Println("⚠️  `cly claude-tern-pane` isn't a hook in ~/.claude/settings.json; panes get the icon but won't resume.")
		fmt.Println(`   Add it under hooks.SessionStart and hooks.SessionEnd: {"type": "command", "command": "cly claude-tern-pane"}`)
	}
	return nil
}

func runTernPluginUninstall(cmd *cobra.Command, args []string) error {
	if err := ternPlugin("unlink", ternPluginID); err != nil {
		if err2 := ternPlugin("remove", ternPluginID); err2 != nil {
			return err
		}
	}
	if err := os.RemoveAll(ternPluginHome()); err != nil {
		return err
	}
	fmt.Printf("✅ removed %s\n", ternPluginID)
	return nil
}

// ternPluginHome is the folder install extracts the plugin to and links
// into Tern; a Ternfile lists it as `link:<dir>?` so syncs keep it.
func ternPluginHome() string {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "cly", "tern-plugins", ternPluginID)
}

var ternVersionRe = regexp.MustCompile(`(?m)^version\s*=\s*"([^"]+)"`)

// ternPluginVersion reads `version` from a plugin.toml.
func ternPluginVersion(fsys fs.FS, path string) (string, error) {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return "", err
	}
	m := ternVersionRe.FindSubmatch(data)
	if m == nil {
		return "", fmt.Errorf("%s has no version", path)
	}
	return string(m[1]), nil
}

// ternPluginLinkedTo is the folder Tern links the plugin from, or "" when
// it isn't linked (absent or an installed copy).
func ternPluginLinkedTo() string {
	out, err := exec.Command("tern", "plugin", "dir").Output()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(strings.TrimSpace(string(out)), ternPluginID+".path"))
	if err != nil {
		return ""
	}
	return filepath.Clean(strings.TrimSpace(string(data)))
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
