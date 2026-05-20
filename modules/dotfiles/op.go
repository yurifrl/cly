package dotfiles

import (
	"github.com/yurifrl/cly/pkg/mut"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yurifrl/cly/pkg/style"
)

var opInjectRun = func(account, source, destination string) error {
	args := []string{"inject", "-f", "-i", source, "-o", destination}
	if account != "" {
		args = []string{"inject", "-f", "--account", account, "-i", source, "-o", destination}
	}
	return mut.Exec("op", args...)
}

var opReadRun = func(account, reference, destination string) error {
	args := []string{"read", "-f", "--out-file", destination}
	if account != "" {
		args = []string{"read", "-f", "--account", account, "--out-file", destination}
	}
	args = append(args, reference)
	return mut.Exec("op", args...)
}

// RemoveOpMapping deletes the generated destination file for an op mapping.
// Returns true if the file was removed, false otherwise.
func RemoveOpMapping(m OpMapping) bool {
	if err := mut.Remove(m.Destination); err != nil {
		return false
	}
	return true
}

func ApplyOpMappings(cfg *Config, failFast bool) error {
	if len(cfg.OpMappings) == 0 {
		return nil
	}

	defaultAccount := dotfilesModuleString("", "op", "account")

	for _, m := range cfg.OpMappings {
		account := m.Account
		if account == "" {
			account = defaultAccount
		}

		if err := mut.MkdirAll(filepath.Dir(m.Destination), 0755); err != nil {
			return fmt.Errorf("create parent dir for %s: %w", m.Destination, err)
		}

		if m.IsReference {
			fmt.Printf("%s %s -> %s\n",
				style.BlueStyle.Render("🔑 op read:"),
				m.Source, shortenPath(m.Destination))

			if err := opReadRun(account, m.Source, m.Destination); err != nil {
				fmt.Printf("  %s %s\n", style.RedStyle.Render("❌"), err)
				if failFast {
					return fmt.Errorf("op read %s: %w", m.Source, err)
				}
				continue
			}
			continue
		}

		if _, err := os.Stat(m.Source); os.IsNotExist(err) {
			fmt.Printf("  %s Source '%s' does not exist, skipping\n",
				style.YellowStyle.Render("⚠️  Warning:"), shortenPath(m.Source))
			continue
		}

		fmt.Printf("%s %s -> %s\n",
			style.BlueStyle.Render("🔑 op inject:"),
			shortenPath(m.Source), shortenPath(m.Destination))

		if err := opInjectRun(account, m.Source, m.Destination); err != nil {
			fmt.Printf("  %s %s\n", style.RedStyle.Render("❌"), err)
			if failFast {
				return fmt.Errorf("op inject %s: %w", m.Source, err)
			}
			continue
		}
	}

	return nil
}
