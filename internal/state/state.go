package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Session represents a single active session
type Session struct {
	IssueID      int       `json:"issue_id"`
	IssueTitle   string    `json:"issue_title"`
	Repo         string    `json:"repo"`
	Branch       string    `json:"branch"`
	Worktree     string    `json:"worktree"`
	ZellijPaneID string    `json:"zellij_pane_id"`
	AgentPID     int       `json:"agent_pid"`
	AgentName    string    `json:"agent_name"`
	Sandboxed    bool      `json:"sandboxed"`
	CreatedAt    time.Time `json:"created_at"`
}

// StateFile represents the state file
type StateFile struct {
	Sessions []Session `json:"sessions"`
}

// statePath returns the path to the state file
func statePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	stateDir := filepath.Join(home, ".local", "share", "aiw")
	return filepath.Join(stateDir, "state.json"), nil
}

// Load loads the state file or returns an empty StateFile if it doesn't exist
func Load() (*StateFile, error) {
	path, err := statePath()
	if err != nil {
		return nil, err
	}

	sf := &StateFile{Sessions: []Session{}}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, return empty state
			return sf, nil
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	// Parse the JSON
	if err := json.Unmarshal(data, sf); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	return sf, nil
}

// Save writes the state file atomically
func (s *StateFile) Save() error {
	path, err := statePath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write atomically (write to .tmp, rename)
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath) // Clean up on error
		return fmt.Errorf("failed to rename state file: %w", err)
	}

	return nil
}

// Add appends a session to the state
func (s *StateFile) Add(sess Session) {
	s.Sessions = append(s.Sessions, sess)
}

// Remove removes a session by issue ID
func (s *StateFile) Remove(issueID int) {
	for i, sess := range s.Sessions {
		if sess.IssueID == issueID {
			s.Sessions = append(s.Sessions[:i], s.Sessions[i+1:]...)
			return
		}
	}
}

// FindByID finds a session by issue ID
func (s *StateFile) FindByID(issueID int) (*Session, bool) {
	for i, sess := range s.Sessions {
		if sess.IssueID == issueID {
			return &s.Sessions[i], true
		}
	}
	return nil, false
}
