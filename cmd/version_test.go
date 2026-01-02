package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateBuildName(t *testing.T) {
	t.Run("same timestamp produces same name", func(t *testing.T) {
		name1 := GenerateBuildName("20260102150405")
		name2 := GenerateBuildName("20260102150405")
		assert.Equal(t, name1, name2)
	})

	t.Run("different timestamps produce different names", func(t *testing.T) {
		name1 := GenerateBuildName("20260102150405")
		name2 := GenerateBuildName("20260102150406")
		assert.NotEqual(t, name1, name2)
	})

	t.Run("format is adjective_noun", func(t *testing.T) {
		name := GenerateBuildName("20260102150405")
		assert.Contains(t, name, "_")
	})

	t.Run("handles unknown/empty build time", func(t *testing.T) {
		name := GenerateBuildName("unknown")
		assert.NotEmpty(t, name)
		assert.Contains(t, name, "_")
	})
}
