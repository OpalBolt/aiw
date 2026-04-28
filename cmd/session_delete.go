package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var sessionDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete a saved session",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine session name
		var name string
		if len(args) > 0 {
			name = args[0]
		} else {
			// Use fzf to pick
			var err error
			name, err = pickSessionWithFzf()
			if err != nil {
				return err
			}
			if name == "" {
				fmt.Println("Cancelled")
				return nil
			}
		}

		// Confirm deletion
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("Delete session '%s'? [y/N]: ", name)
		response, _ := reader.ReadString('\n')
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(response)), "y") {
			fmt.Println("Cancelled")
			return nil
		}

		// Delete file
		sessDir, err := sessionsDir()
		if err != nil {
			return err
		}

		sessionPath := filepath.Join(sessDir, name+".json")
		if err := os.Remove(sessionPath); err != nil {
			return fmt.Errorf("failed to delete session file: %w", err)
		}

		fmt.Printf("Session '%s' deleted\n", name)
		return nil
	},
}
