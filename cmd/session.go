package cmd

import "github.com/spf13/cobra"

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage saved sessions",
}

func init() {
	sessionCmd.AddCommand(sessionSaveCmd)
	sessionCmd.AddCommand(sessionRestoreCmd)
	sessionCmd.AddCommand(sessionListCmd)
	sessionCmd.AddCommand(sessionDeleteCmd)
}
