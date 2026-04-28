package mux

import "fmt"

// Multiplexer interface for terminal multiplexers
type Multiplexer interface {
	NewPane(name, cwd, command string) (string, error) // returns pane ID
	FocusPane(paneID string) error
	ClosePane(paneID string) error
}

// New creates a multiplexer for the given backend
func New(backend string) (Multiplexer, error) {
	switch backend {
	case "tmux":
		return &TmuxMux{}, nil
	case "zellij", "":
		return &ZellijMux{}, nil
	default:
		return nil, fmt.Errorf("unknown multiplexer backend: %s", backend)
	}
}
