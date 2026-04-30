package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/OpalBolt/aidir/internal/agent"
	"github.com/OpalBolt/aidir/internal/config"
	"github.com/OpalBolt/aidir/internal/mux"
	"github.com/OpalBolt/aidir/internal/state"
	"github.com/OpalBolt/aidir/internal/worktree"
	"github.com/spf13/cobra"
)

var sessionRestoreCmd = &cobra.Command{
	Use:   "restore [name]",
	Short: "Restore a saved session",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine session file name
		var name string
		if len(args) > 0 {
			name = args[0]
		} else {
			// List sessions and use fzf to pick
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

		// Load session file
		sessDir, err := sessionsDir()
		if err != nil {
			return err
		}

		sessionPath := filepath.Join(sessDir, name+".json")
		data, err := os.ReadFile(sessionPath)
		if err != nil {
			return fmt.Errorf("failed to read session file: %w", err)
		}

		var sessionFile SessionFile
		if err := json.Unmarshal(data, &sessionFile); err != nil {
			return fmt.Errorf("failed to parse session file: %w", err)
		}

		// Load machine config
		machineCfg, err := config.LoadMachineConfig()
		if err != nil {
			return fmt.Errorf("failed to load machine config: %w", err)
		}

		// Load current state
		sf, err := state.Load()
		if err != nil {
			return fmt.Errorf("failed to load state: %w", err)
		}

		// Create multiplexer
		m, err := mux.New(machineCfg.Mux.Backend)
		if err != nil {
			return fmt.Errorf("failed to create multiplexer: %w", err)
		}

		restored := 0
		skipped := 0

		// Restore each slot
		for _, slot := range sessionFile.Slots {
			// Rewrite worktree path using current config root
			repoName := sessionFile.Repo
			if idx := strings.LastIndex(sessionFile.Repo, "/"); idx != -1 {
				repoName = sessionFile.Repo[idx+1:]
			}
			worktreePath := filepath.Join(machineCfg.Worktrees.Root, repoName, strconv.Itoa(slot.IssueID))

			// Check if worktree exists
			if _, err := os.Stat(worktreePath); err == nil {
				// Worktree exists, check branch
				cmd := exec.Command("git", "-C", worktreePath, "rev-parse", "--abbrev-ref", "HEAD")
				var stdout bytes.Buffer
				cmd.Stdout = &stdout

				if err := cmd.Run(); err == nil {
					currentBranch := strings.TrimSpace(stdout.String())
					if currentBranch != slot.Branch {
						fmt.Printf("Warning: #%d on branch %q (expected %q), skipping\n", slot.IssueID, currentBranch, slot.Branch)
						skipped++
						continue
					}
					// Branch matches, reuse worktree
				}
			} else {
				// Worktree doesn't exist, create it
				if err := worktree.Add(worktreePath, slot.Branch); err != nil {
					fmt.Printf("Warning: failed to create worktree for #%d: %v\n", slot.IssueID, err)
					skipped++
					continue
				}
			}

			// Resolve agent
			agentCfg, err := machineCfg.ResolveAgent(slot.AgentName)
			if err != nil {
				fmt.Printf("Warning: failed to resolve agent for #%d: %v\n", slot.IssueID, err)
				skipped++
				continue
			}

			// Build launch command
			launchCmd := agent.BuildLaunchCommand(agentCfg, machineCfg.Sandbox.Backend, worktreePath)

			// Create pane
			paneName := fmt.Sprintf("#%d: %s", slot.IssueID, slot.IssueTitle)
			paneID, err := m.NewPane(paneName, worktreePath, launchCmd)
			if err != nil {
				fmt.Printf("Warning: failed to create pane for #%d: %v\n", slot.IssueID, err)
				skipped++
				continue
			}

			// Skip if already tracked in state (idempotent restore)
			if _, exists := sf.FindByID(slot.IssueID); exists {
				fmt.Printf("Note: #%d already active, skipping state entry\n", slot.IssueID)
				restored++
				continue
			}

			// Add to state
			sess := state.Session{
				IssueID:    slot.IssueID,
				IssueTitle: slot.IssueTitle,
				Repo:       sessionFile.Repo,
				Branch:     slot.Branch,
				Worktree:   worktreePath,
				PaneID:     paneID,
				AgentPID:   0,
				AgentName:  agentCfg.Name,
				Sandboxed:  machineCfg.Sandbox.Backend == "nono" && agentCfg.Sandbox != "none",
				CreatedAt:  time.Now(),
			}
			sf.Add(sess)
			restored++
		}

		// Save state
		if err := sf.Save(); err != nil {
			return fmt.Errorf("failed to save state: %w", err)
		}

		fmt.Printf("Restored %d slots, skipped %d\n", restored, skipped)
		return nil
	},
}

func pickSessionWithFzf() (string, error) {
	sessDir, err := sessionsDir()
	if err != nil {
		return "", err
	}

	// List session files
	entries, err := os.ReadDir(sessDir)
	if err != nil {
		return "", fmt.Errorf("failed to read sessions directory: %w", err)
	}

	var sessionNames []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			name := strings.TrimSuffix(entry.Name(), ".json")
			sessionNames = append(sessionNames, name)
		}
	}

	if len(sessionNames) == 0 {
		return "", fmt.Errorf("no saved sessions found")
	}

	// Build fzf input
	input := strings.Join(sessionNames, "\n")

	// Build preview command using the resolved sessions directory path.
	// exec.Command does not go through a shell, so ~ would not be expanded.
	previewCmd := fmt.Sprintf("cat %s/{}.json", sessDir)
	fzfCmd := exec.Command("fzf", "--preview", previewCmd)
	fzfCmd.Stdin = strings.NewReader(input)
	fzfCmd.Stderr = os.Stderr

	var stdout bytes.Buffer
	fzfCmd.Stdout = &stdout

	if err := fzfCmd.Run(); err != nil {
		// User cancelled
		return "", nil
	}

	return strings.TrimSpace(stdout.String()), nil
}
