# kanban-md Design Spec & Implementation Plan

## Context

kanban-md is a file-based Kanban CLI tool for managing tasks as Markdown files with YAML frontmatter. Designed for AI agents and humans. The project is scaffolded with Go 1.25.7 + Cobra at `/Users/santop/Projects/kanban-md`.

---

## 1. Directory & File Structure

```
myproject/
  kanban/                   # visible directory (configurable)
    config.yml              # all settings, no hidden variables
    tasks/
      001-setup-database.md
      002-design-api.md
      003-auth-flow.md
```

- `kanban/` is the default data directory (visible, not hidden)
- Overridable via `config.yml`, `--dir` flag, or `KANBAN_DIR` env var
- CLI finds config by walking upward from CWD (like git finds `.git/`)
- Single board per project

## 2. Config File (`kanban/config.yml`)

```yaml
version: 1

board:
  name: "My Project"
  description: ""

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

next_id: 1    # managed by CLI, never reused
```

Everything from `init` is reflected here. No hidden state.

## 3. Task File Format

**Filename:** `{NNN}-{slug}.md` — e.g. `001-setup-database.md`

- NNN: zero-padded (min 3 digits), auto-incremented
- Slug: auto-generated from title (lowercase, hyphens, max 50 chars)

**Example task file:**

```markdown
---
id: 1
title: "Set up PostgreSQL database"
status: todo
priority: high
created: 2026-02-07T10:30:00-05:00
updated: 2026-02-07T14:22:00-05:00
assignee: "santiago"
tags:
  - backend
  - infrastructure
due: 2026-02-14
estimate: "4h"
parent: null
depends_on: []
---

Set up PostgreSQL 16 for the project with the initial schema.

- Database running in Docker for local dev
- Migration tool configured
- Initial schema with users table
```

**Fields:** id, title, status, priority, created, updated (core/required); assignee, tags, due, estimate, parent, depends_on (optional). Optional fields omitted from file when empty.

## 4. Output System

- **Auto-detect:** TTY → human-readable table, piped → JSON
- **Override:** `--json` or `--table` flags, or `KANBAN_OUTPUT` env var
- JSON: arrays for lists (`[]` when empty), objects for single items, errors on stderr
- Tables: lipgloss styling, color-coded priorities, `--` for empty fields

## 5. CLI Commands (Phase 1 — MVP)

### Global flags (persistent on root)

`--json`, `--table`, `--dir`, `--no-color`

### `init`

```
kanban-md init [--name NAME] [--statuses backlog,todo,...] [--dir kanban]
```

Creates `kanban/config.yml` + `kanban/tasks/` directory.

### `create` (alias: `add`)

```
kanban-md create "Set up database" [--status todo] [--priority high] [--assignee santiago] [--tags backend,infra] [--due 2026-02-14] [--estimate 4h] [--body "..."] [--edit]
```

Auto-assigns next ID, generates slug, writes file, increments `next_id`.

### `list` (alias: `ls`)

```
kanban-md list [--status in-progress,review] [--priority high] [--assignee santiago] [--tag backend] [--sort status] [--reverse] [--limit 10]
```

Scans all task files, filters (AND logic), sorts, outputs table or JSON.

### `show`

```
kanban-md show 1
```

Full task detail including markdown body.

### `edit`

```
kanban-md edit 1 [--status in-progress] [--priority medium] [--add-tag api] [--remove-tag infra] [--clear-due] [--edit-body]
```

Modifies frontmatter fields, preserves body, updates `updated` timestamp. Renames file if title changes.

### `move`

```
kanban-md move 1 in-progress
kanban-md move 1 --next
kanban-md move 1 --prev
```

Shortcut for status changes. `--next`/`--prev` follow configured status order.

### `delete` (alias: `rm`)

```
kanban-md delete 1 [--force]
```

Confirmation prompt in TTY, requires `--force` when piped. Warns about orphaned references.

## 6. Go Package Layout

```
cmd/
  root.go           # root command + global flags
  init.go           # init command
  create.go         # create/add command
  list.go           # list/ls command
  show.go           # show command
  edit.go           # edit command
  move.go           # move command
  delete.go         # delete/rm command
internal/
  config/
    config.go       # Config struct, Load, Save, Validate, FindConfigDir
    defaults.go     # default values
  task/
    task.go         # Task struct
    file.go         # read/write task files (frontmatter + body)
    slug.go         # title → slug conversion
    find.go         # find task file by ID
    validate.go     # validation (status, priority, references)
  board/
    board.go        # list all tasks
    filter.go       # filter predicates
    sort.go         # sort comparators
  output/
    output.go       # format detection, Render dispatch
    table.go        # lipgloss table formatter
    json.go         # JSON formatter
  date/
    date.go         # custom Date type (YYYY-MM-DD)
```

## 7. Dependencies to Add


| Library                             | Purpose                                   |
| ----------------------------------- | ----------------------------------------- |
| `go.yaml.in/yaml/v3`                | YAML parsing for config + frontmatter     |
| `github.com/adrg/frontmatter`       | Split YAML frontmatter from markdown body |
| `github.com/charmbracelet/lipgloss` | Terminal table styling                    |
| `golang.org/x/term`                 | TTY detection                             |


## 8. Implementation Order

### Layer 0: Foundation (internal packages, no CLI)

1. `internal/date` — Date type with YAML/JSON marshaling
2. `internal/config` — Config struct, load/save/validate, directory walk
3. `internal/task` — Task struct, slug gen, file read/write, find by ID, validation
4. `internal/board` — List all tasks, filter, sort
5. `internal/output` — Format detection, JSON output, table output

### Layer 1: CLI commands (one at a time)

1. `cmd/init.go` — everything else needs a config
2. `cmd/create.go` — need tasks to test other commands
3. `cmd/list.go` — most used command
4. `cmd/show.go` — single task detail
5. `cmd/edit.go` — most complex (many flags, file rename)
6. `cmd/move.go` — thin wrapper around edit
7. `cmd/delete.go` — simple

### Layer 2: Polish

1. Wire global flags on root command
2. Add `kbmd` alias in GoReleaser
3. End-to-end tests
4. Shell completions (Cobra built-in)

## 9. Verification

After each layer:

- `go test -race ./...` passes
- `go build -o /dev/null ./...` compiles
- Manual test: `go run . init && go run . create "Test task" && go run . list`
- CI passes on push

After Layer 1 complete:

- Full workflow test: init → create tasks → list/filter → show → edit → move → delete
- JSON output piped through `jq` validates correctly
- Test with `--json` and TTY auto-detect

