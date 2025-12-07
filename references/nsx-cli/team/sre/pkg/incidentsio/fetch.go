package incidentsio

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type client struct {
	baseURL    string
	httpClient *http.Client
	authToken  string
}

func newClient(authToken string) *client {
	return &client{
		baseURL:    "https://api.incident.io/v2",
		httpClient: &http.Client{Timeout: 30 * time.Second},
		authToken:  authToken,
	}
}

// IncidentsResponse is the response from the incidents endpoint
type incidentsResponse struct {
	Incidents  []Incident `json:"incidents"`
	Pagination struct {
		AfterCursor string `json:"after_cursor"`
	} `json:"pagination"`
}

// FollowupsResponse is the response from the follow-ups endpoint
type followupsResponse struct {
	Followups  []Followup `json:"follow_ups"`
	Pagination struct {
		AfterCursor string `json:"after_cursor"`
	} `json:"pagination"`
}

func (c *client) get(ctx context.Context, path string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *client) ListIncidents(ctx context.Context, opts Options) ([]Incident, error) {
	var allIncidents []Incident

	for {
		var response incidentsResponse
		if err := c.get(ctx, opts.BuildURL(), &response); err != nil {
			return nil, err
		}

		allIncidents = append(allIncidents, response.Incidents...)

		// If there's no after cursor, we've fetched all pages
		if response.Pagination.AfterCursor == "" {
			break
		}

		// Update cursor for next page
		opts = opts.WithCursor(response.Pagination.AfterCursor)
	}

	return allIncidents, nil
}

func (c *client) GetFollowups(ctx context.Context, incidentID string) ([]Followup, error) {
	var allFollowups []Followup
	afterCursor := ""

	for {
		url := fmt.Sprintf("/follow_ups?incident_id=%s", incidentID)

		if afterCursor != "" {
			url = fmt.Sprintf("%s&after_cursor=%s", url, afterCursor)
		}

		var response followupsResponse
		if err := c.get(ctx, url, &response); err != nil {
			return nil, err
		}

		allFollowups = append(allFollowups, response.Followups...)

		// If there's no after cursor, we've fetched all pages
		if response.Pagination.AfterCursor == "" {
			break
		}

		afterCursor = response.Pagination.AfterCursor
	}

	return allFollowups, nil
}
