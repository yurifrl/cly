package envs

import "github.com/yurifrl/cly/pkg/result"

// Notification-related env vars.

const soundKey = "SOUND"

// Sound returns the override path/name for the notification sound,
// or Empty when the caller should use its configured default.
func Sound() result.Result[string] {
	return readString(soundKey)
}
