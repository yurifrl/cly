package notify

// Notification represents a notification to be sent.
type Notification struct {
	Title   string // Main title
	Message string // Body message
	Sound   string // macOS sound name (Glass, Basso, Sosumi, Ping, ...); empty = default
	Group   string // Notification group ID; same group replaces previous
}
