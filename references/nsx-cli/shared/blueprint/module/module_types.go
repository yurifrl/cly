package module

// ModuleType represents the category of a module
type ModuleType string

const (
	DatabaseModule      ModuleType = "Database"
	HandlerModule       ModuleType = "Handler"
	CachingModule       ModuleType = "Caching"
	MessagingModule     ModuleType = "Messaging"
	ObservabilityModule ModuleType = "Observability"
	SecurityModule      ModuleType = "Security"
)
