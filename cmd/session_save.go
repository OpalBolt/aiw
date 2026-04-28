package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/OpalBolt/aidir/internal/state"
	"github.com/spf13/cobra"
)

// SessionFile represents a saved session file
type SessionFile struct {
	Name    string        `json:"name"`
	Repo    string        `json:"repo"`
	SavedAt time.Time     `json:"saved_at"`
	Slots   []SessionSlot `json:"slots"`
}

// SessionSlot represents a saved issue slot
type SessionSlot struct {
	IssueID    int    `json:"issue_id"`
	IssueTitle string `json:"issue_title"`
	Branch     string `json:"branch"`
	Worktree   string `json:"worktree"`
	AgentName  string `json:"agent_name"`
}

var sessionSaveCmd = &cobra.Command{
	Use:   "save [name]",
	Short: "Save current sessions",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load state
		sf, err := state.Load()
		if err != nil {
			return fmt.Errorf("failed to load state: %w", err)
		}

		if len(sf.Sessions) == 0 {
			return fmt.Errorf("no active sessions to save")
		}

		// Determine name
		var name string
		if len(args) > 0 {
			name = args[0]
		} else {
			// Default: <reponame>-<date>
			repo := sf.Sessions[0].Repo
			repoName := repo
			if idx := strings.LastIndex(repo, "/"); idx != -1 {
				repoName = repo[idx+1:]
			}
			name = fmt.Sprintf("%s-%s", repoName, time.Now().Format("2006-01-02"))
		}

		// Get sessions directory
		sessDir, err := sessionsDir()
		if err != nil {
			return err
		}

		// Ensure sessions directory exists
		if err := os.MkdirAll(sessDir, 0755); err != nil {
			return fmt.Errorf("failed to create sessions directory: %w", err)
		}

		// Check if file exists
		sessionPath := filepath.Join(sessDir, name+".json")
		if _, err := os.Stat(sessionPath); err == nil {
			// File exists, prompt for confirmation
			reader := bufio.NewReader(os.Stdin)
			fmt.Printf("Overwrite session '%s'? [y/N]: ", name)
			response, _ := reader.ReadString('\n')
			if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(response)), "y") {
				fmt.Println("Cancelled")
				return nil
			}
		}

		// Create session file
		sessionFile := SessionFile{
			Name:    name,
			Repo:    sf.Sessions[0].Repo,
			SavedAt: time.Now(),
		}

		for _, sess := range sf.Sessions {
			slot := SessionSlot{
				IssueID:    sess.IssueID,
				IssueTitle: sess.IssueTitle,
				Branch:     sess.Branch,
				Worktree:   sess.Worktree,
				AgentName:  sess.AgentName,
			}
			sessionFile.Slots = append(sessionFile.Slots, slot)
		}

		// Write file
		data, err := json.MarshalIndent(sessionFile, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal session: %w", err)
		}

		if err := os.WriteFile(sessionPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write session file: %w", err)
		}

		fmt.Printf("Session '%s' saved (%d slots)\n", name, len(sessionFile.Slots))
		return nil
	},
}

func sessionsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(home, ".local", "share", "aiw", "sessions"), nil
}
