# aiw — AI Worktree Manager

A CLI tool for orchestrating multiple AI coding agents across git worktrees in a structured Zellij workspace.

---

## What it does

Running multiple AI agents in parallel normally requires manually picking issues, creating git worktrees, opening terminal panes, and keeping track of what agent is doing what. `aiw` automates the entire flow:

1. You run `aiw new`
2. You pick GitHub issues from a fuzzy-finder
3. `aiw` creates a dedicated git worktree and branch for each issue, opens a Zellij pane, and starts your configured AI agent inside it

Each agent works in isolation — it only sees its own worktree.

---

## Requirements

| Tool | Purpose |
|------|---------|
| `git` | Worktree management |
| `gh` | GitHub CLI (issue fetching, authentication) |
| `zellij` | Terminal pane management |
| `fzf` | Interactive issue picker |
| `nono` | Optional: filesystem sandbox for agents |

AI agents (`claude`, `opencode`, etc.) are configured per-machine and are not required at install time.

---

## Installation

### From source

```bash
go install github.com/OpalBolt/aidir@latest
```

### Nix

```bash
nix run github:OpalBolt/aidir
```

Or add to your flake:

```nix
inputs.aiw.url = "github:OpalBolt/aidir";
```

### Development environment

```bash
nix develop   # drops into a shell with all tools pinned
make build    # builds ./aiw
```

---

## Quick start

**1. Configure your machine** (`~/.config/aiw/config.toml`, never committed):

```toml
[worktrees]
root = "~/worktrees"

[sandbox]
backend = "none"   # or "nono" to sandbox agents

[[agents]]
name    = "claude"
command = "claude"
args    = []

[[agents]]
name    = "opencode"
command = "opencode"
args    = []
```

**2. Configure your project** (`.aiw.toml`, safe to commit):

```toml
[agent]
name = "claude"

[issues]
assignee = "@me"
limit    = 50

[zellij]
tab = "ai-work"
```

**3. Run it** (from inside a git repo):

```bash
aiw new
```

A fuzzy-finder opens with your open issues. Select one or more, press Enter, and `aiw` handles the rest.

---

## Commands

### `aiw new`

Pick open GitHub issues and spin up worktrees + panes + agents for each one.

```
aiw new
```

Flow:
1. Detect repo from `git remote get-url origin`
2. Fetch open issues via `gh issue list`
3. Open `fzf` multi-select picker (preview: `gh issue view <id>`)
4. For each selected issue:
   - Create `~/worktrees/<repo>/<issue-id>/` with branch `issue/<id>-<slug>`
   - Open a Zellij pane named `#<id>: <title>`
   - Run the configured agent inside it (optionally sandboxed with `nono`)

### `aiw list` / `aiw status`

Show all active sessions tracked in state.

```
aiw list

ID    TITLE                    REPO         BRANCH                    WORKTREE              AGENT
123   Fix login bug            owner/repo   issue/123-fix-login-bug   ~/worktrees/repo/123  claude
145   Add OAuth support        owner/repo   issue/145-add-oauth       ~/worktrees/repo/145  claude
```

### `aiw kill <id>`

Tear down a session: close the Zellij pane, remove the worktree, and clear the state entry.

```bash
aiw kill 123       # kill session for issue #123
aiw kill --all     # kill all active sessions
```

Kill sequence:
1. Close Zellij pane
2. `git worktree remove --force <path>`
3. Remove from state

### `aiw attach <id>`

Focus the Zellij pane for an issue.

```bash
aiw attach 123
```

> Note: Zellij's action API does not support targeting panes by ID, so this cycles to the next pane as a best-effort approximation.

### `aiw session save [name]`

Snapshot the current workspace to a named session file.

```bash
aiw session save auth-sprint
```

If `name` is omitted, defaults to `<repo>-<date>`. Prompts before overwriting an existing session.

### `aiw session restore [name]`

Rebuild a saved workspace: recreate worktrees, open panes, start agents.

```bash
aiw session restore auth-sprint
aiw session restore   # fzf picker if name omitted
```

Restore is idempotent — running it twice skips sessions that are already active. Cross-machine restore rewrites worktree paths using the local `worktrees.root` from config.

### `aiw session list`

List all saved sessions.

```
NAME            REPO          SLOTS  SAVED
auth-sprint     owner/repo    3      2026-04-17
api-refactor    owner/repo    2      2026-04-15
```

### `aiw session delete [name]`

Delete a saved session file. Prompts for confirmation.

```bash
aiw session delete auth-sprint
aiw session delete   # fzf picker if name omitted
```

---

## Configuration reference

### `~/.config/aiw/config.toml` — machine-local, never committed

```toml
[worktrees]
root = "~/worktrees"        # where all worktrees are created

[sandbox]
backend = "nono"            # "nono" | "none" (default: none)

[[agents]]
name    = "claude"
command = "claude"
args    = []

[[agents]]
name    = "opencode"
command = "opencode"
args    = []
sandbox = "none"            # per-agent override: skip sandbox for this agent
```

### `.aiw.toml` — project-local, safe to commit

```toml
[agent]
name = "claude"             # must match a name in ~/.config/aiw/config.toml

[issues]
labels   = []               # only show issues with these labels (empty = all)
assignee = "@me"            # "@me" | "" (empty = all issues)
limit    = 50

[zellij]
layout = "default"
tab    = "ai-work"          # create panes in this named tab
```

---

## Sandbox

When `sandbox.backend = "nono"`, each agent is wrapped with `nono run`, restricting its filesystem access to its own worktree:

```
nono run --allow ~/worktrees/repo/123 -- claude
```

This means a rogue or hallucinating agent cannot read or write outside its assigned worktree. Agents are isolated from each other and from the rest of your filesystem.

To opt a specific agent out:

```toml
[[agents]]
name    = "opencode"
command = "opencode"
sandbox = "none"
```

---

## State files

| Path | Contents |
|------|---------|
| `~/.local/share/aiw/state.json` | Active sessions (worktrees currently open) |
| `~/.local/share/aiw/sessions/<name>.json` | Saved session snapshots |

State is written atomically (write to `.tmp`, then rename). These files are machine-local and should not be committed.

---

## Known limitations (Phase 1)

- **Zellij pane targeting**: `zellij action` does not expose pane IDs. `aiw kill` closes the currently active pane rather than targeting a specific one by ID. For best results, use `aiw kill` immediately after `aiw attach`.
- **Agent PID tracking**: Agents are started via `zellij action write-chars`, so their PIDs are not tracked. Killing a session terminates it by closing the pane.
- **Config discovery**: `.aiw.toml` is loaded from the current working directory only — it is not walked up to the git root.
- **Multi-repo sessions**: Session save records all active sessions regardless of repo. Mixed-repo sessions will warn on restore.

---

## Development

```bash
make build       # build ./aiw
make test        # run tests
make lint        # run golangci-lint
make fmt         # format all Go source
make fmt-check   # check formatting (used in CI)
make tidy        # go mod tidy + verify
make snapshot    # local goreleaser snapshot (no publish)
make clean       # remove binary and dist/
```

CI runs on every PR and push to `main`: format check → lint → test → build.

Releases are automated via [release-please](https://github.com/googleapis/release-please-action). Write commits using [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add session restore
fix: handle missing worktree on kill
chore: update deps
```

release-please opens a release PR when there are releasable commits. Merging it tags the release and triggers goreleaser to publish binaries for `linux/darwin × amd64/arm64`.

---

## Roadmap

**Phase 2 — Quality of life**
- `aiw gc` — auto-cleanup of merged/closed issues
- `aiw open` — open the issue in the browser from inside the pane
- `aiw init` — interactive first-time setup and config validation

**Phase 3 — tmux support**
- Abstract the terminal multiplexer behind an interface (already stubbed)
- `[mux] backend = "tmux"` in machine config

**Phase 4 — Issue context injection**
- Write `ISSUE.md` into each worktree with the issue body, comments, and linked PRs
- Agent sees full context as soon as it opens
