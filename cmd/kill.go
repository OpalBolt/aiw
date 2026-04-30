package cmd

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/OpalBolt/aidir/internal/config"
	"github.com/OpalBolt/aidir/internal/mux"
	"github.com/OpalBolt/aidir/internal/state"
	"github.com/OpalBolt/aidir/internal/worktree"
	"github.com/spf13/cobra"
)

var killAll bool

var killCmd = &cobra.Command{
	Use:   "kill [id]",
	Short: "Kill a session or all sessions",
	RunE: func(cmd *cobra.Command, args []string) error {
		sf, err := state.Load()
		if err != nil {
			return fmt.Errorf("failed to load state: %w", err)
		}

		var sessionsToKill []*state.Session

		if killAll {
			// Kill all sessions
			for i := range sf.Sessions {
				sessionsToKill = append(sessionsToKill, &sf.Sessions[i])
			}
		} else {
			// Kill specific session
			if len(args) == 0 {
				return fmt.Errorf("session ID required (or use --all flag)")
			}

			issueID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid issue ID: %w", err)
			}

			sess, found := sf.FindByID(issueID)
			if !found {
				return fmt.Errorf("session #%d not found", issueID)
			}

			sessionsToKill = append(sessionsToKill, sess)
		}

		// Load machine config
		machineCfg, err := config.LoadMachineConfig()
		if err != nil {
			return fmt.Errorf("failed to load machine config: %w", err)
		}

		// Create multiplexer
		m, err := mux.New(machineCfg.Mux.Backend)
		if err != nil {
			return fmt.Errorf("failed to create multiplexer: %w", err)
		}

		// Kill each session
		for _, sess := range sessionsToKill {
			// Kill agent process
			if sess.AgentPID > 0 {
				killProcess(sess.AgentPID)
			}

			// Close pane
			_ = m.ClosePane(sess.PaneID)

			// Remove worktree
			_ = worktree.Remove(sess.Worktree)

			// Remove from state
			sf.Remove(sess.IssueID)

			fmt.Printf("Killed #%d: %s\n", sess.IssueID, sess.IssueTitle)
		}

		// Save state
		if err := sf.Save(); err != nil {
			return fmt.Errorf("failed to save state: %w", err)
		}

		return nil
	},
}

func killProcess(pid int) {
	// Try SIGTERM first
	process, err := os.FindProcess(pid)
	if err != nil {
		// Process doesn't exist
		return
	}

	_ = process.Signal(syscall.SIGTERM)

	// Wait up to 5 seconds for the process to exit
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		// Check if process is still alive
		err := process.Signal(syscall.Signal(0))
		if err != nil {
			// Process is gone
			return
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Force kill with SIGKILL
	_ = process.Signal(syscall.SIGKILL)
}

func init() {
	killCmd.Flags().BoolVar(&killAll, "all", false, "Kill all sessions")
}
