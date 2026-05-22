package notify

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	pkgconfig "github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/envs"
	"github.com/yurifrl/cly/pkg/notify"
	"github.com/yurifrl/cly/pkg/style"
)

func Register(parent *cobra.Command) {
	notifyCmd := &cobra.Command{
		Use:   "notify",
		Short: "Claude Code notification system",
		Long:  "Send and manage notifications for Claude Code hook events",
	}

	// Dynamic hook command
	hookCmd := createHookCmd()

	// Utility commands
	soundCmd := createSoundCmd()
	configCmd := createConfigCmd()
	debugCmd := createDebugCmd()
	claudeCmd := createClaudeCmd()

	notifyCmd.AddCommand(hookCmd, soundCmd, configCmd, debugCmd, claudeCmd)
	parent.AddCommand(notifyCmd)
}

func createHookCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "hook <hookname>",
		Short: "Send notification for a hook",
		Long:  "Send notification for any hook defined in config (e.g., stop, notification, mycustomhook)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			hookName := args[0]

			// CLAUDE_VERBOSE=0 (or any falsy value) silences notifications.
			// Empty/Error fall through to verbose-by-default behavior.
			if v, _ := envs.ClaudeVerbose().Unwrap(); envs.ClaudeVerbose().IsOk() && !v {
				return nil
			}

			// Load config
			cfg := pkgconfig.Get()
			if cfg == nil {
				return fmt.Errorf("failed to load config")
			}

			notifyConfig := cfg.GetNotify()

			// Check if notifications are enabled
			if !notifyConfig.Enabled {
				return nil
			}

			// Get hook config
			hookConfig, ok := notifyConfig.Hooks[hookName]
			if !ok {
				return fmt.Errorf("hook '%s' not configured", hookName)
			}

			// Check if hook is enabled
			if !hookConfig.Enabled {
				return nil
			}

			// Generate group name: cly-claude-hooks-{lowercase}
			group := fmt.Sprintf("cly-claude-hooks-%s", strings.ToLower(hookName))

			// Create notification with config values (already env-expanded)
			n := notify.Notification{
				Title:   hookConfig.Title,
				Message: hookConfig.Message,
				Sound:   hookConfig.Sound,
				Group:   group,
			}

			// Get icon path (use embedded if not configured)
			iconPath := notifyConfig.Icon
			if iconPath == "" {
				iconPath, _ = notify.GetIconPath()
			}

			// Send to all enabled notifiers
			notifier := notify.New(
				hookName,
				notifyConfig.UseZellijStatus,
				notifyConfig.UseZellijNotify,
				notifyConfig.UseZellijAttention,
				iconPath,
			)
			return notifier.Send(context.Background(), n)
		},
	}
}

func createSoundCmd() *cobra.Command {
	soundCmd := &cobra.Command{
		Use:   "sound [on|off|status]",
		Short: "Toggle sound on/off/status",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			soundFile := getSoundFilePath()
			cfg := pkgconfig.Get()
			notifyConfig := cfg.GetNotify()

			action := "status"
			if len(args) > 0 {
				action = args[0]
			}

			switch action {
			case "on", "enable":
				if err := setSoundEnabled(soundFile, true); err != nil {
					return err
				}
				fmt.Println(style.GreenStyle.Render("✅ Sound notifications enabled"))

			case "off", "disable":
				if err := setSoundEnabled(soundFile, false); err != nil {
					return err
				}
				fmt.Println(style.YellowStyle.Render("🔇 Sound notifications disabled"))

			case "status", "":
				enabled := isSoundEnabled(soundFile, notifyConfig.Sound)
				if enabled {
					fmt.Println(style.BlueStyle.Render("🔔 Sound is currently ON"))
				} else {
					fmt.Println(style.BlueStyle.Render("🔇 Sound is currently OFF"))
				}

			default:
				return fmt.Errorf("invalid argument: %s. Use: on, off, or status", args[0])
			}

			return nil
		},
	}

	return soundCmd
}

func createConfigCmd() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage notify configuration",
	}

	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Display notify settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := pkgconfig.Get()
			notifyConfig := cfg.GetNotify()
			soundFile := getSoundFilePath()
			soundEnabled := isSoundEnabled(soundFile, notifyConfig.Sound)

			fmt.Println(style.BlueStyle.Render("Notify Configuration:"))
			fmt.Printf("  Enabled: %v\n", notifyConfig.Enabled)
			fmt.Printf("  Sound: %v\n", soundEnabled)
			fmt.Printf("  Use Zellij Status: %v\n", notifyConfig.UseZellijStatus)
			fmt.Printf("  Use Zellij Notify: %v\n", notifyConfig.UseZellijNotify)
			fmt.Printf("  Use Zellij Attention: %v\n", notifyConfig.UseZellijAttention)
			fmt.Printf("  Icon: %s\n", notifyConfig.Icon)
			fmt.Println()
			fmt.Println(style.BlueStyle.Render("Hooks:"))
			for name, hookConfig := range notifyConfig.Hooks {
				fmt.Printf("  %s:\n", name)
				fmt.Printf("    Enabled: %v\n", hookConfig.Enabled)
				fmt.Printf("    Title: %s\n", hookConfig.Title)
				fmt.Printf("    Message: %s\n", hookConfig.Message)
			}

			return nil
		},
	}

	setCmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Update notify setting",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := "notify." + args[0]
			value := args[1]

			// Convert boolean strings
			if value == "true" || value == "false" {
				boolVal := value == "true"
				if err := pkgconfig.Set(key, boolVal); err != nil {
					return err
				}
			} else {
				if err := pkgconfig.Set(key, value); err != nil {
					return err
				}
			}

			fmt.Println(style.GreenStyle.Render(fmt.Sprintf("✓ Set %s = %s", key, value)))
			return nil
		},
	}

	resetCmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset to defaults",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Remove sound file
			soundFile := getSoundFilePath()
			os.Remove(soundFile)

			fmt.Println(style.GreenStyle.Render("✓ Notify configuration reset to defaults"))
			fmt.Println(style.YellowStyle.Render("  Note: config.yaml values remain unchanged"))
			return nil
		},
	}

	configCmd.AddCommand(showCmd, setCmd, resetCmd)
	return configCmd
}
