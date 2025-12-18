package backup

import (
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildExcludePattern(t *testing.T) {
	pattern := buildExcludePattern()

	assert.Contains(t, pattern, ".*node_modules/.*")
	assert.Contains(t, pattern, ".*__pycache__/.*")
	assert.Contains(t, pattern, ".*\\.pyc$")
	assert.Contains(t, pattern, ".*\\.DS_Store$")
	assert.Contains(t, pattern, ".*\\.terraform/.*")
	assert.Contains(t, pattern, ".*\\.vscode/.*")
	assert.Contains(t, pattern, ".*venv/.*")
	assert.Contains(t, pattern, ".*\\.git/objects/.*")

	parts := strings.Split(pattern, "|")
	assert.Greater(t, len(parts), 40, "Should have many exclusion patterns")
}

func TestCalculateParallelProcesses(t *testing.T) {
	processes := calculateParallelProcesses()

	assert.GreaterOrEqual(t, processes, 4, "Should have at least 4 parallel processes")
	assert.LessOrEqual(t, processes, 32, "Should not exceed 32 parallel processes")

	numCores := runtime.NumCPU()
	expected := numCores * 2
	if expected < 4 {
		expected = 4
	}
	if expected > 32 {
		expected = 32
	}

	assert.Equal(t, expected, processes)
}

func TestGetBucket(t *testing.T) {
	bucket := getBucket()
	assert.IsType(t, "", bucket)
}
