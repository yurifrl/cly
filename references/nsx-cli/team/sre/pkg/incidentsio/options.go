package incidentsio

import (
	"fmt"
	"time"
)

// Options contains user-provided configuration for incident operations
type Options struct {
	// Format specifies the output format (markdown, charm, json, text)
	Format string

	// Status is an optional filter for incident status
	Status string

	// StartDate is the beginning of the date range for incidents
	StartDate time.Time

	// EndDate is the end of the date range for incidents
	EndDate time.Time

	// Cursor is used for pagination in API requests
	Cursor string

	// CustomParams holds additional API query parameters
	CustomParams map[string]string
}

// NewOptions creates a new Options with initialized fields
func NewOptions() Options {
	return Options{
		CustomParams: make(map[string]string),
	}
}

// WithFormat sets the output format
func (o Options) WithFormat(format string) Options {
	o.Format = format
	return o
}

// WithStatus sets the incident status filter
func (o Options) WithStatus(status string) Options {
	o.Status = status
	return o
}

// WithStartDate sets the start date for incident range
func (o Options) WithStartDate(date time.Time) Options {
	o.StartDate = date
	return o
}

// WithEndDate sets the end date for incident range
func (o Options) WithEndDate(date time.Time) Options {
	o.EndDate = date
	return o
}

// WithCursor sets the pagination cursor
func (o Options) WithCursor(cursor string) Options {
	o.Cursor = cursor
	return o
}

// WithCustomParam adds a custom query parameter
func (o Options) WithCustomParam(key, value string) Options {
	o.CustomParams[key] = value
	return o
}

// BuildURL constructs the query string for incident listing
func (o Options) BuildURL() string {
	url := fmt.Sprintf("/incidents?created_at_start_time=%s&created_at_end_time=%s",
		o.StartDate.Format(time.RFC3339),
		o.EndDate.Format(time.RFC3339))

	if o.Status != "" {
		url = fmt.Sprintf("%s&incident_status_name=%s", url, o.Status)
	}

	if o.Cursor != "" {
		url = fmt.Sprintf("%s&after_cursor=%s", url, o.Cursor)
	}

	// Add any custom parameters
	for key, value := range o.CustomParams {
		url = fmt.Sprintf("%s&%s=%s", url, key, value)
	}

	return url
}
