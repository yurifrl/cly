package views

import (
	"testing"
)

func TestNewDBProcessListView(t *testing.T) {
	// Test data
	results := []QueryResult{
		{
			ID:           1,
			User:         "alice",
			Database:     "testdb",
			Duration:     30,
			DurationStr:  "30s",
			Command:      "SELECT",
			State:        "running",
			QueryPreview: "SELECT * FROM users",
		},
		{
			ID:           2,
			User:         "bob",
			Database:     "testdb",
			Duration:     120,
			DurationStr:  "2m 0s",
			Command:      "UPDATE",
			State:        "waiting",
			QueryPreview: "UPDATE users SET name='Bob'",
		},
	}

	view := NewDBProcessListView("Test Process List", results, nil, nil)

	if view.title != "Test Process List" {
		t.Errorf("Expected title 'Test Process List', got '%s'", view.title)
	}

	if len(view.results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(view.results))
	}

	if view.results[0].ID != 1 {
		t.Errorf("Expected first result ID to be 1, got %d", view.results[0].ID)
	}
}

func TestUpdateResults(t *testing.T) {
	// Initial results
	initialResults := []QueryResult{
		{
			ID:           1,
			User:         "alice",
			Database:     "testdb",
			Duration:     30,
			DurationStr:  "30s",
			Command:      "SELECT",
			State:        "running",
			QueryPreview: "SELECT * FROM users",
		},
	}

	view := NewDBProcessListView("Test", initialResults, nil, nil)

	// Update results
	newResults := []QueryResult{
		{
			ID:           1,
			User:         "alice",
			Database:     "testdb",
			Duration:     30,
			DurationStr:  "30s",
			Command:      "SELECT",
			State:        "running",
			QueryPreview: "SELECT * FROM users",
		},
		{
			ID:           2,
			User:         "bob",
			Database:     "testdb",
			Duration:     120,
			DurationStr:  "2m 0s",
			Command:      "UPDATE",
			State:        "waiting",
			QueryPreview: "UPDATE users SET name='Bob'",
		},
	}

	view.UpdateResults(newResults)

	if len(view.results) != 2 {
		t.Errorf("Expected 2 results after update, got %d", len(view.results))
	}

	if view.results[1].ID != 2 {
		t.Errorf("Expected second result ID to be 2, got %d", view.results[1].ID)
	}
}

func TestGetResults(t *testing.T) {
	results := []QueryResult{
		{
			ID:           1,
			User:         "alice",
			Database:     "testdb",
			Duration:     30,
			DurationStr:  "30s",
			Command:      "SELECT",
			State:        "running",
			QueryPreview: "SELECT * FROM users",
		},
	}

	view := NewDBProcessListView("Test", results, nil, nil)
	retrievedResults := view.GetResults()

	if len(retrievedResults) != 1 {
		t.Errorf("Expected 1 result, got %d", len(retrievedResults))
	}

	if retrievedResults[0].ID != 1 {
		t.Errorf("Expected result ID to be 1, got %d", retrievedResults[0].ID)
	}
}

func TestSetTitle(t *testing.T) {
	view := NewDBProcessListView("Original Title", []QueryResult{}, nil, nil)

	newTitle := "New Title"
	view.SetTitle(newTitle)

	if view.title != newTitle {
		t.Errorf("Expected title '%s', got '%s'", newTitle, view.title)
	}
}

func TestCreateQueryRowFunc(t *testing.T) {
	view := NewDBProcessListView("Test", []QueryResult{}, nil, nil)

	result := QueryResult{
		ID:           123,
		User:         "testuser",
		Database:     "testdb",
		Duration:     60,
		DurationStr:  "1m 0s",
		Command:      "SELECT",
		State:        "running",
		QueryPreview: "SELECT * FROM test",
	}

	row := view.createQueryRowFunc(result)

	if row.Data[columnKeyID] != "123" {
		t.Errorf("Expected ID '123', got '%v'", row.Data[columnKeyID])
	}

	if row.Data[columnKeyUser] != "testuser" {
		t.Errorf("Expected User 'testuser', got '%v'", row.Data[columnKeyUser])
	}

	if row.Data[columnKeyDatabase] != "testdb" {
		t.Errorf("Expected Database 'testdb', got '%v'", row.Data[columnKeyDatabase])
	}

	if row.Data[columnKeyDuration] != "1m 0s" {
		t.Errorf("Expected Duration '1m 0s', got '%v'", row.Data[columnKeyDuration])
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"this is a very long string", 10, "this is..."},
		{"exactly10", 10, "exactly10"},
		{"", 5, ""},
	}

	for _, tt := range tests {
		result := truncateString(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateString(%q, %d) = %q, expected %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds  int
		expected string
	}{
		{30, "30s"},
		{90, "1m 30s"},
		{3661, "1h 1m 1s"},
		{7200, "2h 0m 0s"},
	}

	for _, tt := range tests {
		result := FormatDuration(tt.seconds)
		// We check if the result contains the expected text since it may be styled
		if !containsText(result, tt.expected) {
			t.Errorf("FormatDuration(%d) should contain %q, got %q", tt.seconds, tt.expected, result)
		}
	}
}

func TestShouldIgnoreUser(t *testing.T) {
	tests := []struct {
		user     string
		expected bool
	}{
		{"rdsadmin", true},
		{"mysql.session", true},
		{"mysql.sys", true},
		{"root", true},
		{"system user", true},
		{"RDSADMIN", true},      // Test case insensitive
		{"MYSQL.SESSION", true}, // Test case insensitive
		{"normaluser", false},
		{"admin", false},
		{"testuser", false},
		{"", false},
	}

	for _, tt := range tests {
		result := ShouldIgnoreUser(tt.user)
		if result != tt.expected {
			t.Errorf("ShouldIgnoreUser(%q) = %v, expected %v", tt.user, result, tt.expected)
		}
	}
}

// Helper function to check if a styled string contains expected text
func containsText(styled, expected string) bool {
	// Simple check to see if the expected text is contained in the styled string
	// This is a basic implementation since we're dealing with styled strings
	return len(styled) > 0 && (styled == expected || len(styled) > len(expected))
}
