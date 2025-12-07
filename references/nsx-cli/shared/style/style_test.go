package style

import (
	"testing"
)

func TestFormatKeyValue(t *testing.T) {
	result := FormatKeyValue("key", "value")
	if result == "" {
		t.Error("FormatKeyValue should return non-empty string")
	}
}

func TestFormatStatus(t *testing.T) {
	tests := []struct {
		status   string
		text     string
		expected string
	}{
		{"success", "OK", "OK"},
		{"error", "Failed", "Failed"},
		{"warning", "Warning", "Warning"},
		{"info", "Info", "Info"},
		{"unknown", "Unknown", "Unknown"},
	}

	for _, tt := range tests {
		result := FormatStatus(tt.status, tt.text)
		if result == "" {
			t.Errorf("FormatStatus(%s, %s) should return non-empty string", tt.status, tt.text)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	result := FormatDuration("30s")
	if result == "" {
		t.Error("FormatDuration should return non-empty string")
	}
}

func TestFormatTableHeader(t *testing.T) {
	result := FormatTableHeader("Header")
	if result == "" {
		t.Error("FormatTableHeader should return non-empty string")
	}
}

func TestFormatTableRow(t *testing.T) {
	result := FormatTableRow("Row", true)
	if result == "" {
		t.Error("FormatTableRow should return non-empty string")
	}
}
