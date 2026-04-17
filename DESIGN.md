# aiw — AI Worktree Manager

A CLI tool for orchestrating multiple AI agents across git worktrees in a structured Zellij workspace.

---

## Problem

Running multiple AI coding agents in parallel requires manually:
- Picking which issues to work on
- Creating git worktrees for isolation
- Launching terminal sessions with the right working directory
- Keeping track of what agent is doing what

This is error-prone and slow. `aiw` automates the entire flow.

---

## Core Concept

Each GitHub issue gets:
- A dedicated **git worktree** at `~/worktrees/<repo>/<issue-id>/`
- A dedicated **Zellij pane** named after the issue
- An **AI agent process** started automatically inside that pane

The user picks issues → `aiw` does the rest.

---

## CLI Interface

```
aiw new              # pick issues → spin up panes + worktrees + agents
aiw list             # show active worktrees and their status
aiw attach <id>      # focus the Zellij pane for a given issue
aiw kill <id>        # tear down pane + worktree for an issue
aiw kill --all       # tear down everything
aiw status           # show all panes: issue, agent PID, worktree path

aiw session save [name]      # snapshot current workspace to a named session
aiw session restore [name]   # restore a saved session (panes + worktrees + agents)
aiw session list             # show all saved sessions
aiw session delete [name]    # remove a saved session record
```

---

## `aiw new` Flow

```
aiw new
  │
  ├─ 1. Detect repo          (git remote → owner/repo)
  ├─ 2. Fetch open issues    (gh issue list --json)
  ├─ 3. Present picker       (fzf, multi-select)
  │
  └─ For each selected issue:
       ├─ 4. Create worktree   ~/worktrees/<repo>/<issue-id>/
       ├─ 5. Create branch     issue/<issue-id>-<slug>
       ├─ 6. Open Zellij pane  named "<issue-id>: <title>"
       └─ 7. Start AI agent    (configured per-project or globally)
```

### Step details

**Repo detection**
```
git remote get-url origin → parse owner/repo
```

**Issue fetch**
```
gh issue list --state open --json number,title,labels,assignees --limit 50
```

**Picker**
```
fzf --multi --preview "gh issue view {1}"
Display: "#123 Fix login bug [bug, priority:high]"
```

**Worktree creation**
```
WORKTREE_ROOT=~/worktrees/<repo>/<issue-id>
git worktree add $WORKTREE_ROOT -b issue/<issue-id>-<slug>
```

**Zellij pane**
```
zellij action new-pane --name "#<issue-id>: <title>"
zellij action write-chars "cd $WORKTREE_ROOT && <agent-launch-cmd>\n"
```

The agent launch command is assembled from the agent config, including any sandbox wrapper (see Configuration).

---

## Configuration

Config is split into two files with a clear separation of concerns:

| File | Scope | Checked in? |
|---|---|---|
| `~/.config/aiw/config.toml` | Machine-specific: agents, sandbox, worktree root | No — never |
| `.aiw.toml` (repo root) | Project-specific: issue filters, zellij tab, which agent profile to use by name | Yes |

This means agent definitions and sandbox config are per-machine (the user decides which tools are installed and trusted), while project preferences travel with the repo.

```toml
# ~/.config/aiw/config.toml  — machine-local, never committed

[worktrees]
root = "~/worktrees"        # where all worktrees are created

[sandbox]
backend = "nono"            # "nono" | "none" (default: none)

[[agents]]
name    = "claude"
command = "claude"
args    = []

[[agents]]
name    = "copilot"
command = "gh"
args    = ["copilot", "suggest", "-t", "shell"]

[[agents]]
name    = "opencode"
command = "opencode"
args    = []
sandbox = "none"            # per-agent override: skip sandbox for this agent
```

```toml
# .aiw.toml  — project-local, safe to commit

[agent]
name = "claude"             # must match a name in ~/.config/aiw/config.toml

[issues]
labels = []                 # filter: only show issues with these labels
assignee = "@me"            # filter: only assigned to me (empty = all)
limit = 50

[zellij]
layout = "default"          # zellij layout to use for new panes
tab = "ai-work"             # create panes in this named tab (optional)
```

**Supported agents:**
- `claude` — Claude Code CLI (`claude`)
- `copilot` — GitHub Copilot CLI (`gh copilot`)
- `opencode` — OpenCode (`opencode`)
- Any arbitrary command can be added as a custom profile

### Sandbox wrapping

When `sandbox.backend = "nono"`, `aiw` wraps each agent launch in `nono run`, granting it access **only to its own worktree**. This means each agent is capability-restricted to the files it actually needs.

The assembled launch command becomes:
```
nono run --allow <worktree_path> -- <agent_command> <agent_args>
```

Example for Claude on issue #123:
```
nono run --allow ~/worktrees/repo/123 -- claude
```

This is a significant security property: a rogue or hallucinating agent cannot read or write outside its assigned worktree. Agents cannot see each other's work mid-flight.

Per-agent sandbox overrides are supported:
```toml
[[agents]]
name    = "opencode"
command = "opencode"
sandbox = "none"            # override: run this agent unsandboxed
```

---

## State Tracking

`aiw` maintains a local state file at `~/.local/share/aiw/state.json` to track active sessions:

```json
{
  "sessions": [
    {
      "issue_id": 123,
      "issue_title": "Fix login bug",
      "repo": "owner/repo",
      "branch": "issue/123-fix-login-bug",
      "worktree": "/home/user/worktrees/repo/123",
      "zellij_pane_id": "abc123",
      "agent_pid": 45123,
      "agent_name": "claude",
      "sandboxed": true,
      "created_at": "2026-04-17T22:14:00Z"
    }
  ]
}
```

---

## Session Save & Restore

A **session** is a named snapshot of the current workspace: which issues are open, their worktrees, and their branches. Saving captures everything needed to rebuild the workspace from scratch. Restoring replays it — even across reboots or on a different machine.

Sessions are stored at:
```
~/.local/share/aiw/sessions/<name>.json
```

### Session file format

```json
{
  "name": "auth-sprint",
  "repo": "owner/repo",
  "saved_at": "2026-04-17T22:00:00Z",
  "slots": [
    {
      "issue_id": 123,
      "issue_title": "Fix login bug",
      "branch": "issue/123-fix-login-bug",
      "worktree": "/home/user/worktrees/repo/123",
      "agent_name": "claude"
    },
    {
      "issue_id": 145,
      "issue_title": "Add OAuth support",
      "branch": "issue/145-add-oauth-support",
      "worktree": "/home/user/worktrees/repo/145",
      "agent_name": "claude"
    }
  ]
}
```

Note: Zellij pane IDs and agent PIDs are **not** saved — these are transient. Only the durable state (issue, branch, worktree path, agent command) is persisted.

### `aiw session save [name]`

```
1. Read current state.json (active slots)
2. Prompt for name if not provided (default: repo name + date)
3. Write session file to ~/.local/share/aiw/sessions/<name>.json
4. Print: "Session 'auth-sprint' saved (3 slots)"
```

If a session with that name already exists, prompt to overwrite.

### `aiw session restore [name]`

```
1. Load session file
2. If name omitted: fzf picker over saved sessions
3. For each slot:
   a. Check if worktree already exists on disk
      - Yes + branch matches → reuse as-is
      - Yes + branch mismatch → warn, skip
      - No → git worktree add (branch must exist remotely or locally)
   b. Open Zellij pane named "#<id>: <title>"
   c. cd into worktree, start agent
   d. Write entry to state.json
4. Print summary: restored N slots, skipped M
```

Restore is **idempotent** for existing worktrees — re-running it only opens new panes for slots that aren't already live.

### `aiw session list`

```
NAME            REPO          SLOTS  SAVED
auth-sprint     owner/repo    3      2026-04-17
api-refactor    owner/repo    2      2026-04-15
```

### Restore across machines

Because worktree paths are absolute, cross-machine restore replaces the path prefix automatically:
- Saved path: `/home/alice/worktrees/repo/123`
- Current home: `/home/bob` → restored to `/home/bob/worktrees/repo/123`

The session file's `worktree` paths are treated as `<worktrees_root>/<repo>/<id>` tuples, not literal strings, so the root from config is always applied on restore.

---

## Zellij Integration

### Pane naming
Zellij supports naming panes via `zellij action rename-pane`. The name is set to `#<id>: <title>` so issues are identifiable at a glance.

### Tab grouping
Optionally, all `aiw` panes live in a dedicated tab (e.g. `ai-work`) so they don't pollute the main workspace.

### Layout (future)
A custom Zellij layout could be defined that shows:
- Left: AI agent pane (large)
- Right: `aiw status` sidebar (small, auto-refreshing)

```kdl
// ~/.config/zellij/layouts/aiw.kdl
layout {
    pane size=1 borderless=true {
        plugin location="zellij:tab-bar"
    }
    pane split_direction="vertical" {
        pane size="80%"   // agent pane
        pane size="20%" { // status sidebar
            command "watch"
            args "-n2" "aiw status"
        }
    }
}
```

---

## Worktree Lifecycle

```
Created by:   aiw new
Cleaned up by: aiw kill <id>

Kill sequence:
  1. Send SIGTERM to agent PID
  2. Wait for exit (timeout 5s, then SIGKILL)
  3. zellij action close-pane <pane-id>
  4. git worktree remove ~/worktrees/<repo>/<id>
  5. Optionally: git branch -d issue/<id>-<slug>
  6. Remove entry from state.json
```

---

## Language Recommendation: Go

**Recommendation: Go**

| Concern | Why Go wins here |
|---|---|
| Single binary | No runtime, no interpreter — `go build` produces one file, easy to install with `go install` or a shell script |
| Fast startup | Critical for a CLI invoked constantly; Go starts in ~5ms vs. Python's ~100ms+ |
| Process management | `os/exec` is ergonomic and well-tested for spawning, signalling, and waiting on child processes |
| JSON state files | `encoding/json` + typed structs — no third-party deps needed |
| CLI subcommands | `cobra` is the standard, used by `gh`, `kubectl`, `docker` — subcommand pattern is a first-class concept |
| Shell-out to zellij/git/gh | Simple `exec.Command()` calls — no friction |
| Cross-platform | Relevant when tmux backend lands; Go's cross-compile is trivial |
| Error handling | Explicit errors make it clear when zellij/git/gh invocations fail and why |

**Ruled out:**

- **Shell (bash)** — tempting for a v0 but JSON state management, error handling across multiple subprocess calls, and the session restore logic will become unmaintainable fast
- **Python** — good ecosystem, but requires the user to have the right version + virtualenv, slow startup, packaging is painful
- **Rust** — startup and binary size are excellent but the learning curve and compile times create friction during the early UX iteration phase
- **TypeScript** — npm dependency graph for a CLI tool is a liability; startup time via Node is noticeable

**Structure:**

```
aiw/
├── cmd/
│   ├── root.go          # root cobra command, global flags
│   ├── new.go           # aiw new
│   ├── list.go          # aiw list / status
│   ├── kill.go          # aiw kill
│   ├── attach.go        # aiw attach
│   └── session/
│       ├── save.go
│       ├── restore.go
│       ├── list.go
│       └── delete.go
├── internal/
│   ├── config/          # load .aiw.toml + ~/.config/aiw/config.toml
│   ├── state/           # read/write state.json
│   ├── worktree/        # git worktree operations
│   ├── mux/             # zellij + tmux backends
│   │   ├── interface.go
│   │   ├── zellij.go
│   │   └── tmux.go
│   ├── agent/           # agent profile resolution + sandbox wrapping
│   └── gh/              # gh CLI wrapper (issue fetch, issue view)
└── main.go
```

---

## Tooling & Infrastructure

### Repository layout

```
aiw/
├── .github/
│   ├── workflows/
│   │   ├── ci.yml              # lint, fmt, test, build — runs on every PR
│   │   ├── release-please.yml  # manages release PRs and tags
│   │   └── release.yml         # goreleaser — triggered by tags from release-please
│   └── release-please-config.json
├── cmd/                        # cobra commands (see Language section)
├── internal/                   # internal packages
├── flake.nix                   # Nix dev environment + aiw package output
├── flake.lock
├── .goreleaser.yaml            # goreleaser build + archive config
├── Makefile                    # developer-facing build targets
├── go.mod
├── go.sum
└── main.go
```

---

### Nix

Nix is a first-class citizen. Two outputs are provided:

**`devShells.default`** — reproducible dev environment with every tool pinned:
```nix
# flake.nix (sketch)
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let pkgs = nixpkgs.legacyPackages.${system}; in {

        # `nix develop` → full dev environment
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            golangci-lint
            goreleaser
            gh
            fzf
            zellij
            gnumake
            nono          # sandbox tool
          ];
        };

        # `nix build` → produces the aiw binary
        packages.default = pkgs.buildGoModule {
          pname = "aiw";
          version = "0.0.0-dev";
          src = ./.;
          vendorHash = null; # fill in after go mod vendor
        };

      });
}
```

`flake.lock` is committed and updated deliberately (`nix flake update`), not automatically.

**Why not make Go deps a Nix input?** `buildGoModule` handles vendoring internally via `vendorHash`. This keeps the Nix integration thin — Nix manages the toolchain, Go modules manage their own deps.

---

### Makefile

The Makefile is the single interface for all build operations, whether you're in a Nix shell or not. CI uses the same targets as local dev.

```makefile
BIN     := aiw
VERSION ?= $(shell git describe --tags --always --dirty)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build test lint fmt fmt-check tidy snapshot clean

build:                          ## Build the binary
	go build $(LDFLAGS) -o $(BIN) .

test:                           ## Run tests
	go test ./...

lint:                           ## Run golangci-lint
	golangci-lint run ./...

fmt:                            ## Format all Go source
	gofmt -w .

fmt-check:                      ## Check formatting (used in CI)
	@diff=$$(gofmt -l .); \
	if [ -n "$$diff" ]; then echo "Unformatted files:\n$$diff"; exit 1; fi

tidy:                           ## Tidy and verify go.mod
	go mod tidy
	go mod verify

snapshot:                       ## Build a local goreleaser snapshot (no publish)
	goreleaser release --snapshot --clean

clean:
	rm -f $(BIN) dist/
```

---

### goreleaser

goreleaser handles cross-compilation and archive creation. It is triggered only by CI on a tag push (see below) — never run locally except via `make snapshot` for testing.

```yaml
# .goreleaser.yaml
version: 2

builds:
  - env: [CGO_ENABLED=0]
    goos: [linux, darwin]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w -X main.version={{.Version}}

archives:
  - format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}"

checksum:
  name_template: "checksums.txt"

changelog:
  use: github-native   # pull from release-please's generated notes

release:
  github:
    owner: "{{ .Env.GITHUB_REPOSITORY_OWNER }}"
    name: aiw
```

---

### Release flow (release-please + goreleaser)

The release process is fully automated and convention-driven:

```
developer writes commits using Conventional Commits format
  │  feat: add session restore
  │  fix: handle missing worktree on kill
  │  chore: update deps
  ▼
release-please action (runs on every push to main)
  │  reads commits since last release
  │  opens/updates a "Release PR" with:
  │    - bumped version in version.go (or wherever)
  │    - auto-generated CHANGELOG.md entry
  ▼
developer merges the Release PR
  │
  ▼
release-please creates a git tag (e.g. v0.3.0)
  │
  ▼
release.yml workflow triggers on tag push
  │  runs goreleaser
  │  produces binaries for linux/darwin × amd64/arm64
  │  creates GitHub Release with archives + checksums
```

```yaml
# .github/workflows/release-please.yml
on:
  push:
    branches: [main]

jobs:
  release-please:
    runs-on: ubuntu-latest
    steps:
      - uses: googleapis/release-please-action@v4
        with:
          release-type: go
          token: ${{ secrets.GITHUB_TOKEN }}
```

```yaml
# .github/workflows/release.yml
on:
  push:
    tags: ["v*"]

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0       # goreleaser needs full history
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: goreleaser/goreleaser-action@v6
        with:
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

---

### CI workflow

Runs on every PR and every push to main. All checks must pass before merge (enforced by branch protection).

```yaml
# .github/workflows/ci.yml
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  ci:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
          cache: true

      - name: Verify dependencies
        run: make tidy && git diff --exit-code go.sum

      - name: Check formatting
        run: make fmt-check

      - name: Lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: latest

      - name: Test
        run: make test

      - name: Build
        run: make build
```

---

### Branch protection (main)

Main is locked. Configure in GitHub → Settings → Branches → Branch protection rules:

| Rule | Setting |
|---|---|
| Require pull request before merging | Yes |
| Required approving reviews | 0 (solo project) — increase for teams |
| Require status checks to pass | `ci / ci` |
| Require branches to be up to date | Yes |
| Do not allow bypassing | Yes |
| Allow force pushes | No |
| Allow deletions | No |

---

## Implementation Plan

### Phase 1 — Core (MVP)

| # | Task |
|---|------|
| 1 | `aiw new`: detect repo, fetch issues, fzf picker |
| 2 | Worktree creation + branch naming |
| 3 | Zellij pane creation + cd + agent start |
| 4 | State file read/write |
| 5 | `aiw list` / `aiw status` |
| 6 | `aiw kill <id>` |
| 7 | `aiw session save` / `restore` / `list` / `delete` |

**Language:** Go. See Language Recommendation section for rationale.

### Phase 2 — Quality of Life

- `aiw attach <id>` — focus the correct Zellij pane
- Auto-cleanup of merged/closed issues (`aiw gc`)
- `aiw open` — open the issue in the browser while in the pane
- Config validation and `aiw init` for first-time setup

### Phase 3 — tmux support

Abstract the terminal multiplexer behind an interface:

```
interface Multiplexer {
  new_pane(name, cwd, command) -> pane_id
  focus_pane(pane_id)
  close_pane(pane_id)
}
```

Implementations: `ZellijMux`, `TmuxMux`.
Selected via config:
```toml
[mux]
backend = "zellij"   # or "tmux"
```

### Phase 4 — Issue context injection

Before starting the agent, write a context file into the worktree:

```
~/worktrees/<repo>/<id>/ISSUE.md
```

Populated with the issue body, comments, and linked PRs from `gh issue view`. The AI agent sees this as immediate context when it opens.

---

## Dependencies

| Tool | Purpose | Required |
|------|---------|----------|
| `git` | Worktree management | Yes |
| `gh` | GitHub issue fetching | Yes |
| `zellij` | Pane management | Yes (Phase 1) |
| `fzf` | Issue picker TUI | Yes |
| `nono` | Sandbox wrapper for agents | No (opt-in via config) |
| `claude` | Claude Code agent | No (configured per-machine) |
| `gh copilot` | GitHub Copilot agent | No (configured per-machine) |
| `opencode` | OpenCode agent | No (configured per-machine) |
| `tmux` | Alternative mux backend | No (Phase 3) |

---

## Open Questions

1. **Branch conflicts** — what if the branch already exists? (offer to reuse or skip)
2. **Dirty worktrees on kill** — warn the user if the worktree has uncommitted changes
3. **Zellij session** — should `aiw` require an existing Zellij session, or start one?
4. **Issue context** — should `ISSUE.md` be gitignored automatically?
5. **Agent flags** — should `aiw` pass the issue title/number to the agent as an initial prompt?
6. **Session restore + closed issues** — what if the issue was closed since the session was saved? Warn and skip, or restore anyway?
7. **Session portability** — should sessions be shareable (checked into the repo) or always user-local?

---

## Non-Goals (v1)

- Managing CI/CD or PR creation (use `gh pr create` manually or via the agent)
- Supporting non-GitHub issue trackers (Linear, Jira) — future
- A TUI dashboard — `aiw status` as a plain table is enough for now
