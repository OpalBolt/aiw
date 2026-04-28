package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var sessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all saved sessions",
	RunE: func(cmd *cobra.Command, args []string) error {
		sessDir, err := sessionsDir()
		if err != nil {
			return err
		}

		// List session files
		entries, err := os.ReadDir(sessDir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("No saved sessions")
				return nil
			}
			return fmt.Errorf("failed to read sessions directory: %w", err)
		}

		// Collect session info
		type sessionInfo struct {
			name  string
			repo  string
			slots int
			saved string
		}

		var sessions []sessionInfo

		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
				name := strings.TrimSuffix(entry.Name(), ".json")
				sessionPath := filepath.Join(sessDir, entry.Name())

				data, err := os.ReadFile(sessionPath)
				if err != nil {
					continue
				}

				var sessionFile SessionFile
				if err := json.Unmarshal(data, &sessionFile); err != nil {
					continue
				}

				sessions = append(sessions, sessionInfo{
					name:  name,
					repo:  sessionFile.Repo,
					slots: len(sessionFile.Slots),
					saved: sessionFile.SavedAt.Format("2006-01-02"),
				})
			}
		}

		if len(sessions) == 0 {
			fmt.Println("No saved sessions")
			return nil
		}

		// Print table
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tREPO\tSLOTS\tSAVED")

		for _, s := range sessions {
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", s.name, s.repo, s.slots, s.saved)
		}

		w.Flush()
		return nil
	},
}
