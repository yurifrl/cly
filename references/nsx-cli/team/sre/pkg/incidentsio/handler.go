package incidentsio

import (
	"context"
	"fmt"
)

// Config contains configuration for the incidentsio handler
type Config struct {
	// AuthToken is the API token for incident.io
	AuthToken string
}

// Handler is the main orchestrator for incident.io operations
type Handler struct {
	client *client
}

// NewHandler creates a new Handler with the provided configuration
func NewHandler(cfg Config) *Handler {
	return &Handler{
		client: newClient(cfg.AuthToken),
	}
}

// findLeaderFromRoleAssignments extracts the leader information from role assignments
func findLeaderFromRoleAssignments(assignments []RoleAssignment) Leader {
	for _, assignment := range assignments {
		if assignment.Role.RoleType == "lead" {
			return Leader{
				ID:    assignment.Assignee.ID,
				Name:  assignment.Assignee.Name,
				Email: assignment.Assignee.Email,
			}
		}
	}

	// Return an empty leader if no lead role is found
	return Leader{}
}

// List fetches and displays incidents based on the provided options
func (h *Handler) List(ctx context.Context, opts Options) error {
	// Create a renderer for the desired output format
	renderer := newRenderer(opts)

	// Fetch incidents using the options directly
	incidents, err := h.client.ListIncidents(ctx, opts)
	if err != nil {
		return fmt.Errorf("error fetching incidents: %w", err)
	}

	// Apply client-side filtering to ensure dates are within range
	filteredIncidents := []Incident{}
	for _, incident := range incidents {
		// Only include incidents created within the date range
		if (incident.CreatedAt.Equal(opts.StartDate) || incident.CreatedAt.After(opts.StartDate)) &&
			(incident.CreatedAt.Equal(opts.EndDate) || incident.CreatedAt.Before(opts.EndDate)) {
			filteredIncidents = append(filteredIncidents, incident)
		}
	}

	// Fetch follow-ups for each incident and build bundles
	var bundles []IncidentBundle
	for _, incident := range filteredIncidents {
		// Fetch followups for this incident
		followups, err := h.client.GetFollowups(ctx, incident.ID)
		if err != nil {
			return fmt.Errorf("error fetching followups for incident %s: %w", incident.Reference, err)
		}

		// Extract leader from incident role assignments
		leader := findLeaderFromRoleAssignments(incident.IncidentRoleAssignments)

		// Create an incident bundle
		bundles = append(bundles, IncidentBundle{
			Incident:  incident,
			Followups: followups,
			Leader:    leader,
		})
	}

	// Render the bundles in the desired format
	return renderer.Render(bundles)
}
