package envs

import "strings"

// Field represents a single environment variable.
type Field struct {
	Label   string
	Value   string
	Section string
}

type item struct {
	Fields []struct {
		Label   string `json:"label"`
		Value   string `json:"value"`
		Type    string `json:"type"`
		Section *struct {
			Label string `json:"label"`
		} `json:"section"`
	} `json:"fields"`
}

// filterValidFields returns only fields with valid env labels and non-empty values.
func filterValidFields(fields []Field) []Field {
	result := make([]Field, 0, len(fields))
	for _, field := range fields {
		if !validEnvLabel(field.Label) || field.Value == "" || field.Value == "null" {
			continue
		}
		result = append(result, field)
	}
	return result
}

func validEnvLabel(label string) bool {
	if label == "" || label == "username" || label == "password" || label == "notesPlain" || label == "server" ||
		label == "port" ||
		label == "database" {
		return false
	}
	for i, ch := range label {
		if ch != '_' && (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') && (i == 0 || ch < '0' || ch > '9') {
			return false
		}
	}
	return true
}

// decodeFields maps a parsed 1Password item into env fields, lowercasing the
// owning section label so GENERAL_/WORK_/PERSONAL_ prefixes are stable.
func decodeFields(parsed item) []Field {
	fields := make([]Field, 0, len(parsed.Fields))
	for _, raw := range parsed.Fields {
		section := ""
		if raw.Section != nil {
			section = strings.ToLower(raw.Section.Label)
		}
		fields = append(fields, Field{Label: raw.Label, Value: raw.Value, Section: section})
	}
	return fields
}
