package agentsession

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

type foundSession struct {
	ID           string `json:"id"`
	Provider     string `json:"provider"`
	JsonlPath    string `json:"jsonl_path"`
	Mtime        string `json:"mtime"`
	Cwd          string `json:"cwd,omitempty"`
	Name         string `json:"name,omitempty"`
	Description  string `json:"description,omitempty"`
	SavedPath    string `json:"saved_path,omitempty"`
	FirstUserMsg string `json:"first_user_msg,omitempty"`
}

func findSessionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "find-session <id>",
		Short: "Find a session by ID across Pi and Claude storage and print info",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			files, err := discoverJsonl()
			if err != nil {
				return err
			}

			catalog, _ := Load(filePathFn())

			var matches []foundSession
			for _, f := range files {
				id := extractIDFromJsonlPath(f.Path)
				if id != query && !strings.HasPrefix(id, query) {
					continue
				}
				firstMsg, _, _ := extractExcerpt(f.Path)
				fs := foundSession{
					ID:           id,
					Provider:     f.Provider,
					JsonlPath:    f.Path,
					Mtime:        f.Mtime.Format("2006-01-02 15:04"),
					Cwd:          readCwdFromJsonl(f.Path),
					FirstUserMsg: firstMsg,
				}
				if e := FindByIDAny(catalog, id); e != nil {
					fs.Name = e.Name
					fs.Description = e.Description
					fs.SavedPath = e.Path
				}
				matches = append(matches, fs)
			}

			if len(matches) == 0 {
				return fmt.Errorf("no session found matching %q", query)
			}
			return jsonOut(cmd, matches)
		},
	}
}
