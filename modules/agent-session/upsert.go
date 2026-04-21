package agentsession

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func upsertCmd() *cobra.Command {
	var flagName string
	var flagDesc string
	var flagSet []string
	var flagMeta string
	var flagOverride bool

	cmd := &cobra.Command{
		Use:     "upsert <id> [name] [description]",
		Aliases: []string{"save"},
		Short:   "Create or update a session (always returns full entry as JSON)",
		Args:    cobra.RangeArgs(1, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			providerName, err := cmd.Flags().GetString(providerFlag)
			if err != nil {
				return err
			}
			id := args[0]
			if normalizeProvider(providerName) == "all" {
				if detected := detectProviderByID(id); detected != "" {
					providerName = detected
				} else {
					providerName = defaultProvider()
				}
			}
			provider, err := providerByName(providerName)
			if err != nil {
				return err
			}

			// Name from positional or flag
			name := flagName
			if name == "" && len(args) > 1 {
				name = args[1]
			}

			// Description from positional or flag
			desc := flagDesc
			if desc == "" && len(args) > 2 {
				desc = args[2]
			}

			cwd, err := os.Getwd()
			if err != nil {
				return err
			}

			// Parse meta from --set and --meta
			meta, err := parseMeta(flagSet, flagMeta)
			if err != nil {
				return err
			}

			filePath := filePathFn()
			sessions, err := Load(filePath)
			if err != nil {
				return err
			}

			// Try to find existing entry by ID
			entry := FindByIDAny(sessions, id)
			if entry == nil {
				// Try by name+provider if name given
				if name != "" {
					entry = FindByNameForProvider(sessions, provider.Name, name)
				}
			}

			if entry == nil {
				// Create new
				entry = &Entry{
					ID:       id,
					Name:     name,
					Provider: provider.Name,
					Path:     cwd,
				}
			} else {
				// Update existing
				entry.ID = id
				if name != "" {
					// Only update name if entry has no name yet, or --override is set
					if entry.Name == "" || flagOverride {
						entry.Name = name
					}
				}
				entry.Provider = provider.Name
				entry.Path = cwd
			}

			if desc != "" {
				entry.Description = desc
			}
			entry.SavedAt = time.Now()

			// Merge meta
			if len(meta) > 0 {
				if entry.Meta == nil {
					entry.Meta = make(map[string]string)
				}
				for k, v := range meta {
					entry.Meta[k] = v
				}
			}

			upsertEntry(sessions, *entry)
			if err := Save(filePath, sessions); err != nil {
				return err
			}

			return jsonOut(cmd, entry)
		},
	}

	cmd.Flags().StringVarP(&flagName, "name", "n", "", "Session name")
	cmd.Flags().StringVarP(&flagDesc, "description", "d", "", "Session description")
	cmd.Flags().StringArrayVar(&flagSet, "set", nil, "Set metadata key=value (repeatable)")
	cmd.Flags().StringVar(&flagMeta, "meta", "", `Set metadata as JSON object (e.g. '{"key":"value"}')`)
	cmd.Flags().BoolVar(&flagOverride, "override", false, "Override existing name when --name is passed")

	return cmd
}

// parseMeta combines --set key=value flags and --meta JSON into a single map.
func parseMeta(setFlags []string, metaJSON string) (map[string]string, error) {
	result := make(map[string]string)

	// Parse --meta JSON
	if metaJSON != "" {
		var m map[string]string
		if err := json.Unmarshal([]byte(metaJSON), &m); err != nil {
			return nil, fmt.Errorf("invalid --meta JSON: %w", err)
		}
		for k, v := range m {
			result[k] = v
		}
	}

	// Parse --set key=value (overrides --meta)
	for _, kv := range setFlags {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid --set format %q, expected key=value", kv)
		}
		result[parts[0]] = parts[1]
	}

	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}
