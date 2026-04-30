package mux

import (
	"fmt"
	"os/exec"
	"strings"
)

// TmuxMux implements Multiplexer for tmux
type TmuxMux struct {
	session string // explicit session name; if empty, queried from the running tmux
}

// getSession returns the target session name by asking tmux directly.
func (t *TmuxMux) getSession() (string, error) {
	if t.session != "" {
		return t.session, nil
	}
	out, err := exec.Command("tmux", "display-message", "-p", "#{session_name}").Output()
	if err != nil {
		return "", fmt.Errorf("not inside a tmux session (tmux display-message failed): %w", err)
	}
	name := strings.TrimSpace(string(out))
	if name == "" {
		return "", fmt.Errorf("could not determine tmux session name")
	}
	return name, nil
}

// NewPane creates a new tmux window, runs command in cwd, and returns the pane ID.
// The pane ID format is "<window_index>:0" (e.g. "3:0").
func (t *TmuxMux) NewPane(name, cwd, command string) (string, error) {
	session, err := t.getSession()
	if err != nil {
		return "", err
	}

	// new-window -P prints the new window's index; -c sets the start directory;
	// passing the command avoids send-keys races.
	out, err := exec.Command("tmux", "new-window",
		"-t", session,
		"-n", name,
		"-c", cwd,
		"-P", "-F", "#{window_index}",
		"--", command,
	).Output()
	if err != nil {
		return "", fmt.Errorf("tmux new-window failed: %w", err)
	}

	windowIndex := strings.TrimSpace(string(out))
	if windowIndex == "" {
		return "", fmt.Errorf("tmux new-window returned empty window index")
	}

	return windowIndex + ":0", nil
}

// FocusPane selects the tmux window containing the given pane ID.
// paneID format: "<window_index>:0" (as returned by NewPane).
func (t *TmuxMux) FocusPane(paneID string) error {
	session, err := t.getSession()
	if err != nil {
		return err
	}

	target := session + ":" + paneID
	if err := exec.Command("tmux", "select-window", "-t", target).Run(); err != nil {
		return fmt.Errorf("tmux select-window failed: %w", err)
	}
	return nil
}

// ClosePane kills the tmux window for the given pane ID.
// paneID format: "<window_index>:0" (as returned by NewPane).
func (t *TmuxMux) ClosePane(paneID string) error {
	if paneID == "" {
		return fmt.Errorf("cannot close pane: empty pane ID")
	}

	session, err := t.getSession()
	if err != nil {
		return err
	}

	// paneID is "window_index:pane_index"; kill the whole window
	windowIndex := strings.SplitN(paneID, ":", 2)[0]
	target := session + ":" + windowIndex

	if err := exec.Command("tmux", "kill-window", "-t", target).Run(); err != nil {
		return fmt.Errorf("tmux kill-window failed: %w", err)
	}
	return nil
}
