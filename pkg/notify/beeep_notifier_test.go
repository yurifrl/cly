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
