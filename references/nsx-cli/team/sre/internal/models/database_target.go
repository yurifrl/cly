package models

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Type label colors
	instanceStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#3b82f6")).Bold(true) // Blue
	clusterStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#06b6d4")).Bold(true) // Cyan

	// Database identifier color
	identifierStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#f59e0b")).Bold(true) // Orange

	// Engine type color
	engineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#8b5cf6")) // Purple

	// Endpoint color
	endpointStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7280")) // Gray

	// Status colors
	statusAvailableStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#10b981")) // Green
	statusWarningStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#f59e0b")) // Orange
	statusErrorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")) // Red
	statusDefaultStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7280")) // Gray
)

// DatabaseTarget represents a selectable database (instance or cluster)
type DatabaseTarget struct {
	Type          string
	Identifier    string
	Engine        string
	EngineVersion string
	Endpoint      string
	Port          int32
	Status        string
	SecretArn     *string
	IsCluster     bool
}

// DisplayString returns formatted string for UI dropdown
func (d *DatabaseTarget) DisplayString() string {
	// Type label with color
	var typeLabel string
	if d.IsCluster {
		typeLabel = clusterStyle.Render("[CLUSTER]")
	} else {
		typeLabel = instanceStyle.Render("[INSTANCE]")
	}

	// Database identifier with color
	identifier := identifierStyle.Render(d.Identifier)

	// Engine type with color
	engine := engineStyle.Render(fmt.Sprintf("(%s)", d.Engine))

	// Endpoint with color
	endpoint := endpointStyle.Render(fmt.Sprintf("%s:%d", d.Endpoint, d.Port))

	// Status with color based on status value
	var status string
	switch d.Status {
	case "available":
		status = statusAvailableStyle.Render(fmt.Sprintf("[%s]", d.Status))
	case "backing-up", "modifying", "configuring-enhanced-monitoring", "configuring-log-exports":
		status = statusWarningStyle.Render(fmt.Sprintf("[%s]", d.Status))
	case "failed", "storage-full", "incompatible-parameters", "incompatible-restore":
		status = statusErrorStyle.Render(fmt.Sprintf("[%s]", d.Status))
	default:
		status = statusDefaultStyle.Render(fmt.Sprintf("[%s]", d.Status))
	}

	return fmt.Sprintf("%s %s %s - %s %s",
		typeLabel, identifier, engine, endpoint, status)
}

// IsMySQL returns true if database is MySQL-based
func (d *DatabaseTarget) IsMySQL() bool {
	return d.Engine == "mysql" || d.Engine == "aurora-mysql"
}

// IsPostgreSQL returns true if database is PostgreSQL-based
func (d *DatabaseTarget) IsPostgreSQL() bool {
	return d.Engine == "postgres" || d.Engine == "aurora-postgresql"
}

// GetDefaultDatabase returns the default database name for connection
func (d *DatabaseTarget) GetDefaultDatabase() string {
	if d.IsPostgreSQL() {
		return "postgres"
	}
	return "" // MySQL doesn't require initial database
}

// GetClientBinary returns the CLI tool name for this database type
func (d *DatabaseTarget) GetClientBinary() string {
	if d.IsMySQL() {
		return "mysql"
	}
	return "psql"
}
