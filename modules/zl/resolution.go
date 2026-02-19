package zl

// Function variables for testing
var (
	loadZlConfigFunc = LoadZlConfig
	queryZoxideFunc  = QueryZoxide
)

// ResolveDirectory resolves a directory for a session name
// Priority: 1. Explicit mapping from config, 2. Zoxide query, 3. Empty string
func ResolveDirectory(session string) string {
	cfg := loadZlConfigFunc()

	// Check explicit mapping first
	if dir, ok := cfg.SessionDirs[session]; ok {
		return dir
	}

	// Fall back to zoxide if enabled
	if !cfg.AutoZoxide {
		return ""
	}

	dir, err := queryZoxideFunc(session)
	if err != nil {
		return ""
	}

	return dir
}
