package notify

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	pkgconfig "github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/notify"
	"github.com/yurifrl/cly/pkg/style"
)

func createDebugCmd() *cobra.Command {
	var respectConfig bool
	var testBeeep bool
	var testTerminal bool
	var testZellijStatus bool
	var testZellijNotify bool

	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Debug notification system (test all notifiers)",
		Long:  "Send test notifications through all notifiers and show detailed debug output",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := pkgconfig.Get()

			fmt.Println(style.BlueStyle.Render("=== CLY Notify Debug ==="))
			fmt.Println()

			// Environment checks
			fmt.Println(style.YellowStyle.Render("Environment:"))
			fmt.Printf("  CLAUDE_VERBOSE: %s\n", getEnvOrDefault("CLAUDE_VERBOSE", "1"))
			fmt.Printf("  ZELLIJ: %s\n", getEnvOrDefault("ZELLIJ", "(not set)"))
			fmt.Printf("  ZELLIJ_SESSION_NAME: %s\n", getEnvOrDefault("ZELLIJ_SESSION_NAME", "(not set)"))
			fmt.Printf("  CLAUDE_SESSION_NAME: %s\n", getEnvOrDefault("CLAUDE_SESSION_NAME", "(not set)"))
			fmt.Printf("  SOUND: %s\n", getEnvOrDefault("SOUND", "(not set)"))
			fmt.Println()

			// Configuration
			fmt.Println(style.YellowStyle.Render("Configuration:"))
			soundFile := getSoundFilePath()
			soundEnabled := isSoundEnabled(soundFile, cfg.Notify.Sound)
			fmt.Printf("  Sound Enabled: %v\n", soundEnabled)
			fmt.Printf("  Sound File: %s (exists: %v)\n", soundFile, fileExists(soundFile))
			fmt.Printf("  Use Beeep: %v\n", cfg.Notify.UseBeeep)
			fmt.Printf("  Use Terminal-Notifier: %v\n", cfg.Notify.UseTerminalNotifier)
			fmt.Printf("  Use Zellij Status: %v\n", cfg.Notify.UseZellijStatus)
			fmt.Printf("  Use Zellij Notify: %v\n", cfg.Notify.UseZellijNotify)

			// Icon path
			iconPath := cfg.Notify.Icon
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

			// Terminal-Notifier
			terminalNotifier := &notify.TerminalNotifier{}
			terminalAvail := terminalNotifier.Available()
			fmt.Printf("  Terminal-Notifier: %s", availStatus(terminalAvail))
			if !terminalAvail {
				fmt.Print(" (install: brew install terminal-notifier)")
			}
			fmt.Println()

			// Zellij
			zellijNotifier := notify.NewZellijNotifier("debug", true, true)
			zellijAvail := zellijNotifier.Available()
			fmt.Printf("  Zellij: %s", availStatus(zellijAvail))
			if !zellijAvail {
				fmt.Print(" (not in Zellij session)")
			}
			fmt.Println()
			fmt.Println()

			// Determine which notifiers to use
			var enabledNotifiers []string
			var notifiers []notify.Notifier

			// If individual test flags are set, use those
			if testBeeep || testTerminal || testZellijStatus || testZellijNotify {
				fmt.Println(style.YellowStyle.Render("Testing selected notifiers:"))
				if testBeeep && beeepAvail {
					enabledNotifiers = append(enabledNotifiers, "Beeep")
					notifiers = append(notifiers, beeepNotifier)
				}
				if testTerminal && terminalAvail {
					enabledNotifiers = append(enabledNotifiers, "Terminal-Notifier")
					notifiers = append(notifiers, terminalNotifier)
				}
				if (testZellijStatus || testZellijNotify) && zellijAvail {
					enabledNotifiers = append(enabledNotifiers, "Zellij")
					notifiers = append(notifiers, notify.NewZellijNotifier("debug", testZellijStatus, testZellijNotify))
				}
			} else if respectConfig {
				fmt.Println(style.YellowStyle.Render("Testing with config settings:"))
				if cfg.Notify.UseBeeep && beeepAvail {
					enabledNotifiers = append(enabledNotifiers, "Beeep")
					notifiers = append(notifiers, beeepNotifier)
				}
				if cfg.Notify.UseTerminalNotifier && terminalAvail {
					enabledNotifiers = append(enabledNotifiers, "Terminal-Notifier")
					notifiers = append(notifiers, terminalNotifier)
				}
				if (cfg.Notify.UseZellijStatus || cfg.Notify.UseZellijNotify) && zellijAvail {
					enabledNotifiers = append(enabledNotifiers, "Zellij")
					notifiers = append(notifiers, notify.NewZellijNotifier("debug", cfg.Notify.UseZellijStatus, cfg.Notify.UseZellijNotify))
				}
			} else {
				fmt.Println(style.YellowStyle.Render("Testing ALL available notifiers:"))
				if beeepAvail {
					enabledNotifiers = append(enabledNotifiers, "Beeep")
					notifiers = append(notifiers, beeepNotifier)
				}
				if terminalAvail {
					enabledNotifiers = append(enabledNotifiers, "Terminal-Notifier")
					notifiers = append(notifiers, terminalNotifier)
				}
				if zellijAvail {
					enabledNotifiers = append(enabledNotifiers, "Zellij")
					notifiers = append(notifiers, zellijNotifier)
				}
			}

			if len(enabledNotifiers) == 0 {
				fmt.Println(style.RedStyle.Render("  ✗ No notifiers available or enabled"))
				return fmt.Errorf("no notifiers available")
			}

			for _, name := range enabledNotifiers {
				fmt.Printf("  ✓ %s\n", name)
			}
			fmt.Println()

			// Context string
			contextStr := buildContextString()
			if contextStr != "" {
				fmt.Println(style.YellowStyle.Render("Context String:"))
				fmt.Printf("  %s\n", contextStr)
				fmt.Println()
			}

			// Send test notification
			fmt.Println(style.YellowStyle.Render("Sending Test Notification:"))

			message := "Test"
			if contextStr != "" {
				message = message + " " + contextStr
			}

			testNotification := notify.Notification{
				Title:    "CLY Notify Debug",
				Subtitle: "",
				Message:  message,
				Sound:    "Ping",
				Group:    "cly-debug",
			}


			ctx := context.Background()
			for i, notifier := range notifiers {
				notifierName := enabledNotifiers[i]
				fmt.Println(style.YellowStyle.Render(fmt.Sprintf("Testing %s:", notifierName)))

				// Put notifier name in subtitle so it's always visible
				testN := testNotification
				testN.Subtitle = notifierName

				// Show what will actually be sent
				switch notifierName {
				case "Beeep":
					fmt.Printf("  Title: %s\n", testN.Title+" - "+testN.Subtitle)
					fmt.Printf("  Message: [beeep] %s\n", testN.Message)

				case "Terminal-Notifier":
					fmt.Printf("  Title: %s\n", testN.Title)
					fmt.Printf("  Subtitle: %s\n", testN.Subtitle)
					fmt.Printf("  Message: [terminal-notifier] %s\n", testN.Message)

				case "Zellij":
					paneID := os.Getenv("ZELLIJ_PANE_ID")
					sessionName := os.Getenv("ZELLIJ_SESSION_NAME")
					fmt.Printf("  Status Bar: zellij pipe \"zjstatus::notify::%s\"\n", testN.Title)
					cmd := fmt.Sprintf("  Tab: zellij pipe -n notify")
					if paneID != "" {
						cmd += fmt.Sprintf(" -a pane_id=%s", paneID)
					}
					if sessionName != "" {
						cmd += fmt.Sprintf(" -a session_name=%s", sessionName)
					}
					cmd += " debug"
					fmt.Println(cmd)
				}

				fmt.Printf("  Sending... ")
				if err := notifier.Send(ctx, testN); err != nil {
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
	cmd.Flags().BoolVar(&testBeeep, "beeep", false, "Test only Beeep notifier")
	cmd.Flags().BoolVar(&testTerminal, "terminal", false, "Test only Terminal-Notifier")
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

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
