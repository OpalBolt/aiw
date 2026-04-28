package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "aiw",
	Short: "AI-assisted Git worktree manager",
	Long: `aiw is a tool for managing Git worktrees with AI agent integration.
It helps you create, track, and manage parallel work on GitHub issues.`,
}

// Execute runs the root command
func Execute(version string) {
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(killCmd)
	rootCmd.AddCommand(attachCmd)
	rootCmd.AddCommand(sessionCmd)
}
