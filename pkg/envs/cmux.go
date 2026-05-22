package envs

import "github.com/yurifrl/cly/pkg/result"

// cmux-injected env vars. cmux sets these in every shell it spawns so
// downstream tools can identify the surrounding tab/surface without
// shelling out to `cmux identify`.

const (
	cmuxSurfaceIDKey   = "CMUX_SURFACE_ID"
	cmuxTabIDKey       = "CMUX_TAB_ID"
	cmuxWorkspaceIDKey = "CMUX_WORKSPACE_ID"
)

// CmuxSurfaceID returns the cmux surface UUID for the current shell,
// or Empty when not running under cmux.
func CmuxSurfaceID() result.Result[string] {
	return readString(cmuxSurfaceIDKey)
}

// CmuxTabID returns the cmux tab UUID for the current shell, or
// Empty when not running under cmux.
func CmuxTabID() result.Result[string] {
	return readString(cmuxTabIDKey)
}

// CmuxWorkspaceID returns the cmux workspace UUID for the current
// shell, or Empty when not running under cmux.
func CmuxWorkspaceID() result.Result[string] {
	return readString(cmuxWorkspaceIDKey)
}

// InCmux reports whether the process is running inside cmux. This is
// inferred from the presence (and non-empty value) of
// CMUX_WORKSPACE_ID, which cmux sets for any process spawned inside a
// workspace.
func InCmux() bool {
	return CmuxWorkspaceID().IsOk()
}
