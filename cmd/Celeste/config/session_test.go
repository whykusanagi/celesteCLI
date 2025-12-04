package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewSessionManager tests session manager creation
func TestNewSessionManager(t *testing.T) {
	tmpDir := t.TempDir()
	oldHomeDir := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHomeDir)
	os.Setenv("HOME", tmpDir)

	manager := NewSessionManager()
	require.NotNil(t, manager)
	assert.Contains(t, manager.sessionsDir, ".celeste/sessions")

	// Verify sessions directory was created
	_, err := os.Stat(manager.sessionsDir)
	assert.NoError(t, err)
}

// TestNewSession tests new session creation
func TestNewSession(t *testing.T) {
	tmpDir := t.TempDir()
	oldHomeDir := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHomeDir)
	os.Setenv("HOME", tmpDir)

	manager := NewSessionManager()
	session := manager.NewSession()

	require.NotNil(t, session)
	assert.NotEmpty(t, session.ID)
	assert.NotZero(t, session.CreatedAt)
	assert.NotZero(t, session.UpdatedAt)
	assert.Empty(t, session.Messages)
	assert.NotNil(t, session.Metadata)
	assert.Equal(t, session.ID, manager.GetCurrentID())
}

// TestSaveAndLoadSession tests session save/load roundtrip
func TestSaveAndLoadSession(t *testing.T) {
	tmpDir := t.TempDir()
	oldHomeDir := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHomeDir)
	os.Setenv("HOME", tmpDir)

	manager := NewSessionManager()
	session := manager.NewSession()

	// Add some messages
	session.Messages = []SessionMessage{
		{
			Role:      "user",
			Content:   "Hello",
			Timestamp: time.Now(),
		},
		{
			Role:      "assistant",
			Content:   "Hi there!",
			Timestamp: time.Now(),
		},
	}
	session.NSFWMode = true
	session.Name = "Test Session"

	// Save session
	err := manager.Save(session)
	require.NoError(t, err)

	// Verify file exists
	sessionPath := filepath.Join(manager.sessionsDir, session.ID+".json")
	_, err = os.Stat(sessionPath)
	assert.NoError(t, err)

	// Load session
	loaded, err := manager.Load(session.ID)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	// Verify values
	assert.Equal(t, session.ID, loaded.ID)
	assert.Equal(t, session.Name, loaded.Name)
	assert.Equal(t, session.NSFWMode, loaded.NSFWMode)
	assert.Len(t, loaded.Messages, 2)
	assert.Equal(t, "user", loaded.Messages[0].Role)
	assert.Equal(t, "Hello", loaded.Messages[0].Content)
	assert.Equal(t, "assistant", loaded.Messages[1].Role)
	assert.Equal(t, "Hi there!", loaded.Messages[1].Content)
}

// TestAddMessage tests adding messages to session
func TestAddMessage(t *testing.T) {
	tmpDir := t.TempDir()
	oldHomeDir := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHomeDir)
	os.Setenv("HOME", tmpDir)

	manager := NewSessionManager()
	session := manager.NewSession()

	// Initially empty
	assert.Empty(t, session.Messages)

	// Add first message
	manager.AddMessage(session, "user", "First message")
	assert.Len(t, session.Messages, 1)
	assert.Equal(t, "user", session.Messages[0].Role)
	assert.Equal(t, "First message", session.Messages[0].Content)
	assert.NotZero(t, session.Messages[0].Timestamp)

	// Add second message
	manager.AddMessage(session, "assistant", "Second message")
	assert.Len(t, session.Messages, 2)
	assert.Equal(t, "assistant", session.Messages[1].Role)
	assert.Equal(t, "Second message", session.Messages[1].Content)

	// Verify UpdatedAt was updated
	assert.NotZero(t, session.UpdatedAt)
}

// TestListSessions tests listing all sessions
func TestListSessions(t *testing.T) {
	tmpDir := t.TempDir()
	oldHomeDir := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHomeDir)
	os.Setenv("HOME", tmpDir)

	manager := NewSessionManager()

	// Empty list initially
	sessions, err := manager.List()
	require.NoError(t, err)
	assert.Empty(t, sessions)

	// Create and save multiple sessions
	session1 := manager.NewSession()
	session1.Name = "Session 1"
	err = manager.Save(session1)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond) // Ensure different IDs

	session2 := manager.NewSession()
	session2.Name = "Session 2"
	err = manager.Save(session2)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	session3 := manager.NewSession()
	session3.Name = "Session 3"
	err = manager.Save(session3)
	require.NoError(t, err)

	// List all sessions
	sessions, err = manager.List()
	require.NoError(t, err)
	assert.Len(t, sessions, 3)

	// Verify session names
	names := make(map[string]bool)
	for _, s := range sessions {
		names[s.Name] = true
	}
	assert.True(t, names["Session 1"])
	assert.True(t, names["Session 2"])
	assert.True(t, names["Session 3"])
}

// TestLoadLatest tests loading the most recent session
func TestLoadLatest(t *testing.T) {
	tmpDir := t.TempDir()
	oldHomeDir := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHomeDir)
	os.Setenv("HOME", tmpDir)

	manager := NewSessionManager()

	// No sessions initially
	_, err := manager.LoadLatest()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no sessions found")

	// Create sessions with different update times
	session1 := manager.NewSession()
	session1.Name = "Oldest"
	session1.UpdatedAt = time.Now().Add(-2 * time.Hour)
	err = manager.Save(session1)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	session2 := manager.NewSession()
	session2.Name = "Middle"
	session2.UpdatedAt = time.Now().Add(-1 * time.Hour)
	err = manager.Save(session2)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	session3 := manager.NewSession()
	session3.Name = "Latest"
	session3.UpdatedAt = time.Now()
	err = manager.Save(session3)
	require.NoError(t, err)

	// Load latest
	latest, err := manager.LoadLatest()
	require.NoError(t, err)
	assert.Equal(t, "Latest", latest.Name)
	assert.Equal(t, session3.ID, latest.ID)
}

// TestDeleteSession tests deleting a session
func TestDeleteSession(t *testing.T) {
	tmpDir := t.TempDir()
	oldHomeDir := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHomeDir)
	os.Setenv("HOME", tmpDir)

	manager := NewSessionManager()

	// Create and save session
	session := manager.NewSession()
	session.Name = "To Delete"
	err := manager.Save(session)
	require.NoError(t, err)

	// Verify it exists
	sessions, err := manager.List()
	require.NoError(t, err)
	assert.Len(t, sessions, 1)

	// Delete session
	err = manager.Delete(session.ID)
	require.NoError(t, err)

	// Verify it's gone
	sessions, err = manager.List()
	require.NoError(t, err)
	assert.Empty(t, sessions)

	// Try to load deleted session
	_, err = manager.Load(session.ID)
	assert.Error(t, err)
}

// TestClearSessions tests clearing all sessions
func TestClearSessions(t *testing.T) {
	tmpDir := t.TempDir()
	oldHomeDir := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHomeDir)
	os.Setenv("HOME", tmpDir)

	manager := NewSessionManager()

	// Create multiple sessions
	for i := 0; i < 5; i++ {
		session := manager.NewSession()
		err := manager.Save(session)
		require.NoError(t, err)
		time.Sleep(5 * time.Millisecond)
	}

	// Verify they exist
	sessions, err := manager.List()
	require.NoError(t, err)
	assert.Len(t, sessions, 5)

	// Clear all
	err = manager.Clear()
	require.NoError(t, err)

	// Verify all gone
	sessions, err = manager.List()
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

// TestGetMessagesForLLM tests message conversion for LLM
func TestGetMessagesForLLM(t *testing.T) {
	session := &Session{
		ID:        "test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Messages: []SessionMessage{
			{
				Role:      "user",
				Content:   "Hello",
				Timestamp: time.Now(),
			},
			{
				Role:      "assistant",
				Content:   "Hi!",
				Timestamp: time.Now(),
			},
			{
				Role:      "user",
				Content:   "How are you?",
				Timestamp: time.Now(),
			},
		},
	}

	messages := GetMessagesForLLM(session)
	require.Len(t, messages, 3)

	assert.Equal(t, "user", messages[0]["role"])
	assert.Equal(t, "Hello", messages[0]["content"])

	assert.Equal(t, "assistant", messages[1]["role"])
	assert.Equal(t, "Hi!", messages[1]["content"])

	assert.Equal(t, "user", messages[2]["role"])
	assert.Equal(t, "How are you?", messages[2]["content"])

	// Verify timestamps are not included
	assert.NotContains(t, messages[0], "timestamp")
}

// TestSessionSummarize tests session summary generation
func TestSessionSummarize(t *testing.T) {
	now := time.Now()
	session := &Session{
		ID:        "test-123",
		Name:      "Test Session",
		CreatedAt: now,
		UpdatedAt: now,
		Messages: []SessionMessage{
			{
				Role:      "user",
				Content:   "This is the first user message that should appear in the preview",
				Timestamp: now,
			},
			{
				Role:      "assistant",
				Content:   "Assistant response",
				Timestamp: now,
			},
			{
				Role:      "user",
				Content:   "Another user message",
				Timestamp: now,
			},
		},
	}

	summary := session.Summarize()

	assert.Equal(t, "test-123", summary.ID)
	assert.Equal(t, "Test Session", summary.Name)
	assert.Equal(t, 3, summary.MessageCount)
	assert.Equal(t, now, summary.CreatedAt)
	assert.Equal(t, now, summary.UpdatedAt)

	// First message should be truncated at 50 chars
	assert.Contains(t, summary.FirstMessage, "This is the first user message")
	assert.Len(t, summary.FirstMessage, 50) // 47 chars + "..."
	assert.True(t, len(summary.FirstMessage) > 0 && summary.FirstMessage[len(summary.FirstMessage)-3:] == "...")
}

// TestSessionSummarizeNoMessages tests summary with no messages
func TestSessionSummarizeNoMessages(t *testing.T) {
	now := time.Now()
	session := &Session{
		ID:        "empty-123",
		Name:      "Empty Session",
		CreatedAt: now,
		UpdatedAt: now,
		Messages:  []SessionMessage{},
	}

	summary := session.Summarize()

	assert.Equal(t, "empty-123", summary.ID)
	assert.Equal(t, "Empty Session", summary.Name)
	assert.Equal(t, 0, summary.MessageCount)
	assert.Empty(t, summary.FirstMessage)
}

// TestSessionSummarizeShortMessage tests summary with short first message
func TestSessionSummarizeShortMessage(t *testing.T) {
	now := time.Now()
	session := &Session{
		ID:        "short-123",
		CreatedAt: now,
		UpdatedAt: now,
		Messages: []SessionMessage{
			{
				Role:      "user",
				Content:   "Short",
				Timestamp: now,
			},
		},
	}

	summary := session.Summarize()
	assert.Equal(t, "Short", summary.FirstMessage)
	assert.NotContains(t, summary.FirstMessage, "...")
}

// TestSessionUpdateTime tests that save updates UpdatedAt
func TestSessionUpdateTime(t *testing.T) {
	tmpDir := t.TempDir()
	oldHomeDir := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHomeDir)
	os.Setenv("HOME", tmpDir)

	manager := NewSessionManager()
	session := manager.NewSession()

	originalUpdatedAt := session.UpdatedAt
	time.Sleep(50 * time.Millisecond)

	// Save session
	err := manager.Save(session)
	require.NoError(t, err)

	// UpdatedAt should be newer
	assert.True(t, session.UpdatedAt.After(originalUpdatedAt))
}

// TestSessionMetadata tests session metadata handling
func TestSessionMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	oldHomeDir := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHomeDir)
	os.Setenv("HOME", tmpDir)

	manager := NewSessionManager()
	session := manager.NewSession()

	// Add metadata
	session.Metadata["model"] = "gpt-4"
	session.Metadata["temperature"] = 0.7
	session.Metadata["custom_field"] = "test"

	// Save and reload
	err := manager.Save(session)
	require.NoError(t, err)

	loaded, err := manager.Load(session.ID)
	require.NoError(t, err)

	// Verify metadata persisted
	assert.Equal(t, "gpt-4", loaded.Metadata["model"])
	assert.Equal(t, 0.7, loaded.Metadata["temperature"])
	assert.Equal(t, "test", loaded.Metadata["custom_field"])
}
