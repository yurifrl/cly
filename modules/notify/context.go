package notify

import "os"

// buildContextString builds a context string from session names
// Format: "[zellij-session] claude-session"
func buildContextString() string {
	zellijSession := os.Getenv("ZELLIJ_SESSION_NAME")
	claudeSession := os.Getenv("CLAUDE_SESSION_NAME")

	var contextStr string

	if zellijSession != "" {
		contextStr = "[" + zellijSession + "]"
	}

	if claudeSession != "" {
		if contextStr != "" {
			contextStr = contextStr + " " + claudeSession
		} else {
			contextStr = claudeSession
		}
	}

	return contextStr
}
