package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// AgentConfig represents a single agent profile
type AgentConfig struct {
	Name    string   `toml:"name"`
	Command string   `toml:"command"`
	Args    []string `toml:"args"`
	Sandbox string   `toml:"sandbox"`
}

// WorktreesConfig represents the worktrees section
type WorktreesConfig struct {
	Root string `toml:"root"`
}

// SandboxConfig represents the sandbox section
type SandboxConfig struct {
	Backend string `toml:"backend"`
}

// MuxConfig represents the mux section
type MuxConfig struct {
	Backend string `toml:"backend"`
}

// MachineConfig represents the global machine configuration
type MachineConfig struct {
	Worktrees WorktreesConfig `toml:"worktrees"`
	Sandbox   SandboxConfig   `toml:"sandbox"`
	Mux       MuxConfig       `toml:"mux"`
	Agents    []AgentConfig   `toml:"agents"`
}

// AgentConfig in project config
type ProjectAgentConfig struct {
	Name string `toml:"name"`
}

// IssuesConfig represents the issues section
type IssuesConfig struct {
	Assignee string   `toml:"assignee"`
	Labels   []string `toml:"labels"`
	Limit    int      `toml:"limit"`
}

// ProjectConfig represents the project configuration
type ProjectConfig struct {
	Agent  ProjectAgentConfig `toml:"agent"`
	Issues IssuesConfig       `toml:"issues"`
}

// LoadMachineConfig loads the machine config from ~/.config/aiw/config.toml
func LoadMachineConfig() (*MachineConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(home, ".config", "aiw", "config.toml")

	cfg := &MachineConfig{
		Mux: MuxConfig{Backend: "tmux"}, // default
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, return defaults with expanded worktree root
			if cfg.Worktrees.Root == "" {
				cfg.Worktrees.Root = filepath.Join(home, "worktrees")
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read machine config: %w", err)
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse machine config: %w", err)
	}

	// Expand ~ in worktree root
	if strings.HasPrefix(cfg.Worktrees.Root, "~/") {
		cfg.Worktrees.Root = filepath.Join(home, cfg.Worktrees.Root[2:])
	}

	// Apply defaults for fields that may not be set in the config file
	if cfg.Worktrees.Root == "" {
		cfg.Worktrees.Root = filepath.Join(home, "worktrees")
	}
	if cfg.Mux.Backend == "" {
		cfg.Mux.Backend = "tmux"
	}

	return cfg, nil
}

// ResolveAgent finds an agent by name. If name is empty, returns the first configured agent.
func (m *MachineConfig) ResolveAgent(name string) (*AgentConfig, error) {
	if name == "" {
		if len(m.Agents) > 0 {
			return &m.Agents[0], nil
		}
		return nil, fmt.Errorf("no agents configured")
	}
	for i, agent := range m.Agents {
		if agent.Name == name {
			return &m.Agents[i], nil
		}
	}
	return nil, fmt.Errorf("agent %q not found", name)
}

// LoadProjectConfig loads the project config by walking up from cwd
func LoadProjectConfig() (*ProjectConfig, error) {
	path, err := findProjectConfig()
	if err != nil {
		// Not found, return defaults
		return &ProjectConfig{
			Issues: IssuesConfig{Limit: 50},
		}, nil
	}

	cfg := &ProjectConfig{
		Issues: IssuesConfig{Limit: 50},
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read project config: %w", err)
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse project config: %w", err)
	}

	return cfg, nil
}

// findProjectConfig walks up from cwd looking for .aiw.toml
func findProjectConfig() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	for {
		candidate := filepath.Join(cwd, ".aiw.toml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}

		parent := filepath.Dir(cwd)
		if parent == cwd {
			// Reached root
			return "", os.ErrNotExist
		}
		cwd = parent
	}
}
