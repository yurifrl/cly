package envs

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"strings"
)

// keychainService intentionally keeps the "nsx-envs" service name so caches
// written by the original nsx implementation remain readable.
const keychainService = "nsx-envs"

// keychainStoreAll stores all fields as a single Keychain entry.
// Format: tab-separated label/section/value lines, base64-encoded for a
// reliable round-trip through the security CLI.
func keychainStoreAll(fields []Field) error {
	var lines []string
	for _, f := range fields {
		lines = append(lines, f.Label+"\t"+f.Section+"\t"+f.Value)
	}
	raw := strings.Join(lines, "\n")
	encoded := base64Encode(raw)
	return keychainSet("__blob__", encoded)
}

// keychainReadAll reads all fields from a single Keychain entry.
func keychainReadAll() ([]Field, error) {
	encoded, err := keychainGet("__blob__")
	if err != nil {
		return nil, fmt.Errorf("no envs in Keychain")
	}
	if encoded == "" {
		return nil, fmt.Errorf("empty blob")
	}

	raw, err := base64Decode(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode blob: %w", err)
	}

	lines := strings.Split(raw, "\n")
	fields := make([]Field, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			continue
		}
		fields = append(fields, Field{Label: parts[0], Section: parts[1], Value: parts[2]})
	}
	return fields, nil
}

// --- macOS Keychain helpers via `security` CLI ---

func keychainSet(key, value string) error {
	_ = keychainDelete(key)
	cmd := exec.Command(
		"security", "add-generic-password",
		"-a", "nsx-envs",
		"-s", keychainService+"/"+key,
		"-w", value,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, output)
	}
	return nil
}

func keychainGet(key string) (string, error) {
	cmd := exec.Command(
		"security", "find-generic-password",
		"-a", "nsx-envs",
		"-s", keychainService+"/"+key,
		"-w",
	)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(string(output), "\n"), nil
}

func keychainDelete(key string) error {
	cmd := exec.Command(
		"security", "delete-generic-password",
		"-a", "nsx-envs",
		"-s", keychainService+"/"+key,
	)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

func base64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func base64Decode(s string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
