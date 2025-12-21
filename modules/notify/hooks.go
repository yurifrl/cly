package notify

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	pkgconfig "github.com/yurifrl/cly/pkg/config"
	"github.com/yurifrl/cly/pkg/style"
)

func createClaudeCmd() *cobra.Command {
	claudeCmd := &cobra.Command{
		Use:   "claude",
		Short: "Manage Claude Code hooks",
	}

	hooksCmd := &cobra.Command{
		Use:   "hooks",
		Short: "Manage hooks",
	}

	installCmd := createInstallCmd()
	removeCmd := createRemoveCmd()
	verifyCmd := createVerifyCmd()

	hooksCmd.AddCommand(installCmd, removeCmd, verifyCmd)
	claudeCmd.AddCommand(hooksCmd)
	return claudeCmd
}

func createInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install Claude Code hooks from config",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config
			cfg := pkgconfig.Get()
			if cfg == nil {
				return fmt.Errorf("failed to load config")
			}

			// Generate hooks from config
			hooks := getHooksFromConfig(cfg)
			if len(hooks) == 0 {
				fmt.Println(style.YellowStyle.Render("No hooks configured (set hook field in config)"))
				return nil
			}

			// Get settings path
			settingsPath, err := getSettingsPath()
			if err != nil {
				return err
			}

			// Read existing settings
			settings, err := readSettings(settingsPath)
			if err != nil {
				// Create new settings if file doesn't exist
				settings = make(map[string]interface{})
			}

			// Remove old cly hooks first
			removeNotifyHooks(settings)

			// Install fresh hooks from config
			installed := installHooks(settings, hooks)

			// Write settings
			if err := writeSettings(settingsPath, settings); err != nil {
				return err
			}

			// Success message
			hooksList := strings.Join(installed, ", ")
			fmt.Println(style.GreenStyle.Render(fmt.Sprintf("✅ Installed hooks: %s", hooksList)))
			return nil
		},
	}
}

func createRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove",
		Short: "Remove hooks containing 'cly' in command",
		RunE: func(cmd *cobra.Command, args []string) error {
			settingsPath, err := getSettingsPath()
			if err != nil {
				return err
			}

			settings, err := readSettings(settingsPath)
			if err != nil {
				fmt.Println(style.YellowStyle.Render("No hooks to remove"))
				return nil
			}

			// Remove ONLY cly hooks
			removed := removeNotifyHooks(settings)
			if len(removed) == 0 {
				fmt.Println(style.YellowStyle.Render("No cly hooks to remove"))
				return nil
			}

			// Write settings
			if err := writeSettings(settingsPath, settings); err != nil {
				return err
			}

			fmt.Println(style.GreenStyle.Render("✅ Removed cly hooks"))
			return nil
		},
	}
}

func createVerifyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "verify",
		Short: "Show installed Claude Code hooks",
		RunE: func(cmd *cobra.Command, args []string) error {
			settingsPath, err := getSettingsPath()
			if err != nil {
				return err
			}

			settings, err := readSettings(settingsPath)
			if err != nil {
				fmt.Println(style.YellowStyle.Render("No hooks installed"))
				return nil
			}

			// Get hooks section
			hooksRaw, ok := settings["hooks"]
			if !ok {
				fmt.Println(style.YellowStyle.Render("No hooks installed"))
				return nil
			}

			hooks, ok := hooksRaw.(map[string]interface{})
			if !ok || len(hooks) == 0 {
				fmt.Println(style.YellowStyle.Render("No hooks installed"))
				return nil
			}

			// Display hooks
			fmt.Println(style.BlueStyle.Render("Claude hooks:"))
			found := false
			for event, entriesRaw := range hooks {
				entries, ok := entriesRaw.([]interface{})
				if !ok {
					continue
				}

				for _, entryRaw := range entries {
					entry, ok := entryRaw.(map[string]interface{})
					if !ok {
						continue
					}

					hooksListRaw, ok := entry["hooks"]
					if !ok {
						continue
					}

					hooksList, ok := hooksListRaw.([]interface{})
					if !ok {
						continue
					}

					for _, hookRaw := range hooksList {
						hook, ok := hookRaw.(map[string]interface{})
						if !ok {
							continue
						}

						command, ok := hook["command"].(string)
						if !ok {
							continue
						}

						// Only show commands containing "cly"
						if strings.Contains(command, "cly") {
							found = true
							fmt.Printf("  %s: %s\n", event, command)
						}
					}
				}
			}

			if !found {
				fmt.Println(style.YellowStyle.Render("No cly hooks installed"))
			}

			return nil
		},
	}
}

// getHooksFromConfig generates hooks from notification hooks config
func getHooksFromConfig(cfg *pkgconfig.Config) map[string][]map[string]interface{} {
	eventGroups := make(map[string][]map[string]interface{})

	for hookName := range cfg.Notify.Hooks {
		// Hook names are already lowercase (notification, stop, posttooluse)
		entry := map[string]interface{}{
			"hooks": []map[string]string{
				{
					"type":    "command",
					"command": fmt.Sprintf("cly notify hook %s", hookName),
				},
			},
		}

		// posttooluse needs a matcher to catch all tools
		if hookName == "posttooluse" {
			entry["matcher"] = "*"
		}

		// Map to Claude hook names (notification -> Notification, stop -> Stop)
		claudeHookName := strings.ToUpper(string(hookName[0])) + hookName[1:]
		eventGroups[claudeHookName] = append(eventGroups[claudeHookName], entry)
	}

	return eventGroups
}

// getSettingsPath returns the path to Claude settings.json
func getSettingsPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(homeDir, ".claude", "settings.json"), nil
}

// readSettings reads and parses settings.json
func readSettings(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse settings.json: %w", err)
	}

	return settings, nil
}

// writeSettings writes settings to settings.json
func writeSettings(path string, settings map[string]interface{}) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal with indentation
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	// Write file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings: %w", err)
	}

	return nil
}

// installHooks installs new hooks into settings, returns list of installed hook names
func installHooks(settings map[string]interface{}, newHooks map[string][]map[string]interface{}) []string {
	installed := []string{}

	// Get or create hooks section
	var hooks map[string]interface{}
	if hooksRaw, ok := settings["hooks"]; ok {
		hooks, _ = hooksRaw.(map[string]interface{})
	}
	if hooks == nil {
		hooks = make(map[string]interface{})
	}

	// Install each event
	for event, entries := range newHooks {
		// Get existing entries for this event
		var existingEntries []interface{}
		if existingRaw, ok := hooks[event]; ok {
			existingEntries, _ = existingRaw.([]interface{})
		}

		// Add new entries
		for _, entry := range entries {
			existingEntries = append(existingEntries, entry)
			// Extract type name from command for display
			if hooksListRaw, ok := entry["hooks"]; ok {
				hooksList, _ := hooksListRaw.([]map[string]string)
				if len(hooksList) > 0 {
					command := hooksList[0]["command"]
					typeName := strings.TrimPrefix(command, "cly notify ")
					installed = append(installed, fmt.Sprintf("%s (%s)", typeName, event))
				}
			}
		}

		hooks[event] = existingEntries
	}

	settings["hooks"] = hooks
	return installed
}

// hookExists checks if a hook entry already exists
func hookExists(existingEntries []interface{}, newEntry map[string]interface{}) bool {
	newHooksRaw, ok := newEntry["hooks"]
	if !ok {
		return false
	}
	newHooks, ok := newHooksRaw.([]map[string]string)
	if !ok || len(newHooks) == 0 {
		return false
	}
	newCommand := newHooks[0]["command"]

	for _, existingRaw := range existingEntries {
		existing, ok := existingRaw.(map[string]interface{})
		if !ok {
			continue
		}

		existingHooksRaw, ok := existing["hooks"]
		if !ok {
			continue
		}

		existingHooksList, ok := existingHooksRaw.([]interface{})
		if !ok {
			continue
		}

		for _, hookRaw := range existingHooksList {
			hook, ok := hookRaw.(map[string]interface{})
			if !ok {
				continue
			}

			if command, ok := hook["command"].(string); ok && command == newCommand {
				return true
			}
		}
	}

	return false
}

// removeNotifyHooks removes ONLY cly hooks from settings
func removeNotifyHooks(settings map[string]interface{}) []string {
	removed := []string{}

	hooksRaw, ok := settings["hooks"]
	if !ok {
		return removed
	}

	hooks, ok := hooksRaw.(map[string]interface{})
	if !ok {
		return removed
	}

	// Process each event
	for event, entriesRaw := range hooks {
		entries, ok := entriesRaw.([]interface{})
		if !ok {
			continue
		}

		// Filter out ONLY hooks that contain "cly" in the command
		filtered := []interface{}{}
		for _, entryRaw := range entries {
			entry, ok := entryRaw.(map[string]interface{})
			if !ok {
				filtered = append(filtered, entryRaw)
				continue
			}

			hooksListRaw, ok := entry["hooks"]
			if !ok {
				filtered = append(filtered, entryRaw)
				continue
			}

			hooksList, ok := hooksListRaw.([]interface{})
			if !ok {
				filtered = append(filtered, entryRaw)
				continue
			}

			// Check if this hook contains "cly" in the command
			isClyHook := false
			for _, hookRaw := range hooksList {
				hook, ok := hookRaw.(map[string]interface{})
				if !ok {
					continue
				}

				command, ok := hook["command"].(string)
				if ok && strings.Contains(command, "cly") {
					isClyHook = true
					removed = append(removed, command)
					break
				}
			}

			// Keep if NOT a cly hook
			if !isClyHook {
				filtered = append(filtered, entryRaw)
			}
		}

		// Update or remove event
		if len(filtered) == 0 {
			delete(hooks, event)
		} else {
			hooks[event] = filtered
		}
	}

	// Remove hooks key if empty
	if len(hooks) == 0 {
		delete(settings, "hooks")
	}

	return removed
}
