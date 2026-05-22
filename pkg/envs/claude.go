package envs

import "github.com/yurifrl/cly/pkg/result"

// Claude-related env vars consumed by cly's notification and
// debug tooling.

const claudeVerboseKey = "CLAUDE_VERBOSE"

// ClaudeVerbose returns the parsed value of CLAUDE_VERBOSE (a bool
// flag controlling verbose output in claude-related modules).
//
//	Empty -> caller should apply its own default
//	Error -> the value was set but unparseable; carries *ParseError
func ClaudeVerbose() result.Result[bool] {
	return readBool(claudeVerboseKey)
}
