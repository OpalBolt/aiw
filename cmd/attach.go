package cmd

import (
	"fmt"
	"strconv"

	"github.com/OpalBolt/aidir/internal/config"
	"github.com/OpalBolt/aidir/internal/mux"
	"github.com/OpalBolt/aidir/internal/state"
	"github.com/spf13/cobra"
)

var attachCmd = &cobra.Command{
	Use:   "attach <id>",
	Short: "Attach to an existing session",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		issueID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid issue ID: %w", err)
		}

		sf, err := state.Load()
		if err != nil {
			return fmt.Errorf("failed to load state: %w", err)
		}

		sess, found := sf.FindByID(issueID)
		if !found {
			return fmt.Errorf("session #%d not found", issueID)
		}

		machineCfg, err := config.LoadMachineConfig()
		if err != nil {
			return fmt.Errorf("failed to load machine config: %w", err)
		}

		m, err := mux.New(machineCfg.Mux.Backend)
		if err != nil {
			return fmt.Errorf("failed to create multiplexer: %w", err)
		}

		if err := m.FocusPane(sess.PaneID); err != nil {
			return fmt.Errorf("failed to focus pane: %w", err)
		}

		return nil
	},
}
