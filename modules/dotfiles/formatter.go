package dotfiles

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/yurifrl/cly/pkg/mut"
)

type opFormatter func(destination string) error

var opFormatters = map[string]opFormatter{
	"format-ssh": formatSSHKey,
}

var sshKeygenPublicRun = func(privateKey string) ([]byte, error) {
	return exec.Command("ssh-keygen", "-y", "-f", privateKey).Output()
}

func isOpFormatter(name string) bool {
	_, ok := opFormatters[name]
	return ok
}

func applyOpFormatters(m OpMapping) error {
	for _, name := range m.Formatters {
		if err := opFormatters[name](m.Destination); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}

func formatSSHKey(destination string) error {
	if err := mut.Chmod(destination, 0600); err != nil {
		return fmt.Errorf("restrict private key permissions: %w", err)
	}
	if mut.DryRun() {
		return nil
	}

	privateKey, err := os.ReadFile(destination)
	if err != nil {
		return fmt.Errorf("read private key: %w", err)
	}
	if !strings.Contains(string(privateKey), "PRIVATE KEY") {
		return fmt.Errorf("destination is not an SSH private key")
	}

	publicKey, err := sshKeygenPublicRun(destination)
	if err != nil {
		return fmt.Errorf("derive public key: %w", err)
	}
	publicDestination := destination + ".pub"
	temporaryPublicDestination := filepath.Join(filepath.Dir(publicDestination), "."+filepath.Base(publicDestination)+".tmp")
	if err := os.WriteFile(temporaryPublicDestination, publicKey, 0644); err != nil {
		return fmt.Errorf("write public key: %w", err)
	}
	if err := mut.Rename(temporaryPublicDestination, publicDestination); err != nil {
		return fmt.Errorf("install public key: %w", err)
	}
	return mut.Chmod(publicDestination, 0644)
}
