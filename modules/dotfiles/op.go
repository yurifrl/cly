package dotfiles

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yurifrl/cly/pkg/mut"
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
			fmt.Printf("%s %s -> %s%s\n", style.BlueStyle.Render("🔑 op read:"), m.Source, shortenPath(m.Destination), gateSuffix(m.Gate))
			err := writeOpMapping(m, func(destination string) error {
				return opReadRun(account, m.Source, destination)
			})
			if err := handleOpMappingError("op read", m, err, failFast); err != nil {
				return err
			}
			continue
		}

		if _, err := os.Stat(m.Source); os.IsNotExist(err) {
			fmt.Printf("  %s Source '%s' does not exist, skipping\n", style.YellowStyle.Render("⚠️  Warning:"), shortenPath(m.Source))
			continue
		}
		fmt.Printf("%s %s -> %s%s\n", style.BlueStyle.Render("🔑 op inject:"), shortenPath(m.Source), shortenPath(m.Destination), gateSuffix(m.Gate))
		err := writeOpMapping(m, func(destination string) error {
			return opInjectRun(account, m.Source, destination)
		})
		if err := handleOpMappingError("op inject", m, err, failFast); err != nil {
			return err
		}
	}
	return nil
}

func handleOpMappingError(operation string, m OpMapping, err error, failFast bool) error {
	if err == nil {
		return nil
	}
	fmt.Printf("  %s %s\n", style.RedStyle.Render("❌"), err)
	if failFast {
		return fmt.Errorf("%s %s: %w", operation, m.Source, err)
	}
	return nil
}

func writeOpMapping(m OpMapping, write func(destination string) error) error {
	if len(m.Formatters) == 0 {
		if err := write(m.Destination); err != nil {
			return err
		}
		return applyOpFormatters(m)
	}

	stagedDestination := filepath.Join(filepath.Dir(m.Destination), "."+filepath.Base(m.Destination)+".tmp")
	if err := write(stagedDestination); err != nil {
		return err
	}
	staged := m
	staged.Destination = stagedDestination
	if err := applyOpFormatters(staged); err != nil {
		_ = mut.Remove(stagedDestination)
		_ = mut.Remove(stagedDestination + ".pub")
		return err
	}
	if mut.DryRun() {
		return nil
	}
	if err := mut.Rename(stagedDestination, m.Destination); err != nil {
		return fmt.Errorf("install formatted output: %w", err)
	}
	if err := mut.Rename(stagedDestination+".pub", m.Destination+".pub"); err != nil {
		return fmt.Errorf("install formatted public output: %w", err)
	}
	return nil
}
