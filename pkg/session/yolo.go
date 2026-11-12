package session

import _ "embed"

//go:embed prompts/safe-autonomous-mode.md
var safeAutonomousPrompt string

// YoloArgs returns the args to pass to claude for dangerously-skip-permissions mode.
func YoloArgs() []string {
	return []string{"--dangerously-skip-permissions", "--append-system-prompt", safeAutonomousPrompt}
}

// SafeAutonomousPrompt returns the safe-autonomous-mode preprompt that
// yolo sessions get appended to their system prompt.
func SafeAutonomousPrompt() string {
	return safeAutonomousPrompt
}
