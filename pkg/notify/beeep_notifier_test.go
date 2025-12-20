package notify

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBeeepNotifier_Available(t *testing.T) {
	bn := &BeeepNotifier{}
	// beeep is always available (handles platform detection internally)
	assert.True(t, bn.Available())
}

func TestBeeepNotifier_CombinesTitleAndSubtitle(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		subtitle string
		expected string
	}{
		{
			name:     "with subtitle",
			title:    "Main Title",
			subtitle: "Subtitle",
			expected: "Main Title - Subtitle",
		},
		{
			name:     "without subtitle",
			title:    "Main Title",
			subtitle: "",
			expected: "Main Title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bn := &BeeepNotifier{}
			combined := bn.combineTitle(tt.title, tt.subtitle)
			assert.Equal(t, tt.expected, combined)
		})
	}
}
