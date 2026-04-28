package mux

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ZellijMux implements Multiplexer for Zellij
type ZellijMux struct{}

// shellQuote wraps s in single quotes, escaping any embedded single quotes.
// Safe for POSIX shell arguments that may contain spaces or special characters.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// NewPane creates a new pane in Zellij and starts the given command inside it.
// Note: zellij action does not expose the new pane's ID, so we always return "".
// The kill sequence closes the active pane when called immediately after setup.
func (z *ZellijMux) NewPane(name, cwd, command string) (string, error) {
	// Create a new pane with a name
	cmd := exec.Command("zellij", "action", "new-pane", "--name", name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("zellij new-pane failed: %w: %s", err, stderr.String())
	}

	// Write the startup command into the new pane. The cwd is shell-quoted to
	// handle paths with spaces. The \n (real newline byte) acts as Enter.
	writeCmd := exec.Command("zellij", "action", "write-chars",
		fmt.Sprintf("cd %s && %s\n", shellQuote(cwd), command))
	writeCmd.Stderr = &stderr

	if err := writeCmd.Run(); err != nil {
		return "", fmt.Errorf("zellij write-chars failed: %w: %s", err, stderr.String())
	}

	// Return empty string as pane ID — zellij's action API does not expose pane IDs.
	return "", nil
}

// FocusPane focuses a pane. Since zellij action does not support targeting by ID,
// this cycles to the next pane as a best-effort approximation.
func (z *ZellijMux) FocusPane(_ string) error {
	cmd := exec.Command("zellij", "action", "focus-next-pane")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: zellij focus-next-pane failed: %v\n", err)
	}

	return nil
}

// ClosePane closes the currently active Zellij pane. Because pane IDs are not
// available via the action API, the paneID argument is ignored and the active
// pane is always closed. Callers should ensure the correct pane is focused
// before invoking this.
func (z *ZellijMux) ClosePane(_ string) error {
	cmd := exec.Command("zellij", "action", "close-pane")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("zellij close-pane failed: %w: %s", err, stderr.String())
	}

	return nil
}
