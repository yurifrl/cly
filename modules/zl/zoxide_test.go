package zl

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckZoxideInstalled(t *testing.T) {
	t.Run("returns true when zoxide in PATH", func(t *testing.T) {
		// Skip if zoxide not actually installed
		if _, err := exec.LookPath("zoxide"); err != nil {
			t.Skip("zoxide not installed")
		}
		assert.True(t, CheckZoxideInstalled())
	})

	t.Run("returns false when zoxide not in PATH", func(t *testing.T) {
		// This test assumes zoxide is NOT called "nonexistent-binary"
		oldPath := findZoxidePath
		findZoxidePath = func(file string) (string, error) {
			return "", exec.ErrNotFound
		}
		defer func() { findZoxidePath = oldPath }()

		assert.False(t, CheckZoxideInstalled())
	})
}

func TestQueryZoxide(t *testing.T) {
	t.Run("returns empty when not installed", func(t *testing.T) {
		oldInstalled := checkZoxideInstalledFunc
		checkZoxideInstalledFunc = func() bool { return false }
		defer func() { checkZoxideInstalledFunc = oldInstalled }()

		path, err := QueryZoxide("work")
		require.NoError(t, err)
		assert.Empty(t, path)
	})

	t.Run("returns path on successful query", func(t *testing.T) {
		oldInstalled := checkZoxideInstalledFunc
		oldExec := execZoxideQuery
		checkZoxideInstalledFunc = func() bool { return true }
		execZoxideQuery = func(keywords ...string) (string, error) {
			return "/home/user/work\n", nil
		}
		defer func() {
			checkZoxideInstalledFunc = oldInstalled
			execZoxideQuery = oldExec
		}()

		path, err := QueryZoxide("work")
		require.NoError(t, err)
		assert.Equal(t, "/home/user/work", path)
	})

	t.Run("returns empty on no match", func(t *testing.T) {
		oldInstalled := checkZoxideInstalledFunc
		oldExec := execZoxideQuery
		checkZoxideInstalledFunc = func() bool { return true }
		execZoxideQuery = func(keywords ...string) (string, error) {
			return "", nil
		}
		defer func() {
			checkZoxideInstalledFunc = oldInstalled
			execZoxideQuery = oldExec
		}()

		path, err := QueryZoxide("nonexistent")
		require.NoError(t, err)
		assert.Empty(t, path)
	})
}

func TestQueryZoxideInteractive(t *testing.T) {
	t.Run("returns empty when not installed", func(t *testing.T) {
		oldInstalled := checkZoxideInstalledFunc
		checkZoxideInstalledFunc = func() bool { return false }
		defer func() { checkZoxideInstalledFunc = oldInstalled }()

		path, err := QueryZoxideInteractive()
		require.NoError(t, err)
		assert.Empty(t, path)
	})

	t.Run("returns path on successful selection", func(t *testing.T) {
		oldInstalled := checkZoxideInstalledFunc
		oldExec := execZoxideInteractive
		checkZoxideInstalledFunc = func() bool { return true }
		execZoxideInteractive = func() (string, error) {
			return "/home/user/project\n", nil
		}
		defer func() {
			checkZoxideInstalledFunc = oldInstalled
			execZoxideInteractive = oldExec
		}()

		path, err := QueryZoxideInteractive()
		require.NoError(t, err)
		assert.Equal(t, "/home/user/project", path)
	})

	t.Run("returns empty on cancellation", func(t *testing.T) {
		oldInstalled := checkZoxideInstalledFunc
		oldExec := execZoxideInteractive
		checkZoxideInstalledFunc = func() bool { return true }
		execZoxideInteractive = func() (string, error) {
			return "", nil
		}
		defer func() {
			checkZoxideInstalledFunc = oldInstalled
			execZoxideInteractive = oldExec
		}()

		path, err := QueryZoxideInteractive()
		require.NoError(t, err)
		assert.Empty(t, path)
	})
}

func TestUpdateZoxide(t *testing.T) {
	t.Run("does nothing when not installed", func(t *testing.T) {
		oldInstalled := checkZoxideInstalledFunc
		checkZoxideInstalledFunc = func() bool { return false }
		defer func() { checkZoxideInstalledFunc = oldInstalled }()

		err := UpdateZoxide("/home/user/work")
		assert.NoError(t, err)
	})

	t.Run("calls zoxide add", func(t *testing.T) {
		oldInstalled := checkZoxideInstalledFunc
		oldExec := execZoxideAdd
		called := false
		checkZoxideInstalledFunc = func() bool { return true }
		execZoxideAdd = func(dir string) error {
			called = true
			assert.Equal(t, "/home/user/work", dir)
			return nil
		}
		defer func() {
			checkZoxideInstalledFunc = oldInstalled
			execZoxideAdd = oldExec
		}()

		err := UpdateZoxide("/home/user/work")
		require.NoError(t, err)
		assert.True(t, called)
	})
}
