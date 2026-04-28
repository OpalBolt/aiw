package gh

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Issue represents a GitHub issue
type Issue struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	Labels    []struct {
		Name string `json:"name"`
	} `json:"labels"`
	Assignees []struct {
		Login string `json:"login"`
	} `json:"assignees"`
}

// DetectRepo detects the GitHub repository from git remote
func DetectRepo() (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get git remote: %w", err)
	}

	url := strings.TrimSpace(stdout.String())

	// Handle https://github.com/owner/repo.git format
	if strings.HasPrefix(url, "https://") {
		url = strings.TrimPrefix(url, "https://github.com/")
		url = strings.TrimSuffix(url, ".git")
		return url, nil
	}

	// Handle git@github.com:owner/repo.git format
	if strings.HasPrefix(url, "git@github.com:") {
		url = strings.TrimPrefix(url, "git@github.com:")
		url = strings.TrimSuffix(url, ".git")
		return url, nil
	}

	return "", fmt.Errorf("could not parse repository URL: %s", url)
}

// CurrentUser returns the login of the authenticated GitHub user.
func CurrentUser() (string, error) {
	cmd := exec.Command("gh", "api", "user", "--jq", ".login")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gh api user failed: %w: %s", err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

// ListIssues lists issues from GitHub
func ListIssues(assignee string, labels []string, limit int) ([]Issue, error) {
	args := []string{
		"issue", "list",
		"--state", "open",
		"--json", "number,title,labels,assignees",
		"--limit", strconv.Itoa(limit),
	}

	if assignee != "" {
		args = append(args, "--assignee", assignee)
	}

	for _, label := range labels {
		args = append(args, "--label", label)
	}

	cmd := exec.Command("gh", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gh issue list failed: %w: %s", err, stderr.String())
	}

	var issues []Issue
	if err := json.Unmarshal(stdout.Bytes(), &issues); err != nil {
		return nil, fmt.Errorf("failed to parse issues: %w", err)
	}

	return issues, nil
}

// PickIssues uses fzf to pick issues from a list. Issues assigned to `me`
// are sorted first and marked with a ★ indicator.
func PickIssues(issues []Issue, me string) ([]Issue, error) {
	// Partition: mine first, then the rest.
	var mine, others []Issue
	for _, issue := range issues {
		assigned := false
		for _, a := range issue.Assignees {
			if strings.EqualFold(a.Login, me) {
				assigned = true
				break
			}
		}
		if assigned {
			mine = append(mine, issue)
		} else {
			others = append(others, issue)
		}
	}
	sorted := append(mine, others...)

	// Format issues for fzf
	var input strings.Builder
	for _, issue := range sorted {
		labels := ""
		if len(issue.Labels) > 0 {
			var labelNames []string
			for _, l := range issue.Labels {
				labelNames = append(labelNames, l.Name)
			}
			labels = " [" + strings.Join(labelNames, ", ") + "]"
		}
		indicator := "  "
		for _, a := range issue.Assignees {
			if strings.EqualFold(a.Login, me) {
				indicator = "★ "
				break
			}
		}
		// Use bare number as first field so fzf {1} passes a bare int to "gh issue view {1}"
		fmt.Fprintf(&input, "%d %s%s%s\n", issue.Number, indicator, issue.Title, labels)
	}

	// Run fzf with preview. {1} is the first field (bare number, no #).
	fzfCmd := exec.Command(
		"fzf",
		"--multi",
		"--preview", "gh issue view {1}",
	)
	fzfCmd.Stdin = strings.NewReader(input.String())
	fzfCmd.Stderr = os.Stderr

	var stdout bytes.Buffer
	fzfCmd.Stdout = &stdout

	if err := fzfCmd.Run(); err != nil {
		// User cancelled
		return []Issue{}, nil
	}

	// Parse selected lines
	var selected []Issue
	selectedLines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	for _, line := range selectedLines {
		if line == "" {
			continue
		}

		// Parse "<number> <title> [labels]" — first field is bare number
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		numberStr := parts[0]
		number, err := strconv.Atoi(numberStr)
		if err != nil {
			continue
		}

		// Find the matching issue
		for _, issue := range issues {
			if issue.Number == number {
				selected = append(selected, issue)
				break
			}
		}
	}

	return selected, nil
}
