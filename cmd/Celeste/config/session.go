// Package config provides configuration management for Celeste CLI.
// This file handles session persistence (conversation history).
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Session represents a saved conversation session.
type Session struct {
	ID        string           `json:"id"`
	Name      string           `json:"name,omitempty"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
	Messages  []SessionMessage `json:"messages"`
	NSFWMode  bool             `json:"nsfw_mode,omitempty"`
	Metadata  map[string]any   `json:"metadata,omitempty"`
}

// SessionMessage represents a message in a session.
type SessionMessage struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// SessionManager manages session persistence.
type SessionManager struct {
	sessionsDir string
	currentID   string
}

// NewSessionManager creates a new session manager.
func NewSessionManager() *SessionManager {
	homeDir, _ := os.UserHomeDir()
	sessionsDir := filepath.Join(homeDir, ".celeste", "sessions")
	os.MkdirAll(sessionsDir, 0755)

	return &SessionManager{
		sessionsDir: sessionsDir,
	}
}

// NewSession creates a new session with a unique ID.
func (m *SessionManager) NewSession() *Session {
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	m.currentID = id

	return &Session{
		ID:        id,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Messages:  []SessionMessage{},
		Metadata:  make(map[string]any),
	}
}

// Save saves a session to disk.
func (m *SessionManager) Save(session *Session) error {
	session.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	path := filepath.Join(m.sessionsDir, session.ID+".json")
	return os.WriteFile(path, data, 0644)
}

// Load loads a session by ID.
func (m *SessionManager) Load(id string) (*Session, error) {
	path := filepath.Join(m.sessionsDir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read session: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to parse session: %w", err)
	}

	m.currentID = id
	return &session, nil
}

// LoadLatest loads the most recent session.
func (m *SessionManager) LoadLatest() (*Session, error) {
	sessions, err := m.List()
	if err != nil {
		return nil, err
	}

	if len(sessions) == 0 {
		return nil, fmt.Errorf("no sessions found")
	}

	// Sort by updated time (newest first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	return m.Load(sessions[0].ID)
}

// List returns all saved sessions.
func (m *SessionManager) List() ([]Session, error) {
	files, err := filepath.Glob(filepath.Join(m.sessionsDir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	var sessions []Session
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var session Session
		if err := json.Unmarshal(data, &session); err != nil {
			continue
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// Delete deletes a session by ID.
func (m *SessionManager) Delete(id string) error {
	path := filepath.Join(m.sessionsDir, id+".json")
	return os.Remove(path)
}

// Clear deletes all sessions.
func (m *SessionManager) Clear() error {
	files, err := filepath.Glob(filepath.Join(m.sessionsDir, "*.json"))
	if err != nil {
		return err
	}

	for _, file := range files {
		os.Remove(file)
	}

	return nil
}

// GetCurrentID returns the current session ID.
func (m *SessionManager) GetCurrentID() string {
	return m.currentID
}

// AddMessage adds a message to the session and saves.
func (m *SessionManager) AddMessage(session *Session, role, content string) {
	session.Messages = append(session.Messages, SessionMessage{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
	session.UpdatedAt = time.Now()
}

// SetEndpoint stores the current endpoint in session metadata.
func (s *Session) SetEndpoint(endpoint string) {
	if s.Metadata == nil {
		s.Metadata = make(map[string]any)
	}
	s.Metadata["endpoint"] = endpoint
}

// GetEndpoint retrieves the endpoint from session metadata.
func (s *Session) GetEndpoint() string {
	if s.Metadata == nil {
		return ""
	}
	if endpoint, ok := s.Metadata["endpoint"].(string); ok {
		return endpoint
	}
	return ""
}

// SetModel stores the current model in session metadata.
func (s *Session) SetModel(model string) {
	if s.Metadata == nil {
		s.Metadata = make(map[string]any)
	}
	s.Metadata["model"] = model
}

// GetModel retrieves the model from session metadata.
func (s *Session) GetModel() string {
	if s.Metadata == nil {
		return ""
	}
	if model, ok := s.Metadata["model"].(string); ok {
		return model
	}
	return ""
}

// SetNSFWMode stores the NSFW mode in session.
func (s *Session) SetNSFWMode(enabled bool) {
	s.NSFWMode = enabled
}

// GetNSFWMode retrieves the NSFW mode from session.
func (s *Session) GetNSFWMode() bool {
	return s.NSFWMode
}

// GetMessagesForLLM converts session messages to a format suitable for LLM.
func GetMessagesForLLM(session *Session) []map[string]string {
	var result []map[string]string
	for _, msg := range session.Messages {
		result = append(result, map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}
	return result
}

// SessionSummary provides a brief overview of a session.
type SessionSummary struct {
	ID           string    `json:"id"`
	Name         string    `json:"name,omitempty"`
	MessageCount int       `json:"message_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	FirstMessage string    `json:"first_message,omitempty"`
}

// Summarize returns a summary of the session.
func (s *Session) Summarize() SessionSummary {
	summary := SessionSummary{
		ID:           s.ID,
		Name:         s.Name,
		MessageCount: len(s.Messages),
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}

	// Get first user message as preview
	for _, msg := range s.Messages {
		if msg.Role == "user" {
			preview := msg.Content
			if len(preview) > 50 {
				preview = preview[:47] + "..."
			}
			summary.FirstMessage = preview
			break
		}
	}

	return summary
}
