package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/OpalBolt/aidir/internal/agent"
	"github.com/OpalBolt/aidir/internal/config"
	"github.com/OpalBolt/aidir/internal/gh"
	"github.com/OpalBolt/aidir/internal/mux"
	"github.com/OpalBolt/aidir/internal/state"
	"github.com/OpalBolt/aidir/internal/worktree"
	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Create new worktrees for selected issues",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configs
		machineCfg, err := config.LoadMachineConfig()
		if err != nil {
			return fmt.Errorf("failed to load machine config: %w", err)
		}

		projectCfg, err := config.LoadProjectConfig()
		if err != nil {
			return fmt.Errorf("failed to load project config: %w", err)
		}

		// Detect repo
		repo, err := gh.DetectRepo()
		if err != nil {
			return fmt.Errorf("failed to detect repository: %w", err)
		}

		// List issues
		issues, err := gh.ListIssues(projectCfg.Issues.Assignee, projectCfg.Issues.Labels, projectCfg.Issues.Limit)
		if err != nil {
			return fmt.Errorf("failed to list issues: %w", err)
		}

		// Resolve current user for sorting/annotation (best-effort)
		me, _ := gh.CurrentUser()

		// Pick issues
		selected, err := gh.PickIssues(issues, me)
		if err != nil {
			return fmt.Errorf("failed to pick issues: %w", err)
		}

		if len(selected) == 0 {
			fmt.Println("No issues selected")
			return nil
		}

		// Load state
		sf, err := state.Load()
		if err != nil {
			return fmt.Errorf("failed to load state: %w", err)
		}

		// Create multiplexer
		m, err := mux.New(machineCfg.Mux.Backend)
		if err != nil {
			return fmt.Errorf("failed to create multiplexer: %w", err)
		}

		// Process each selected issue
		for _, issue := range selected {
			// Create worktree path
			worktreePath := worktree.Path(machineCfg.Worktrees.Root, repo, issue.Number)

			// Create branch name
			branchName := worktree.BranchName(issue.Number, issue.Title)

			// Add worktree
			if err := worktree.Add(worktreePath, branchName); err != nil {
				fmt.Fprintf(os.Stderr, "Error: failed to create worktree for #%d: %v\n", issue.Number, err)
				continue
			}

			// Resolve agent
			agentCfg, err := machineCfg.ResolveAgent(projectCfg.Agent.Name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: failed to resolve agent: %v\n", err)
				continue
			}

			// Build launch command
			launchCmd := agent.BuildLaunchCommand(agentCfg, machineCfg.Sandbox.Backend, worktreePath)

			// Create pane
			paneName := fmt.Sprintf("#%d: %s", issue.Number, issue.Title)
			paneID, err := m.NewPane(paneName, worktreePath, launchCmd)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: failed to create pane: %v\n", err)
				continue
			}

			// Add to state
			sess := state.Session{
				IssueID:    issue.Number,
				IssueTitle: issue.Title,
				Repo:       repo,
				Branch:     branchName,
				Worktree:   worktreePath,
				PaneID:     paneID,
				AgentPID:   0,
				AgentName:  agentCfg.Name,
				Sandboxed:  machineCfg.Sandbox.Backend == "nono" && agentCfg.Sandbox != "none",
				CreatedAt:  time.Now(),
			}
			sf.Add(sess)

			// Print confirmation
			fmt.Printf("Started #%d: %s in %s\n", issue.Number, issue.Title, worktreePath)
		}

		// Save state
		if err := sf.Save(); err != nil {
			return fmt.Errorf("failed to save state: %w", err)
		}

		return nil
	},
}
