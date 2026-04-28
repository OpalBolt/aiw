package mux

import "fmt"

// TmuxMux implements Multiplexer for tmux
type TmuxMux struct{}

// NewPane creates a new pane in tmux
func (t *TmuxMux) NewPane(name, cwd, command string) (string, error) {
	return "", fmt.Errorf("tmux backend not yet implemented (Phase 3)")
}

// FocusPane focuses on a pane in tmux
func (t *TmuxMux) FocusPane(paneID string) error {
	return fmt.Errorf("tmux backend not yet implemented (Phase 3)")
}

// ClosePane closes a pane in tmux
func (t *TmuxMux) ClosePane(paneID string) error {
	return fmt.Errorf("tmux backend not yet implemented (Phase 3)")
}
