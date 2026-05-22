package notify

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	pkgconfig "github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/envs"
	"github.com/yurifrl/cly/pkg/notify"
	"github.com/yurifrl/cly/pkg/result"
	"github.com/yurifrl/cly/pkg/style"
)

func createDebugCmd() *cobra.Command {
	var respectConfig bool
	var testZellijStatus bool
	var testZellijNotify bool

	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Debug notification system (test all notifiers)",
		Long:  "Send test notifications through all notifiers and show detailed debug output",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := pkgconfig.Get()
			notifyConfig := cfg.GetNotify()

			fmt.Println(style.BlueStyle.Render("=== CLY Notify Debug ==="))
			fmt.Println()

			// Environment checks
			fmt.Println(style.YellowStyle.Render("Environment:"))
			fmt.Printf("  CLAUDE_VERBOSE: %s\n", displayBool(envs.ClaudeVerbose(), "1"))
			fmt.Printf("  ZELLIJ: %s\n", display(envs.Zellij()))
			fmt.Printf("  ZELLIJ_SESSION_NAME: %s\n", display(envs.ZellijSession()))
			fmt.Printf("  Session name: %s\n", display(envs.SessionName()))
			fmt.Printf("  SOUND: %s\n", display(envs.Sound()))
			fmt.Println()

			// Configuration
			fmt.Println(style.YellowStyle.Render("Configuration:"))
			soundFile := getSoundFilePath()
			soundEnabled := isSoundEnabled(soundFile, notifyConfig.Sound)
			fmt.Printf("  Enabled: %v\n", notifyConfig.Enabled)
			fmt.Printf("  Sound Enabled: %v\n", soundEnabled)
			fmt.Printf("  Sound File: %s (exists: %v)\n", soundFile, fileExists(soundFile))
			fmt.Printf("  Use Zellij Status: %v\n", notifyConfig.UseZellijStatus)
			fmt.Printf("  Use Zellij Notify: %v\n", notifyConfig.UseZellijNotify)
			fmt.Printf("  Use Zellij Attention: %v\n", notifyConfig.UseZellijAttention)

			// Icon path
			iconPath := notifyConfig.Icon
			if iconPath == "" {
				iconPath, _ = notify.GetIconPath()
				fmt.Printf("  Icon: %s (embedded, extracted)\n", iconPath)
			} else {
				fmt.Printf("  Icon: %s (custom)\n", iconPath)
			}
			fmt.Printf("  Icon Exists: %v\n", fileExists(iconPath))
			fmt.Println()

			// Check notifier availability
			fmt.Println(style.YellowStyle.Render("Notifier Availability:"))

			// Beeep
			beeepNotifier := &notify.BeeepNotifier{Icon: iconPath}
			beeepAvail := beeepNotifier.Available()
			fmt.Printf("  Beeep: %s\n", availStatus(beeepAvail))

			// Zellij
			zellijNotifier := notify.NewZellijNotifier("debug", true, true, true)
			zellijAvail := zellijNotifier.Available()
			fmt.Printf("  Zellij: %s", availStatus(zellijAvail))
			if !zellijAvail {
				fmt.Print(" (not in Zellij session)")
			}
			fmt.Println()
			fmt.Println()

			// Send test notification
			fmt.Println(style.YellowStyle.Render("Sending Test Notification:"))

			testNotification := notify.Notification{
				Title:   "CLY Notify Debug",
				Message: "Test notification with env vars",
				Sound:   "Ping",
				Group:   "cly-debug",
			}

			ctx := context.Background()

			// Test Beeep
			fmt.Println(style.YellowStyle.Render("Testing Beeep:"))
			fmt.Printf("  Title: %s\n", testNotification.Title)
			fmt.Printf("  Message: %s\n", testNotification.Message)
			fmt.Printf("  Sending... ")
			if err := beeepNotifier.Send(ctx, testNotification); err != nil {
				fmt.Println(style.RedStyle.Render("✗ Failed: " + err.Error()))
			} else {
				fmt.Println(style.GreenStyle.Render("✓ Sent"))
			}
			fmt.Println()

			// Test Zellij
			if zellijAvail {
				fmt.Println(style.YellowStyle.Render("Testing Zellij:"))
				paneID := envs.ZellijPane().Or("")
				fmt.Printf("  Status Bar: zellij pipe \"zjstatus::notify::%s\"\n", testNotification.Title)
				fmt.Printf("  Attention: zellij pipe --name \"zellij-attention::waiting::%s\"\n", paneID)
				fmt.Printf("  Sending... ")
				if err := zellijNotifier.Send(ctx, testNotification); err != nil {
					fmt.Println(style.RedStyle.Render("✗ Failed: " + err.Error()))
				} else {
					fmt.Println(style.GreenStyle.Render("✓ Sent"))
				}
				fmt.Println()
			}

			fmt.Println()
			fmt.Println(style.GreenStyle.Render("=== Debug Complete ==="))
			fmt.Println()
			fmt.Println("Tip: Use --config flag to test only configured notifiers")
			fmt.Println("     Example: cly notify debug --config")

			return nil
		},
	}

	cmd.Flags().BoolVar(&respectConfig, "config", false, "Only test notifiers enabled in config")
	cmd.Flags().BoolVar(&testZellijStatus, "zellij-status", false, "Test only Zellij status bar")
	cmd.Flags().BoolVar(&testZellijNotify, "zellij-notify", false, "Test only Zellij notify plugin")
	return cmd
}

func availStatus(available bool) string {
	if available {
		return style.GreenStyle.Render("✓ Available")
	}
	return style.RedStyle.Render("✗ Unavailable")
}

// display formats a Result[string] for the debug environment table.
// Empty and Error fall through to a human-readable marker so the user
// can tell them apart at a glance.
func display(r result.Result[string]) string {
	if err := r.Error(); err != nil {
		return fmt.Sprintf("(invalid: %v)", err)
	}
	if r.Empty() {
		return "(not set)"
	}
	v, _ := r.Unwrap()
	return v
}

// displayBool formats a Result[bool] for debug output. When the
// underlying var is unset, def is shown verbatim (preserves legacy
// behavior of CLAUDE_VERBOSE defaulting to "1").
func displayBool(r result.Result[bool], def string) string {
	if err := r.Error(); err != nil {
		return fmt.Sprintf("(invalid: %v)", err)
	}
	if r.Empty() {
		return def
	}
	v, _ := r.Unwrap()
	return fmt.Sprintf("%t", v)
}

// rawCLAUDE_SESSION_NAME removed: pkg/envs.SessionName() is the
// canonical accessor and already covers the legacy alias.

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
