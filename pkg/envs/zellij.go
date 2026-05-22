package envs

import "github.com/yurifrl/cly/pkg/result"

// zellij-injected env vars. ZELLIJ is the marker variable; it is set
// to the Zellij version string but callers usually only care about
// presence.

const (
	zellijKey        = "ZELLIJ"
	zellijSessionKey = "ZELLIJ_SESSION_NAME"
	zellijPaneKey    = "ZELLIJ_PANE_ID"
)

// Zellij returns the raw value of $ZELLIJ (the Zellij version when
// running inside a session), or Empty otherwise. Most callers want
// InZellij instead.
func Zellij() result.Result[string] {
	return readString(zellijKey)
}

// InZellij reports whether the process is running inside a Zellij
// session. Mirrors the historic IsInZellij helper: requires $ZELLIJ
// to be set to a non-empty value.
func InZellij() bool {
	return Zellij().IsOk()
}

// ZellijSession returns the current Zellij session name, or Empty
// when not running in Zellij.
func ZellijSession() result.Result[string] {
	return readString(zellijSessionKey)
}

// ZellijPane returns the Zellij pane ID for the current shell, or
// Empty when not running in Zellij.
func ZellijPane() result.Result[string] {
	return readString(zellijPaneKey)
}
