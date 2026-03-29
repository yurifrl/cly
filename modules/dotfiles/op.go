package dotfiles

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/yurifrl/cly/pkg/style"
)

var opInjectRun = func(account, source, destination string) error {
	args := []string{"inject", "-f", "-i", source, "-o", destination}
	if account != "" {
		args = append([]string{"inject", "-f", "--account", account, "-i", source, "-o", destination})
	}
	cmd := exec.Command("op", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RemoveOpMapping deletes the generated destination file for an op mapping.
// Returns true if the file was removed, false otherwise.
func RemoveOpMapping(m OpMapping) bool {
	if err := os.Remove(m.Destination); err != nil {
		return false
	}
	return true
}

func ApplyOpMappings(cfg *Config) error {
	if len(cfg.OpMappings) == 0 {
		return nil
	}

	defaultAccount := dotfilesModuleString("", "op", "account")

	for _, m := range cfg.OpMappings {
		account := m.Account
		if account == "" {
			account = defaultAccount
		}

		if _, err := os.Stat(m.Source); os.IsNotExist(err) {
			fmt.Printf("  %s Source '%s' does not exist, skipping\n",
				style.YellowStyle.Render("⚠️  Warning:"), shortenPath(m.Source))
			continue
		}

		if err := os.MkdirAll(filepath.Dir(m.Destination), 0755); err != nil {
			return fmt.Errorf("create parent dir for %s: %w", m.Destination, err)
		}

		fmt.Printf("%s %s -> %s\n",
			style.BlueStyle.Render("🔑 op inject:"),
			shortenPath(m.Source), shortenPath(m.Destination))

		if err := opInjectRun(account, m.Source, m.Destination); err != nil {
			fmt.Printf("  %s %s\n", style.RedStyle.Render("❌"), err)
			continue
		}
	}

	return nil
}
