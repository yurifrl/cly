// config.go centralises piwrap config keys and provides typed
// accessors with sensible defaults. Reads via pkg/config (Viper).
package piwrap

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/yurifrl/cly/pkg/config"
)

// importConfig holds the resolved values for the session_import
// feature. Each field has a default applied when the corresponding
// config key is unset or blank.
type importConfig struct {
	Override      bool
	QuarantineDir string
	SearchScope   SearchScope
}

// loadImportConfig reads modules.piwrap.session_import.* with
// fallbacks. Path-shaped defaults expand `~` to the current user's
// home directory.
func loadImportConfig() importConfig {
	c := importConfig{
		Override:    config.GetBool("modules.piwrap.session_import.override"),
		SearchScope: ScopeCwd,
	}

	q := strings.TrimSpace(config.GetString("modules.piwrap.session_import.quarantine_dir"))
	if q == "" {
		home, _ := os.UserHomeDir()
		q = filepath.Join(home, ".local", "share", "cly", "trash", "pi-sessions")
	} else if strings.HasPrefix(q, "~/") {
		home, _ := os.UserHomeDir()
		q = filepath.Join(home, q[2:])
	}
	c.QuarantineDir = q

	scope := strings.TrimSpace(config.GetString("modules.piwrap.session_import.search_scope"))
	if scope == "all" {
		c.SearchScope = ScopeAll
	}

	return c
}
