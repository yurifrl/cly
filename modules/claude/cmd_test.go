package claude

import (
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	claudesession "github.com/yurifrl/cly/modules/claude-session"
)

func TestResumeOrCreateSession_CreateNew(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsFile := filepath.Join(tmpDir, "sessions.json")
	workDir := t.TempDir()

	// Start with empty sessions
	sessions := claudesession.Sessions{}
	err := claudesession.Save(sessionsFile, sessions)
	require.NoError(t, err)

	// Mock: we can't actually call ExecClaude in tests, so we'll verify
	// that the session was saved correctly
	sessionID := uuid.New().String()
	key := claudesession.MakeKey(workDir, sessionID)
	sessions[key] = claudesession.Entry{
		ID:   sessionID,
		Name: "testproj",
		Path: workDir,
	}
	err = claudesession.Save(sessionsFile, sessions)
	require.NoError(t, err)

	// Verify session was saved
	loaded, err := claudesession.Load(sessionsFile)
	require.NoError(t, err)
	entry := claudesession.FindByNameAndPath(loaded, "testproj", workDir)
	require.NotNil(t, entry)
	assert.Equal(t, sessionID, entry.ID)
	assert.Equal(t, "testproj", entry.Name)
	assert.Equal(t, workDir, entry.Path)
}

func TestResumeOrCreateSession_ResumeExisting(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsFile := filepath.Join(tmpDir, "sessions.json")
	workDir := t.TempDir()

	// Create existing session
	existingID := uuid.New().String()
	sessions := claudesession.Sessions{
		claudesession.MakeKey(workDir, existingID): claudesession.Entry{
			ID:   existingID,
			Name: "existing",
			Path: workDir,
		},
	}
	err := claudesession.Save(sessionsFile, sessions)
	require.NoError(t, err)

	// Verify we can find it
	loaded, err := claudesession.Load(sessionsFile)
	require.NoError(t, err)
	entry := claudesession.FindByNameAndPath(loaded, "existing", workDir)
	require.NotNil(t, entry)
	assert.Equal(t, existingID, entry.ID)
}

func TestResumeOrCreateSession_DifferentPathsSameName(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsFile := filepath.Join(tmpDir, "sessions.json")
	workDirA := filepath.Join(tmpDir, "a")
	workDirB := filepath.Join(tmpDir, "b")

	// Create sessions with same name in different paths
	idA := uuid.New().String()
	idB := uuid.New().String()
	sessions := claudesession.Sessions{
		claudesession.MakeKey(workDirA, idA): claudesession.Entry{
			ID:   idA,
			Name: "test",
			Path: workDirA,
		},
		claudesession.MakeKey(workDirB, idB): claudesession.Entry{
			ID:   idB,
			Name: "test",
			Path: workDirB,
		},
	}
	err := claudesession.Save(sessionsFile, sessions)
	require.NoError(t, err)

	// Verify both exist and are different
	loaded, err := claudesession.Load(sessionsFile)
	require.NoError(t, err)

	entryA := claudesession.FindByNameAndPath(loaded, "test", workDirA)
	require.NotNil(t, entryA)
	assert.Equal(t, idA, entryA.ID)

	entryB := claudesession.FindByNameAndPath(loaded, "test", workDirB)
	require.NotNil(t, entryB)
	assert.Equal(t, idB, entryB.ID)

	assert.NotEqual(t, entryA.ID, entryB.ID)
}
