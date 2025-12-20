package notify

// Notification represents a notification to be sent
type Notification struct {
	Title    string // Main title of the notification
	Subtitle string // Subtitle (may be combined with title on some platforms)
	Message  string // Body message
	Sound    string // Sound name (Glass, Blow, Submarine, Ping)
	Group    string // Notification group ID
}
