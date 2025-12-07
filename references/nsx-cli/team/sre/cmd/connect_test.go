package cmd

import (
	"regexp"
	"strings"
	"testing"

	"github.com/NSXBet/nsx-cli/team/sre/internal/models"
)

// stripAnsi removes ANSI color codes from a string for testing
func stripAnsi(str string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiRegex.ReplaceAllString(str, "")
}

func TestDatabaseTarget_DisplayString(t *testing.T) {
	tests := []struct {
		name     string
		target   models.DatabaseTarget
		expected string
	}{
		{
			name: "MySQL instance",
			target: models.DatabaseTarget{
				Identifier: "prod-mysql",
				Engine:     "mysql",
				Endpoint:   "prod.abc.rds.amazonaws.com",
				Port:       3306,
				Status:     "available",
				IsCluster:  false,
			},
			expected: "[INSTANCE] prod-mysql (mysql) - prod.abc.rds.amazonaws.com:3306 [available]",
		},
		{
			name: "Aurora MySQL cluster",
			target: models.DatabaseTarget{
				Identifier: "prod-aurora",
				Engine:     "aurora-mysql",
				Endpoint:   "prod.cluster.rds.amazonaws.com",
				Port:       3306,
				Status:     "available",
				IsCluster:  true,
			},
			expected: "[CLUSTER] prod-aurora (aurora-mysql) - prod.cluster.rds.amazonaws.com:3306 [available]",
		},
		{
			name: "PostgreSQL instance",
			target: models.DatabaseTarget{
				Identifier: "dev-postgres",
				Engine:     "postgres",
				Endpoint:   "dev.xyz.rds.amazonaws.com",
				Port:       5432,
				Status:     "backing-up",
				IsCluster:  false,
			},
			expected: "[INSTANCE] dev-postgres (postgres) - dev.xyz.rds.amazonaws.com:5432 [backing-up]",
		},
		{
			name: "Aurora PostgreSQL cluster",
			target: models.DatabaseTarget{
				Identifier: "staging-aurora-pg",
				Engine:     "aurora-postgresql",
				Endpoint:   "staging.cluster.rds.amazonaws.com",
				Port:       5432,
				Status:     "available",
				IsCluster:  true,
			},
			expected: "[CLUSTER] staging-aurora-pg (aurora-postgresql) - staging.cluster.rds.amazonaws.com:5432 [available]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.target.DisplayString()
			// Strip ANSI color codes for comparison
			resultStripped := stripAnsi(result)
			if resultStripped != tt.expected {
				t.Errorf("DisplayString() = %v, want %v", resultStripped, tt.expected)
			}
		})
	}
}

func TestDatabaseTarget_IsMySQL(t *testing.T) {
	tests := []struct {
		name     string
		engine   string
		expected bool
	}{
		{"mysql engine", "mysql", true},
		{"aurora-mysql engine", "aurora-mysql", true},
		{"postgres engine", "postgres", false},
		{"aurora-postgresql engine", "aurora-postgresql", false},
		{"unknown engine", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := models.DatabaseTarget{Engine: tt.engine}
			result := target.IsMySQL()
			if result != tt.expected {
				t.Errorf("IsMySQL() for engine %s = %v, want %v", tt.engine, result, tt.expected)
			}
		})
	}
}

func TestDatabaseTarget_IsPostgreSQL(t *testing.T) {
	tests := []struct {
		name     string
		engine   string
		expected bool
	}{
		{"postgres engine", "postgres", true},
		{"aurora-postgresql engine", "aurora-postgresql", true},
		{"mysql engine", "mysql", false},
		{"aurora-mysql engine", "aurora-mysql", false},
		{"unknown engine", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := models.DatabaseTarget{Engine: tt.engine}
			result := target.IsPostgreSQL()
			if result != tt.expected {
				t.Errorf("IsPostgreSQL() for engine %s = %v, want %v", tt.engine, result, tt.expected)
			}
		})
	}
}

func TestDatabaseTarget_GetClientBinary(t *testing.T) {
	tests := []struct {
		name     string
		engine   string
		expected string
	}{
		{"mysql engine", "mysql", "mysql"},
		{"aurora-mysql engine", "aurora-mysql", "mysql"},
		{"postgres engine", "postgres", "psql"},
		{"aurora-postgresql engine", "aurora-postgresql", "psql"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := models.DatabaseTarget{Engine: tt.engine}
			result := target.GetClientBinary()
			if result != tt.expected {
				t.Errorf("GetClientBinary() for engine %s = %v, want %v", tt.engine, result, tt.expected)
			}
		})
	}
}

func TestDatabaseTarget_GetDefaultDatabase(t *testing.T) {
	tests := []struct {
		name     string
		engine   string
		expected string
	}{
		{"mysql engine", "mysql", ""},
		{"aurora-mysql engine", "aurora-mysql", ""},
		{"postgres engine", "postgres", "postgres"},
		{"aurora-postgresql engine", "aurora-postgresql", "postgres"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := models.DatabaseTarget{Engine: tt.engine}
			result := target.GetDefaultDatabase()
			if result != tt.expected {
				t.Errorf("GetDefaultDatabase() for engine %s = %v, want %v", tt.engine, result, tt.expected)
			}
		})
	}
}

func TestParseRegionFromArn(t *testing.T) {
	tests := []struct {
		name             string
		arn              string
		expectedRegion   string
		expectError      bool
		errorDescription string
	}{
		{
			name:           "valid ARN with suffix",
			arn:            "arn:aws:secretsmanager:us-east-1:123456789012:secret:rds-db-credentials-AbCdEf",
			expectedRegion: "us-east-1",
			expectError:    false,
		},
		{
			name:           "valid ARN us-west-2",
			arn:            "arn:aws:secretsmanager:us-west-2:987654321098:secret:prod-mysql-admin-XyZ123",
			expectedRegion: "us-west-2",
			expectError:    false,
		},
		{
			name:           "valid ARN with special chars in secret name",
			arn:            "arn:aws:secretsmanager:us-east-1:492684252576:secret:rds!db-4b2c881c-a322-48df-a0c9-7579823a541c-XjoANx",
			expectedRegion: "us-east-1",
			expectError:    false,
		},
		{
			name:           "valid ARN eu-west-1",
			arn:            "arn:aws:secretsmanager:eu-west-1:123456789012:secret:my-secret-123456",
			expectedRegion: "eu-west-1",
			expectError:    false,
		},
		{
			name:             "invalid ARN - too few parts",
			arn:              "arn:aws:secretsmanager",
			expectError:      true,
			errorDescription: "should fail with too few parts",
		},
		{
			name:             "invalid ARN - empty string",
			arn:              "",
			expectError:      true,
			errorDescription: "should fail with empty string",
		},
		{
			name:             "invalid ARN - malformed",
			arn:              "not-an-arn",
			expectError:      true,
			errorDescription: "should fail with malformed ARN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			region, err := parseRegionFromArn(tt.arn)

			if tt.expectError {
				if err == nil {
					t.Errorf("parseRegionFromArn() expected error (%s), but got none", tt.errorDescription)
				}
			} else {
				if err != nil {
					t.Errorf("parseRegionFromArn() unexpected error: %v", err)
					return
				}
				if region != tt.expectedRegion {
					t.Errorf("parseRegionFromArn() region = %v, want %v", region, tt.expectedRegion)
				}
			}
		})
	}
}

func TestSearchFilter(t *testing.T) {
	searchFilter := func(filter string, value string, index int) bool {
		return strings.Contains(strings.ToLower(value), strings.ToLower(filter))
	}

	tests := []struct {
		name     string
		filter   string
		value    string
		expected bool
	}{
		{"exact match", "prod", "prod-mysql", true},
		{"case insensitive", "PROD", "prod-mysql", true},
		{"substring match", "mysql", "prod-mysql-01", true},
		{"no match", "staging", "prod-mysql", false},
		{"empty filter matches all", "", "prod-mysql", true},
		{"partial match", "sql", "prod-mysql", true},
		{"case insensitive no match", "STAGING", "prod-mysql", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := searchFilter(tt.filter, tt.value, 0)
			if result != tt.expected {
				t.Errorf("searchFilter(%q, %q) = %v, want %v", tt.filter, tt.value, result, tt.expected)
			}
		})
	}
}
