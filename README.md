# kanban-md

[![CI](https://github.com/antopolskiy/kanban-md/actions/workflows/build.yml/badge.svg)](https://github.com/antopolskiy/kanban-md/actions/workflows/build.yml)
[![Release](https://github.com/antopolskiy/kanban-md/actions/workflows/release.yml/badge.svg)](https://github.com/antopolskiy/kanban-md/actions/workflows/release.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/antopolskiy/kanban-md)](https://go.dev/)
[![Latest Release](https://img.shields.io/github/v/release/antopolskiy/kanban-md)](https://github.com/antopolskiy/kanban-md/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A file-based Kanban tool powered by Markdown. Tasks are stored as individual `.md` files with YAML frontmatter, making them easy to read, edit, and version-control.

## Why kanban-md?

Most project management tools lock your data behind a UI or an API. kanban-md takes a different approach:

- **Plain files.** Every task is a Markdown file. You can read, edit, grep, and diff them with any tool you already use.
- **Version-controlled by default.** Task files live in your repo alongside code. Every change is a commit. History is free.
- **Built for automation.** Structured JSON output, deterministic file naming, and a CLI-first interface make it easy to script and integrate with AI agents, CI pipelines, or custom tooling.
- **Zero dependencies at runtime.** A single static binary. No database, no server, no config service.

## Installation

### Homebrew (macOS/Linux)

```bash
brew install antopolskiy/tap/kanban-md
```

### Go

```bash
go install github.com/antopolskiy/kanban-md@latest
```

Homebrew also installs `kbmd` as a shorthand alias for `kanban-md`.

### Binary downloads

Pre-built binaries for macOS, Linux, and Windows are available on the [Releases](https://github.com/antopolskiy/kanban-md/releases/latest) page.

## Quick start

```bash
# Initialize a board in the current directory
kanban-md init --name "My Project"

# Create some tasks
kanban-md create "Set up CI pipeline" --priority high --tags devops
kanban-md create "Write API docs" --assignee alice --due 2026-03-01
kanban-md create "Fix login bug" --status todo --priority critical

# List all tasks
kanban-md list

# Filter and sort
kanban-md list --status todo,in-progress --sort priority --reverse

# Move a task forward
kanban-md move 3 in-progress
kanban-md move 3 --next

# Edit a task
kanban-md edit 2 --add-tag documentation --body "Cover all REST endpoints"

# View task details
kanban-md show 1

# Done with a task
kanban-md move 1 done

# Or delete it
kanban-md delete 3 --force
```

## How it works

Running `kanban-md init` creates a `kanban/` directory:

```
kanban/
  config.yml
  tasks/
    001-set-up-ci-pipeline.md
    002-write-api-docs.md
    003-fix-login-bug.md
```

Each task file is standard Markdown with YAML frontmatter:

```markdown
---
id: 1
title: Set up CI pipeline
status: backlog
priority: high
created: 2026-02-07T10:30:00Z
updated: 2026-02-07T10:30:00Z
tags:
  - devops
---

Optional body with more detail, context, or notes.
```

The `config.yml` tracks board settings:

```yaml
version: 1
board:
  name: My Project
tasks_dir: tasks
statuses:
  - backlog
  - todo
  - in-progress
  - review
  - done
priorities:
  - low
  - medium
  - high
  - critical
defaults:
  status: backlog
  priority: medium
next_id: 4
```

## Commands

### `init`

Create a new kanban board.

```bash
kanban-md init [--name NAME] [--statuses s1,s2,s3]
```

| Flag | Description |
|------|-------------|
| `--name` | Board name (defaults to parent directory name) |
| `--statuses` | Comma-separated status list (default: backlog,todo,in-progress,review,done) |

### `create`

Create a new task. Aliases: `add`.

```bash
kanban-md create TITLE [FLAGS]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--status` | backlog | Initial status |
| `--priority` | medium | Priority level |
| `--assignee` | | Person assigned |
| `--tags` | | Comma-separated tags |
| `--due` | | Due date (YYYY-MM-DD) |
| `--estimate` | | Time estimate (e.g. 4h, 2d) |
| `--body` | | Task description |

### `list`

List tasks with filtering and sorting. Aliases: `ls`.

```bash
kanban-md list [FLAGS]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--status` | | Filter by status (comma-separated) |
| `--priority` | | Filter by priority (comma-separated) |
| `--assignee` | | Filter by assignee |
| `--tag` | | Filter by tag |
| `--sort` | id | Sort by: id, status, priority, created, updated, due |
| `-r`, `--reverse` | false | Reverse sort order |
| `-n`, `--limit` | 0 | Max results (0 = unlimited) |

### `show`

Show full details of a task.

```bash
kanban-md show ID
```

### `edit`

Modify an existing task.

```bash
kanban-md edit ID [FLAGS]
kanban-md edit 1,2,3 --priority high  # batch edit
```

| Flag | Description |
|------|-------------|
| `--title` | New title (renames the file) |
| `--status` | New status |
| `--priority` | New priority |
| `--assignee` | New assignee |
| `--add-tag` | Add tags (comma-separated) |
| `--remove-tag` | Remove tags (comma-separated) |
| `--due` | New due date (YYYY-MM-DD) |
| `--clear-due` | Remove due date |
| `--estimate` | New time estimate |
| `--body` | New body text |

### `move`

Change a task's status.

```bash
kanban-md move ID [STATUS]
kanban-md move ID --next
kanban-md move ID --prev
kanban-md move 1,2,3 todo          # batch move
```

| Flag | Description |
|------|-------------|
| `--next` | Advance to next status in the configured order |
| `--prev` | Move back to previous status |
| `--force` | Override WIP limit |

### `delete`

Delete a task. Aliases: `rm`.

```bash
kanban-md delete ID [--force]
kanban-md delete 1,2,3 --force     # batch delete
```

Prompts for confirmation in interactive terminals. Use `--force` to skip the prompt (required in non-interactive contexts like scripts). Batch delete always requires `--force`.

### `config`

View or modify board configuration.

```bash
kanban-md config                       # show all config values
kanban-md config get KEY               # get a single value
kanban-md config set KEY VALUE         # set a writable value
```

Available keys:

| Key | Writable | Description |
|-----|----------|-------------|
| `board.name` | yes | Board name |
| `board.description` | yes | Board description |
| `defaults.status` | yes | Default status for new tasks |
| `defaults.priority` | yes | Default priority for new tasks |
| `statuses` | no | List of statuses |
| `priorities` | no | List of priorities |
| `tasks_dir` | no | Tasks directory name |
| `wip_limits` | no | WIP limits per status |
| `next_id` | no | Next task ID |
| `version` | no | Config schema version |

### `context`

Generate a markdown summary of the board state for embedding in context files (e.g. `CLAUDE.md`, `AGENTS.md`).

```bash
kanban-md context                             # print to stdout
kanban-md context --write-to AGENTS.md        # write/update in file
kanban-md context --sections blocked,overdue  # limit sections
kanban-md context --days 14                   # recently completed lookback
```

| Flag | Default | Description |
|------|---------|-------------|
| `--write-to` | | Write context to file (creates or updates in-place) |
| `--sections` | all | Comma-separated section filter |
| `--days` | 7 | Recently completed lookback in days |

When using `--write-to`, the context block is wrapped in HTML comment markers (`<!-- BEGIN kanban-md context -->` / `<!-- END kanban-md context -->`). If the file already contains these markers, only the block between them is replaced — all other content is preserved.

## Global flags

These work with any command:

| Flag | Description |
|------|-------------|
| `--json` | Force JSON output |
| `--table` | Force table output |
| `--dir` | Path to kanban directory (overrides auto-detection) |
| `--no-color` | Disable color output (also respects `NO_COLOR` env var) |

### Output format detection

kanban-md automatically picks the right output format:

1. `--json` or `--table` flags (highest priority)
2. `KANBAN_OUTPUT` environment variable (`json` or `table`)
3. Auto-detect: **table** in terminals, **JSON** when piped

This makes it easy to use in scripts:

```bash
# Parse with jq
kanban-md list --status todo --json | jq '.[].title'

# Or let auto-detection handle it
kanban-md list --status todo | jq '.[].title'  # piped = JSON automatically
```

## Configuration

kanban-md discovers its config by walking upward from the current directory, similar to how `git` finds `.git/`. This means you can run commands from any subdirectory in your project.

Use `--dir` to point to a specific board:

```bash
kanban-md --dir /path/to/kanban list
```

### Custom statuses

Define your own workflow columns:

```bash
kanban-md init --statuses "open,in-progress,blocked,closed"
```

The order matters — it defines the progression for `move --next` and `move --prev`, and the sort order for `list --sort status`.

### Custom priorities

Edit `config.yml` directly to customize priorities:

```yaml
priorities:
  - trivial
  - normal
  - urgent
  - showstopper
defaults:
  priority: normal
```

## Shell completions

Generate completions for your shell:

```bash
# bash
source <(kanban-md completion bash)

# zsh
kanban-md completion zsh > "${fpath[1]}/_kanban-md"

# fish
kanban-md completion fish | source

# PowerShell
kanban-md completion powershell | Out-String | Invoke-Expression
```

## Design principles

**Files are the API.** The CLI is a convenience layer over a simple file format. You can always fall back to editing files directly — the tool will pick up changes.

**Predictable output.** JSON output follows a stable schema. Table output is human-friendly. The format auto-switches based on context so scripts and humans both get what they need.

**No hidden state.** Everything is in `config.yml` and the task files. There's no database, no cache, no lock file. Two people (or agents) can work on the same board by editing different files and merging via git.

**Minimal by default.** The tool does one thing — manage task files — and stays out of the way. It doesn't sync, notify, render boards, or integrate with anything. Those are better handled by other tools reading the same files.

## Development

```bash
# Build
make build

# Run all tests (unit + e2e)
make test

# Run only e2e tests
make test-e2e

# Lint
make lint

# Full pipeline
make all
```

## License

[MIT](LICENSE)
