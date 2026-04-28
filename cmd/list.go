package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/OpalBolt/aidir/internal/state"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List active sessions",
	RunE: func(cmd *cobra.Command, args []string) error {
		sf, err := state.Load()
		if err != nil {
			return fmt.Errorf("failed to load state: %w", err)
		}

		if len(sf.Sessions) == 0 {
			fmt.Println("No active sessions.")
			return nil
		}

		// Print table
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tTITLE\tREPO\tBRANCH\tWORKTREE\tAGENT")

		for _, sess := range sf.Sessions {
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n",
				sess.IssueID, sess.IssueTitle, sess.Repo, sess.Branch, sess.Worktree, sess.AgentName)
		}

		w.Flush()
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Alias for list",
	RunE:  listCmd.RunE,
}
