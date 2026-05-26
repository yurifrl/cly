package notify

// Notification represents a notification to be sent
type Notification struct {
	Title   string   // Main title of the notification
	Message string   // Body message
	Sound   string   // Sound name (Glass, Blow, Submarine, Ping)
	Group   string   // Notification group ID; also used to route ActionEvents back to the caller
	Actions []Action // Optional action buttons; empty = no buttons
}

// Action is a user-clickable button on a notification.
// Backends that don't support actions (beeep, zellij) ignore this field.
type Action struct {
	ID    string // Stable identifier returned in ActionEvent (e.g. "snooze", "retry")
	Title string // User-visible label
}

// ActionEvent is emitted by a Notifier when the user clicks an action button.
// It is the caller's responsibility to interpret Group and ActionID semantics.
type ActionEvent struct {
	Group    string // Mirrors Notification.Group
	ActionID string // Matches one of the Action.ID values that were sent
}
