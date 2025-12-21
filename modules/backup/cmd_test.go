package backup

import (
	"runtime"
	"strings"
	"sync"
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

func TestCategorizeGsutilLine(t *testing.T) {
	// Save original flag value and restore after test
	originalFlag := showSkippedFlag
	defer func() { showSkippedFlag = originalFlag }()

	tests := []struct {
		name        string
		line        string
		showSkipped bool
		wantOp      operationType
		wantPrint   bool
		wantCount   bool
	}{
		{
			name:        "upload copying line",
			line:        "Copying file:///home/work/file.txt to gs://bucket/file.txt",
			showSkipped: false,
			wantOp:      opUpload,
			wantPrint:   true,
			wantCount:   true,
		},
		{
			name:        "skipped symbolic link - flag false",
			line:        "Skipping symbolic link /home/work/link",
			showSkipped: false,
			wantOp:      opSkipped,
			wantPrint:   false,
			wantCount:   true,
		},
		{
			name:        "skipped symbolic link - flag true",
			line:        "Skipping symbolic link /home/work/link",
			showSkipped: true,
			wantOp:      opSkipped,
			wantPrint:   true,
			wantCount:   true,
		},
		{
			name:        "progress indicator",
			line:        "At source listing 10000...",
			showSkipped: false,
			wantOp:      opProgress,
			wantPrint:   true,
			wantCount:   false,
		},
		{
			name:        "error line",
			line:        "CommandException: 1 files/objects could not be copied/removed.",
			showSkipped: false,
			wantOp:      opError,
			wantPrint:   true,
			wantCount:   true,
		},
		{
			name:        "building sync state",
			line:        "Building synchronization state...",
			showSkipped: false,
			wantOp:      opInfo,
			wantPrint:   true,
			wantCount:   false,
		},
		{
			name:        "warning line",
			line:        "WARNING: You have requested checksumming...",
			showSkipped: false,
			wantOp:      opInfo,
			wantPrint:   true,
			wantCount:   false,
		},
		{
			name:        "empty line",
			line:        "",
			showSkipped: false,
			wantOp:      opInfo,
			wantPrint:   false,
			wantCount:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			showSkippedFlag = tt.showSkipped
			gotOp, gotPrint, gotCount := categorizeGsutilLine(tt.line)
			assert.Equal(t, tt.wantOp, gotOp, "operation type mismatch")
			assert.Equal(t, tt.wantPrint, gotPrint, "shouldPrint mismatch")
			assert.Equal(t, tt.wantCount, gotCount, "shouldCount mismatch")
		})
	}
}

func TestSyncStatsIncrement(t *testing.T) {
	stats := &syncStats{}

	stats.increment(opUpload)
	stats.increment(opUpload)
	stats.increment(opSkipped)
	stats.increment(opError)

	assert.Equal(t, 2, stats.uploaded)
	assert.Equal(t, 1, stats.skipped)
	assert.Equal(t, 1, stats.failed)
}

func TestSyncStatsConcurrency(t *testing.T) {
	stats := &syncStats{}
	var wg sync.WaitGroup

	// Simulate concurrent updates from multiple goroutines
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			stats.increment(opUpload)
		}()
	}

	wg.Wait()
	assert.Equal(t, 100, stats.uploaded)
}
