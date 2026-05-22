package envs

import "github.com/yurifrl/cly/pkg/result"

// Session-related env vars.
//
// CLY_SESSION_NAME is the canonical name. CLAUDE_SESSION_NAME is a
// legacy alias kept so existing claude-centric tooling and config
// templates keep working. Writes propagate to both names.

const (
	sessionNameKey    = "CLY_SESSION_NAME"
	sessionNameLegacy = "CLAUDE_SESSION_NAME"
)

// SessionName returns the active session name. The canonical
// CLY_SESSION_NAME wins; CLAUDE_SESSION_NAME is consulted as a
// fallback. An unset session is Empty.
func SessionName() result.Result[string] {
	return readString(sessionNameKey, sessionNameLegacy)
}

// SetSessionName assigns name to both CLY_SESSION_NAME and the legacy
// CLAUDE_SESSION_NAME so older readers see the same value.
func SetSessionName(name string) error {
	return write(name, sessionNameKey, sessionNameLegacy)
}

// HasSessionName reports whether either the canonical or legacy
// session name var is set, regardless of value.
func HasSessionName() bool {
	return has(sessionNameKey, sessionNameLegacy)
}

// UnsetSessionName clears both the canonical and legacy session name
// vars from the active source.
func UnsetSessionName() {
	clear(sessionNameKey, sessionNameLegacy)
}
