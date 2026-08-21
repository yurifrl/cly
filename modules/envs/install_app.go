package envs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/spf13/cobra"
)

const appName = "ClyEnvs"
const launchAgentLabel = "com.yurifrl.cly-envs"

func registerInstallApp(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "install-app",
		Short: "Install a macOS app and LaunchAgent to load envs at login",
		Long: `Creates ~/Applications/ClyEnvs.app (clickable) and a LaunchAgent
that runs "cly envs --plain --launchctl" at login. Environment variables
are injected via launchctl setenv so all GUI apps inherit them.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return installApp()
		},
	}
	parent.AddCommand(cmd)
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

	// 1. Create ~/Applications/ClyEnvs.app
	appDir := filepath.Join(home, "Applications", appName+".app", "Contents", "MacOS")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return fmt.Errorf("create app dir: %w", err)
	}

	scriptPath := filepath.Join(appDir, appName)
	script := fmt.Sprintf(`#!/bin/bash
# ClyEnvs — load 1Password secrets into macOS GUI environment
export PATH="/opt/homebrew/bin:$PATH"
%s envs --plain --launchctl
`, clyPath)
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		return fmt.Errorf("write app script: %w", err)
	}

	// Write Info.plist
	plistDir := filepath.Join(home, "Applications", appName+".app", "Contents")
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>ClyEnvs</string>
    <key>CFBundleIdentifier</key>
    <string>com.yurifrl.cly-envs</string>
    <key>CFBundleName</key>
    <string>ClyEnvs</string>
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

	fmt.Printf("✓ Created ~/Applications/%s.app\n", appName)

	// 2. Create LaunchAgent
	agentDir := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		return fmt.Errorf("create LaunchAgents dir: %w", err)
	}

	agentPath := filepath.Join(agentDir, launchAgentLabel+".plist")
	tmpl := template.Must(template.New("plist").Parse(launchAgentTemplate))
	f, err := os.OpenFile(agentPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("create LaunchAgent: %w", err)
	}
	defer f.Close()

	data := struct{ ClyPath string }{ClyPath: clyPath}
	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("write LaunchAgent: %w", err)
	}

	fmt.Printf("✓ Created %s\n", agentPath)

	// 3. Load the agent
	// Unload first (ignore error if not loaded)
	_ = exec.Command("launchctl", "unload", agentPath).Run()
	if err := exec.Command("launchctl", "load", agentPath).Run(); err != nil {
		fmt.Printf("⚠ Could not load agent (try: launchctl load %s)\n", agentPath)
	} else {
		fmt.Println("✓ LaunchAgent loaded — will run at every login")
	}

	fmt.Println("\nDone! Your env vars will be available to all GUI apps after login.")
	fmt.Printf("Click ~/Applications/%s.app anytime to refresh manually.\n", appName)
	return nil
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
        <string>--plain</string>
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
