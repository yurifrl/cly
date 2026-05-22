package session

import (
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateName(t *testing.T) {
	name := GenerateName()

	assert.NotEmpty(t, name)
	assert.Regexp(t, regexp.MustCompile(`^[A-Z][a-z]+[A-Z][a-z]+$`), name)
}

func TestGenerateName_Uniqueness(t *testing.T) {
	names := make(map[string]bool)
	for i := 0; i < 100; i++ {
		name := GenerateName()
		names[name] = true
	}
	// With 50+ adjectives and 50+ animals, we should get variety
	assert.Greater(t, len(names), 50)
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid alphanumeric", "WorkProject", false},
		{"valid with hyphen", "work-project", false},
		{"valid with underscore", "work_project", false},
		{"valid mixed", "Work-Project_123", false},
		{"invalid space", "Work Project", true},
		{"invalid special char", "Work@Project", true},
		{"invalid slash", "Work/Project", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestInitialize_ExplicitName(t *testing.T) {
	sess, err := Initialize("MyProject")

	require.NoError(t, err)
	assert.Equal(t, "MyProject", sess.Name)
}

func TestInitialize_EnvVar(t *testing.T) {
	os.Unsetenv("CLY_SESSION_NAME")
	os.Setenv("CLAUDE_SESSION_NAME", "EnvProject")
	defer os.Unsetenv("CLAUDE_SESSION_NAME")

	sess, err := Initialize("")

	require.NoError(t, err)
	assert.Equal(t, "EnvProject", sess.Name)
}

func TestInitialize_CLYEnvVar(t *testing.T) {
	os.Setenv("CLY_SESSION_NAME", "CLYProject")
	os.Setenv("CLAUDE_SESSION_NAME", "LegacyProject")
	defer os.Unsetenv("CLY_SESSION_NAME")
	defer os.Unsetenv("CLAUDE_SESSION_NAME")

	sess, err := Initialize("")

	require.NoError(t, err)
	assert.Equal(t, "CLYProject", sess.Name)
}

func TestInitialize_AutoGenerate(t *testing.T) {
	os.Unsetenv("CLY_SESSION_NAME")
	os.Unsetenv("CLAUDE_SESSION_NAME")

	sess, err := Initialize("")

	require.NoError(t, err)
	assert.NotEmpty(t, sess.Name)
	assert.Regexp(t, regexp.MustCompile(`^[A-Z][a-z]+[A-Z][a-z]+$`), sess.Name)
}

func TestInitialize_InvalidName(t *testing.T) {
	_, err := Initialize("Invalid Name")

	assert.Error(t, err)
}

func TestIsInZellij(t *testing.T) {
	os.Unsetenv("ZELLIJ")
	assert.False(t, IsInZellij())

	os.Setenv("ZELLIJ", "0")
	defer os.Unsetenv("ZELLIJ")
	assert.True(t, IsInZellij())
}

func TestBuildAnonymousArgs(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "empty args",
			input:    []string{},
			expected: []string{"--setting-sources", "user"},
		},
		{
			name:     "with existing args",
			input:    []string{"-p", "hello"},
			expected: []string{"-p", "hello", "--setting-sources", "user"},
		},
		{
			name:     "with resume flag",
			input:    []string{"--resume"},
			expected: []string{"--resume", "--setting-sources", "user"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildAnonymousArgs(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
