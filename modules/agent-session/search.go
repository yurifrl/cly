package agentsession

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/pkg/ai"
	"github.com/yurifrl/cly/pkg/config"
)

// searchCmd registers `cly agent-session search [query]`. It opens an
// interactive TUI that fuzzy-filters across cached session metadata and
// body excerpts, optionally re-ranking with the configured LLM.
//
// On Enter, the chosen session is resumed via the existing resumeEntry flow
// (chdir + exec the provider). On the user pressing esc, search exits and
// returns to the original shell with no side effects.
func searchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "search [query]",
		Aliases: []string{"find"},
		Short:   "Interactively search past sessions (local + AI)",
		Long: `Search across past pi/claude sessions by name, description, path, and
bounded body excerpts. When an LLM is configured, the top candidates are
re-ranked by the model — without ever sending full session content.`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			query := ""
			if len(args) > 0 {
				query = strings.Join(args, " ")
			}
			useAI, _ := cmd.Flags().GetBool("ai")
			refresh, _ := cmd.Flags().GetBool("refresh")
			provider, _ := cmd.Flags().GetString(providerFlag)
			folderFlag, _ := cmd.Flags().GetString("folder")

			prefs := loadSearchPrefs()
			cwd, _ := os.Getwd()
			// Folder: explicit flag wins; otherwise fall back to the persisted
			// scope preference (current dir vs global).
			folder := resolveFolder(folderFlag, cwd)
			if !cmd.Flags().Changed("folder") && config.GetBool(prefFolderScope) {
				folder = cwd
			}
			// AI: explicit flag wins; otherwise use the persisted preference.
			if !cmd.Flags().Changed("ai") {
				useAI = config.GetBool(prefAI)
			}

			catalog, err := Load(filePathFn())
			if err != nil {
				return err
			}

			idx := loadSearchIndex()
			if refresh || len(idx.Sessions) == 0 {
				fmt.Fprintf(os.Stderr, "Indexing sessions…\n")
				if err := rebuildSearchIndex(idx, catalog); err != nil {
					return fmt.Errorf("rebuild index: %w", err)
				}
				if err := saveSearchIndex(idx); err != nil {
					fmt.Fprintf(os.Stderr, "warn: save index failed: %v\n", err)
				}
			} else {
				if err := rebuildSearchIndex(idx, catalog); err != nil {
					fmt.Fprintf(os.Stderr, "warn: refresh index failed: %v\n", err)
				}
				_ = saveSearchIndex(idx)
			}

			aiOn := useAI && ai.HasAPIKeyFor("agent_session.search")

			chosen, err := runSearchTUI(idx, provider, folder, cwd, query, aiOn, prefs)
			if err != nil {
				return err
			}
			if chosen == nil {
				return nil
			}

			provObj, ok := loadProvidersFn()[chosen.Provider]
			if !ok {
				return fmt.Errorf("unknown provider %q for session %s", chosen.Provider, chosen.ID)
			}
			entry := Entry{
				ID:          chosen.ID,
				Name:        chosen.Name,
				Provider:    chosen.Provider,
				Path:        chosen.Path,
				Description: chosen.Description,
				SavedAt:     chosen.SavedAt,
			}
			fmt.Fprintf(os.Stderr, "Resuming %s session %s in %s…\n",
				chosen.Provider, chosen.ID[:8], chosen.Path)
			return resumeEntry(&entry, provObj, false)
		},
	}
	cmd.Flags().Bool("ai", false, "Enable LLM re-rank (off by default; uses configured LLM)")
	cmd.Flags().Bool("refresh", false, "Force a full re-scan of session files (ignore cache)")
	cmd.Flags().StringP("folder", "f", "", "Limit search to sessions under a folder (\".\" = current dir; empty = global)")
	return cmd
}

// resolveFolder maps the --folder flag to an absolute path filter. "" stays
// global, "." (or any relative path) resolves against cwd.
func resolveFolder(flag, cwd string) string {
	if flag == "" {
		return ""
	}
	if flag == "." {
		return cwd
	}
	if filepath.IsAbs(flag) {
		return filepath.Clean(flag)
	}
	return filepath.Join(cwd, flag)
}
