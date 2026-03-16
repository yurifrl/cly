package claude

import (
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	agentsession "github.com/yurifrl/cly/modules/agent-session"
)

func TestRunRename_ValidName(t *testing.T) {
	err := runRename(nil, []string{"my-project"})
	assert.NoError(t, err)
}

func TestRunRename_InvalidName(t *testing.T) {
	err := runRename(nil, []string{"invalid name!"})
	assert.Error(t, err)
}

func TestResumeOrCreateSession_CreateNew(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsFile := filepath.Join(tmpDir, "sessions.json")
	workDir := t.TempDir()

	sessions := agentsession.Sessions{}
	err := agentsession.Save(sessionsFile, sessions)
	require.NoError(t, err)

	sessionID := uuid.New().String()
	sessions["testproj"] = agentsession.Entry{
		ID:   sessionID,
		Name: "testproj",
		Path: workDir,
	}
	err = agentsession.Save(sessionsFile, sessions)
	require.NoError(t, err)

	loaded, err := agentsession.Load(sessionsFile)
	require.NoError(t, err)
	entry := agentsession.FindByName(loaded, "testproj")
	require.NotNil(t, entry)
	assert.Equal(t, sessionID, entry.ID)
	assert.Equal(t, workDir, entry.Path)
}

func TestResumeOrCreateSession_ResumeExisting(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsFile := filepath.Join(tmpDir, "sessions.json")
	workDir := t.TempDir()

	existingID := uuid.New().String()
	sessions := agentsession.Sessions{
		"existing": agentsession.Entry{
			ID:   existingID,
			Name: "existing",
			Path: workDir,
		},
	}
	err := agentsession.Save(sessionsFile, sessions)
	require.NoError(t, err)

	loaded, err := agentsession.Load(sessionsFile)
	require.NoError(t, err)
	entry := agentsession.FindByName(loaded, "existing")
	require.NotNil(t, entry)
	assert.Equal(t, existingID, entry.ID)
}
