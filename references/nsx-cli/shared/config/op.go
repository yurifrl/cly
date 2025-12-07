package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/NSXBet/nsx-cli/shared/interact"
)

const (
	nsxAccountUUID   = "nsx-team.1password.com"
	opCommandDefault = "op"
	opCommandWindows = "op.exe"
)

func DownloadConfig(registry, destination string) error {
	if registry == "" {
		return fmt.Errorf("registry flag is required")
	}

	opCommand, err := opCommand()
	if err != nil {
		return fmt.Errorf("failed to get op command: %w", err)
	}

	opCmd := exec.Command(opCommand, "read", registry, "--account", nsxAccountUUID)
	var stdout, stderr bytes.Buffer
	opCmd.Stdout = &stdout
	opCmd.Stderr = &stderr

	if err := opCmd.Run(); err != nil {
		return fmt.Errorf("failed to execute op command: %w\n%s", err, stderr.String())
	}

	configDir := BaseFolder()
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Encrypt the config data before saving
	encryptedData, err := Encrypt(stdout.Bytes())
	if err != nil {
		return fmt.Errorf("failed to encrypt config data: %w", err)
	}

	configPath := filepath.Join(configDir, destination)
	if err := os.WriteFile(configPath, encryptedData, 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	interact.Success("Config downloaded and encrypted successfully to: %s", configPath)
	return nil
}

func GetItem(itemID string) (map[string]string, error) {
	opCommand, err := opCommand()
	if err != nil {
		return nil, fmt.Errorf("failed to get op command: %w", err)
	}

	opCmd := exec.Command(opCommand, "item", "get", itemID, "--account", nsxAccountUUID, "--format", "json", "--reveal")
	var stdout, stderr bytes.Buffer
	opCmd.Stdout = &stdout
	opCmd.Stderr = &stderr

	if err := opCmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to execute op command: %w\n%s", err, stderr.String())
	}

	var item map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal item: %w", err)
	}

	fieldsRaw, ok := item["fields"]
	if !ok {
		return nil, fmt.Errorf("fields is not a map")
	}

	fields, ok := fieldsRaw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("fields is not a map")
	}

	itemMap := make(map[string]string)
	for k, v := range fields {
		itemMap[k] = fmt.Sprintf("%v", v)
	}

	return itemMap, nil
}

func opCommand() (string, error) {
	opCommand := opCommandDefault
	if _, err := exec.LookPath("op"); err != nil {
		if _, err := exec.LookPath("op.exe"); err != nil {
			return "", fmt.Errorf("neither 'op' nor 'op.exe' found in PATH: %w", err)
		}
		opCommand = opCommandWindows
	}
	return opCommand, nil
}
