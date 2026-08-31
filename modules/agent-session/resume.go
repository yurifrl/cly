package agentsession

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

func resumeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resume <name|id>",
		Short: "Resume a saved session by name or ID",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			sessions, err := loadScopedSessions(cmd)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			names := make([]string, 0, len(sessions))
			for _, e := range sessions {
				names = append(names, e.Name)
			}
			return names, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			provider, err := providerFromCmd(cmd)
			if err != nil {
				return err
			}

			query := args[0]
			sessions, err := loadScopedSessions(cmd)
			if err != nil {
				return err
			}

			entry := findEntry(sessions, provider.Name, query)
			if entry == nil {
				return fmt.Errorf("%s session %q not found", provider.Name, query)
			}

			return resumeEntry(entry, provider, false)
		},
	}
}

func resumeEntry(entry *Entry, provider Provider, yolo bool) error {
	if err := ensureProviderSupportsYolo(provider, yolo); err != nil {
		return err
	}

	entry.Provider = provider.Name
	entry.SavedAt = time.Now()
	sessions, err := Load(filePathFn())
	if err != nil {
		return err
	}
	upsertEntry(sessions, *entry)
	if err := Save(filePathFn(), sessions); err != nil {
		return err
	}

	if entry.Path == "" {
		if p := lookupCwdForSession(entry.ID, provider.Name); p != "" {
			entry.Path = p
		}
	}
	if entry.Path == "" {
		return fmt.Errorf("session %s has no recorded path; run `cly as save` from the project dir", entry.ID)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	if cwd != entry.Path {
		if err := os.Chdir(entry.Path); err != nil {
			return fmt.Errorf("chdir %s: %w", entry.Path, err)
		}
	}
	return execProvider(entry, provider, yolo)
}

// lookupCwdForSession finds the jsonl file for a session and reads the
// recorded cwd from its first lines. omp, pi, and claude all write
// `"cwd":"..."` near the top of the file.
func lookupCwdForSession(id, provider string) string {
	if id == "" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	var candidates []string
	if provider == "" || provider == "omp" {
		m, _ := filepath.Glob(filepath.Join(home, ".omp", "agent", "sessions", "*", "*"+id+".jsonl"))
		candidates = append(candidates, m...)
	}
	if provider == "" || provider == "pi" {
		m, _ := filepath.Glob(filepath.Join(home, ".pi", "agent", "sessions", "*", "*"+id+".jsonl"))
		candidates = append(candidates, m...)
	}
	if provider == "" || provider == "claude" {
		m, _ := filepath.Glob(filepath.Join(home, ".claude", "projects", "*", id+".jsonl"))
		candidates = append(candidates, m...)
	}
	for _, p := range candidates {
		if cwd := readCwdFromJsonl(p); cwd != "" {
			return cwd
		}
	}
	return ""
}

func readCwdFromJsonl(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 4*1024*1024)
	for i := 0; i < 5 && scanner.Scan(); i++ {
		var v map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &v); err != nil {
			continue
		}
		if s, _ := v["cwd"].(string); s != "" {
			return s
		}
	}
	return ""
}
