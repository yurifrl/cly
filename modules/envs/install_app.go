package envs

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

const appName = "Envs"
const launchAgentLabel = "com.yurifrl.cly-envs"

func registerInstallApp(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "install-app",
		Short: "Install a macOS app and LaunchAgent to load envs at login",
		Long: `Interactively creates ~/Applications/Envs.app and optionally a LaunchAgent
that runs "cly envs --launchctl" at login. Prompts before each step and
before overwriting existing files.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return installApp()
		},
	}
	parent.AddCommand(cmd)
}

func confirm(prompt string) bool {
	fmt.Printf("%s [y/N] ", prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		return answer == "y" || answer == "yes"
	}
	return false
}

func installApp() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	clyPath, err := exec.LookPath("cly")
	if err != nil {
		return fmt.Errorf("cly not found in PATH: %w", err)
	}

	// 1. App
	appPath := filepath.Join(home, "Applications", appName+".app")
	createApp := true
	if _, err := os.Stat(appPath); err == nil {
		if !confirm(fmt.Sprintf("~/Applications/%s.app already exists. Overwrite?", appName)) {
			createApp = false
			fmt.Println("  Skipped app creation.")
		}
	} else {
		if !confirm(fmt.Sprintf("Create ~/Applications/%s.app?", appName)) {
			createApp = false
			fmt.Println("  Skipped app creation.")
		}
	}

	if createApp {
		if err := writeApp(home, clyPath); err != nil {
			return err
		}
		fmt.Printf("✓ Created ~/Applications/%s.app\n", appName)
	}

	// 2. LaunchAgent
	agentPath := filepath.Join(home, "Library", "LaunchAgents", launchAgentLabel+".plist")
	createAgent := true
	if _, err := os.Stat(agentPath); err == nil {
		if !confirm("LaunchAgent already exists. Overwrite?") {
			createAgent = false
			fmt.Println("  Skipped LaunchAgent.")
		}
	} else {
		if !confirm("Install LaunchAgent to run envs at login?") {
			createAgent = false
			fmt.Println("  Skipped LaunchAgent.")
		}
	}

	if createAgent {
		if err := writeAgent(home, clyPath, agentPath); err != nil {
			return err
		}
		fmt.Printf("✓ Created %s\n", agentPath)

		// 3. Load
		if confirm("Load the LaunchAgent now?") {
			_ = exec.Command("launchctl", "unload", agentPath).Run()
			if err := exec.Command("launchctl", "load", agentPath).Run(); err != nil {
				fmt.Printf("⚠ Could not load agent (try: launchctl load %s)\n", agentPath)
			} else {
				fmt.Println("✓ LaunchAgent loaded — will run at every login")
			}
		}
	}

	fmt.Println("\nDone!")
	return nil
}

func writeApp(home, clyPath string) error {
	appDir := filepath.Join(home, "Applications", appName+".app", "Contents", "MacOS")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return fmt.Errorf("create app dir: %w", err)
	}

	scriptPath := filepath.Join(appDir, appName)
	script := fmt.Sprintf(`#!/bin/bash
# Envs — load 1Password secrets into macOS GUI environment
export PATH="/opt/homebrew/bin:$PATH"
%s envs --launchctl
`, clyPath)
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		return fmt.Errorf("write app script: %w", err)
	}

	plistDir := filepath.Join(home, "Applications", appName+".app", "Contents")
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>Envs</string>
    <key>CFBundleIdentifier</key>
    <string>com.yurifrl.cly-envs</string>
    <key>CFBundleName</key>
    <string>Envs</string>
    <key>CFBundleVersion</key>
    <string>1.0</string>
    <key>LSUIElement</key>
    <true/>
</dict>
</plist>
`
	if err := os.WriteFile(filepath.Join(plistDir, "Info.plist"), []byte(plist), 0o644); err != nil {
		return fmt.Errorf("write Info.plist: %w", err)
	}
	return nil
}

func writeAgent(home, clyPath, agentPath string) error {
	agentDir := filepath.Dir(agentPath)
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		return fmt.Errorf("create LaunchAgents dir: %w", err)
	}

	tmpl := template.Must(template.New("plist").Parse(launchAgentTemplate))
	f, err := os.OpenFile(agentPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("create LaunchAgent: %w", err)
	}
	defer f.Close()

	data := struct{ ClyPath string }{ClyPath: clyPath}
	return tmpl.Execute(f, data)
}

const launchAgentTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.yurifrl.cly-envs</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.ClyPath}}</string>
        <string>envs</string>
        <string>--launchctl</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/cly-envs-agent.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/cly-envs-agent.log</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin</string>
    </dict>
</dict>
</plist>
`
