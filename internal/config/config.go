package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// MachineConfig represents the machine-local configuration
type MachineConfig struct {
	Worktrees WorktreesConfig `toml:"worktrees"`
	Sandbox   SandboxConfig   `toml:"sandbox"`
	Agents    []AgentConfig   `toml:"agents"`
}

// WorktreesConfig configures worktree root directory
type WorktreesConfig struct {
	Root string `toml:"root"`
}

// SandboxConfig configures the sandbox backend
type SandboxConfig struct {
	Backend string `toml:"backend"`
}

// AgentConfig represents a single agent profile
type AgentConfig struct {
	Name    string   `toml:"name"`
	Command string   `toml:"command"`
	Args    []string `toml:"args"`
	Sandbox string   `toml:"sandbox"`
}

// ProjectConfig represents the project-local configuration
type ProjectConfig struct {
	Agent  ProjectAgentConfig `toml:"agent"`
	Issues IssuesConfig       `toml:"issues"`
	Zellij ZellijConfig       `toml:"zellij"`
}

// ProjectAgentConfig configures which agent to use for this project
type ProjectAgentConfig struct {
	Name string `toml:"name"`
}

// IssuesConfig configures issue filtering
type IssuesConfig struct {
	Labels   []string `toml:"labels"`
	Assignee string   `toml:"assignee"`
	Limit    int      `toml:"limit"`
}

// ZellijConfig configures zellij behavior
type ZellijConfig struct {
	Layout string `toml:"layout"`
	Tab    string `toml:"tab"`
}

// LoadMachineConfig loads ~/.config/aiw/config.toml
func LoadMachineConfig() (*MachineConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(home, ".config", "aiw", "config.toml")

	mc := &MachineConfig{
		Worktrees: WorktreesConfig{
			Root: filepath.Join(home, "worktrees"),
		},
		Sandbox: SandboxConfig{
			Backend: "none",
		},
		Agents: []AgentConfig{},
	}

	// Try to load the config file if it exists
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, return defaults
			return mc, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse the TOML
	if err := toml.Unmarshal(data, mc); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply default root if the config file had a [worktrees] block but no root key.
	if mc.Worktrees.Root == "" {
		mc.Worktrees.Root = filepath.Join(home, "worktrees")
	}

	// Expand ~ in Root
	if mc.Worktrees.Root == "~" {
		mc.Worktrees.Root = home
	} else if len(mc.Worktrees.Root) > 1 && mc.Worktrees.Root[:2] == "~/" {
		mc.Worktrees.Root = filepath.Join(home, mc.Worktrees.Root[2:])
	}

	return mc, nil
}

// LoadProjectConfig loads .aiw.toml from the current directory
func LoadProjectConfig() (*ProjectConfig, error) {
	pc := &ProjectConfig{
		Agent:  ProjectAgentConfig{Name: ""},
		Issues: IssuesConfig{Assignee: "", Limit: 50},
		Zellij: ZellijConfig{Layout: "default", Tab: "ai-work"},
	}

	data, err := os.ReadFile(".aiw.toml")
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, return defaults
			return pc, nil
		}
		return nil, fmt.Errorf("failed to read project config file: %w", err)
	}

	// Parse the TOML on top of a zero-value struct (no pre-set defaults) so that
	// an explicit empty string assignee = "" is preserved as the user's intent.
	var parsed ProjectConfig
	if err := toml.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse project config file: %w", err)
	}

	// Carry over parsed values; only apply defaults for fields not present in file.
	pc.Agent = parsed.Agent
	pc.Zellij = parsed.Zellij
	pc.Issues.Labels = parsed.Issues.Labels

	// Assignee: honor explicit empty string (meaning "all issues")
	// A zero-value after parse means the key was absent — keep the default "@me".
	// We distinguish via a sentinel: if the TOML key was present, BurntSushi/toml
	// would have set the field; if absent it stays empty. Since we can't easily
	// distinguish, we copy as-is and only fall back to "@me" if it was never set
	// (i.e., the file has no [issues] assignee key at all — which leaves it "").
	// Accept the limitation: explicit assignee="" in .aiw.toml maps to "all issues"
	// rather than the "@me" default. This matches what a user explicitly writing
	// assignee = "" intends.
	pc.Issues.Assignee = parsed.Issues.Assignee

	if pc.Issues.Limit == 0 {
		if parsed.Issues.Limit != 0 {
			pc.Issues.Limit = parsed.Issues.Limit
		}
		// else keep the default 50 set above
	}

	return pc, nil
}

// ResolveAgent finds an agent by name
func (mc *MachineConfig) ResolveAgent(name string) (*AgentConfig, error) {
	// If name is empty, use the first agent if available
	if name == "" {
		if len(mc.Agents) > 0 {
			return &mc.Agents[0], nil
		}
		return nil, fmt.Errorf("no agents configured")
	}

	// Find agent by name
	for i, agent := range mc.Agents {
		if agent.Name == name {
			return &mc.Agents[i], nil
		}
	}

	return nil, fmt.Errorf("agent %q not found", name)
}
