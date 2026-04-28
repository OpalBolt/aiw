package agent

import (
	"fmt"
	"strings"

	"github.com/OpalBolt/aidir/internal/config"
)

// BuildLaunchCommand builds the command to launch an agent
func BuildLaunchCommand(agentCfg *config.AgentConfig, sandboxBackend, worktreePath string) string {
	// Build the base command
	cmd := agentCfg.Command
	if len(agentCfg.Args) > 0 {
		cmd = cmd + " " + strings.Join(agentCfg.Args, " ")
	}

	// Wrap with sandbox if needed
	if sandboxBackend == "nono" && agentCfg.Sandbox != "none" {
		cmd = fmt.Sprintf("nono run --allow %s -- %s", worktreePath, cmd)
	}

	// Trim trailing space
	return strings.TrimSpace(cmd)
}
