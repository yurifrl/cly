package incidentsio

import "time"

// IncidentRole represents a role in incident.io
type IncidentRole struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	RoleType    string    `json:"role_type"`
	Shortform   string    `json:"shortform"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RoleAssignment represents an assignment of a role to a user in an incident
type RoleAssignment struct {
	Assignee User         `json:"assignee"`
	Role     IncidentRole `json:"role"`
}

// Incident represents an incident from incident.io
type Incident struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Reference string `json:"reference"`
	Status    struct {
		Name string `json:"name"`
	} `json:"incident_status"`
	CreatedAt               time.Time        `json:"created_at"`
	UpdatedAt               time.Time        `json:"updated_at"`
	IncidentRoleAssignments []RoleAssignment `json:"incident_role_assignments"`
}

// User represents a user in incident.io
type User struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	SlackUserID string `json:"slack_user_id"`
	Role        string `json:"role"`
}

// Followup represents a follow-up action for an incident
type Followup struct {
	ID                string    `json:"id"`
	IncidentID        string    `json:"incident_id"`
	Title             string    `json:"title"`
	Description       string    `json:"description"`
	CompletedAt       time.Time `json:"completed_at"`
	DueAt             time.Time `json:"due_at"`
	Status            string    `json:"status"`
	Assignee          User      `json:"assignee"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	ExternalIssueURL  string    `json:"external_issue_url"`
	ExternalIssueID   string    `json:"external_issue_id"`
	ExternalIssueType string    `json:"external_issue_type"`
}

// Leader represents a user in incident.io
type Leader struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// IncidentBundle groups an incident with its follow-ups and leader
type IncidentBundle struct {
	Incident  Incident   `json:"incident"`
	Followups []Followup `json:"followups"`
	Leader    Leader     `json:"leader"`
}
