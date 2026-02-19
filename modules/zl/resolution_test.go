package zl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveDirectory(t *testing.T) {
	t.Run("returns explicit mapping first", func(t *testing.T) {
		oldLoad := loadZlConfigFunc
		loadZlConfigFunc = func() ZlConfig {
			return ZlConfig{
				AutoZoxide: true,
				SessionDirs: map[string]string{
					"work": "/home/user/explicit-work",
				},
			}
		}
		defer func() { loadZlConfigFunc = oldLoad }()

		oldQuery := queryZoxideFunc
		queryZoxideFunc = func(keywords ...string) (string, error) {
			return "/home/user/zoxide-work", nil
		}
		defer func() { queryZoxideFunc = oldQuery }()

		dir := ResolveDirectory("work")
		assert.Equal(t, "/home/user/explicit-work", dir)
	})

	t.Run("falls back to zoxide when no explicit mapping", func(t *testing.T) {
		oldLoad := loadZlConfigFunc
		loadZlConfigFunc = func() ZlConfig {
			return ZlConfig{
				AutoZoxide:  true,
				SessionDirs: map[string]string{},
			}
		}
		defer func() { loadZlConfigFunc = oldLoad }()

		oldQuery := queryZoxideFunc
		queryZoxideFunc = func(keywords ...string) (string, error) {
			assert.Equal(t, []string{"work"}, keywords)
			return "/home/user/zoxide-work", nil
		}
		defer func() { queryZoxideFunc = oldQuery }()

		dir := ResolveDirectory("work")
		assert.Equal(t, "/home/user/zoxide-work", dir)
	})

	t.Run("returns empty when auto_zoxide disabled", func(t *testing.T) {
		oldLoad := loadZlConfigFunc
		loadZlConfigFunc = func() ZlConfig {
			return ZlConfig{
				AutoZoxide:  false,
				SessionDirs: map[string]string{},
			}
		}
		defer func() { loadZlConfigFunc = oldLoad }()

		dir := ResolveDirectory("work")
		assert.Empty(t, dir)
	})

	t.Run("returns empty when zoxide returns empty", func(t *testing.T) {
		oldLoad := loadZlConfigFunc
		loadZlConfigFunc = func() ZlConfig {
			return ZlConfig{
				AutoZoxide:  true,
				SessionDirs: map[string]string{},
			}
		}
		defer func() { loadZlConfigFunc = oldLoad }()

		oldQuery := queryZoxideFunc
		queryZoxideFunc = func(keywords ...string) (string, error) {
			return "", nil
		}
		defer func() { queryZoxideFunc = oldQuery }()

		dir := ResolveDirectory("nonexistent")
		assert.Empty(t, dir)
	})

	t.Run("handles zoxide errors gracefully", func(t *testing.T) {
		oldLoad := loadZlConfigFunc
		loadZlConfigFunc = func() ZlConfig {
			return ZlConfig{
				AutoZoxide:  true,
				SessionDirs: map[string]string{},
			}
		}
		defer func() { loadZlConfigFunc = oldLoad }()

		oldQuery := queryZoxideFunc
		queryZoxideFunc = func(keywords ...string) (string, error) {
			return "", assert.AnError
		}
		defer func() { queryZoxideFunc = oldQuery }()

		dir := ResolveDirectory("work")
		assert.Empty(t, dir)
	})
}
