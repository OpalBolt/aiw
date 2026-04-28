package worktree

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Slug converts a title to a slug (lowercase, alphanumeric + dash, max 40 chars)
func Slug(title string) string {
	// Convert to lowercase
	slug := strings.ToLower(title)

	// Replace non-alphanumeric with dash
	slug = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(slug, "-")

	// Trim leading/trailing dashes
	slug = strings.Trim(slug, "-")

	// Limit to 40 chars
	if len(slug) > 40 {
		slug = slug[:40]
	}

	// Trim trailing dashes again (in case truncation created one)
	slug = strings.TrimRight(slug, "-")

	return slug
}

// BranchName creates a branch name from issue ID and title
func BranchName(issueID int, title string) string {
	return fmt.Sprintf("issue/%d-%s", issueID, Slug(title))
}

// repoName extracts just the repo part from "owner/repo"
func repoName(repo string) string {
	if idx := strings.LastIndex(repo, "/"); idx != -1 {
		return repo[idx+1:]
	}
	return repo
}

// Path returns the worktree path for an issue
func Path(root, repo string, issueID int) string {
	return filepath.Join(root, repoName(repo), strconv.Itoa(issueID))
}

// Add creates a new git worktree. It first tries to create a new branch with -b.
// If that fails specifically because the branch already exists, it retries without
// -b to reuse the existing branch. Any other failure is returned immediately.
func Add(worktreePath, branch string) error {
	var stderr bytes.Buffer

	// First try to create with -b (new branch)
	cmd := exec.Command("git", "worktree", "add", worktreePath, "-b", branch)
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		return nil
	}

	errMsg := stderr.String()

	// Only retry if the branch already exists; any other error is fatal
	if !strings.Contains(errMsg, "already exists") {
		return fmt.Errorf("git worktree add failed: %w: %s", err, errMsg)
	}

	// Branch exists — reuse it without -b
	stderr.Reset()
	cmd = exec.Command("git", "worktree", "add", worktreePath, branch)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git worktree add (reuse branch) failed: %w: %s", err, stderr.String())
	}

	return nil
}

// Remove removes a git worktree
func Remove(worktreePath string) error {
	cmd := exec.Command("git", "worktree", "remove", "--force", worktreePath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("git worktree remove failed: %w: %s", err, stderr.String())
	}

	return nil
}
