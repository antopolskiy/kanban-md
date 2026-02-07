---
name: kanban
description: >
  Manage project tasks using kanban-md, a file-based kanban board CLI.
  Use when the user mentions tasks, kanban, board, backlog, sprint,
  project management, work items, priorities, blockers, or wants to
  track, create, list, move, edit, or delete tasks. Also use for
  standup, status update, sprint planning, triage, or project metrics.
allowed-tools:
  - Bash(kanban-md *)
  - Bash(kbmd *)
---

# kanban-md

Manage kanban boards stored as Markdown files with YAML frontmatter.
Each task is a `.md` file in `kanban/tasks/`. The CLI is `kanban-md`
(alias `kbmd` if installed via Homebrew).

## Current Board State

!`kanban-md board --json 2>/dev/null || echo '{"error": "no board found — run: kanban-md init --name PROJECT_NAME"}'`

## Rules

- Always pass `--json` for programmatic output. Parse JSON, never scrape table output.
- Always pass `--force` when deleting (`kanban-md delete ID --force --json`).
- Dates use `YYYY-MM-DD` format.
- Statuses and priorities are board-specific. Check the board state above or run
  `kanban-md board --json` to discover valid values before using them.
- Default statuses: backlog, todo, in-progress, review, done.
- Default priorities: low, medium, high, critical.
- Present results to the user in readable format (tables, bullets), not raw JSON.
- For JSON output schemas, see [references/json-schemas.md](references/json-schemas.md).

## Decision Tree

| User wants to...                        | Command                                                     |
|-----------------------------------------|-------------------------------------------------------------|
| See board overview / standup            | `kanban-md board --json`                                    |
| List all tasks                          | `kanban-md list --json`                                     |
| List tasks by status                    | `kanban-md list --status todo,in-progress --json`           |
| List tasks by priority                  | `kanban-md list --priority high,critical --json`            |
| List tasks by assignee                  | `kanban-md list --assignee alice --json`                    |
| List tasks by tag                       | `kanban-md list --tag bug --json`                           |
| List blocked tasks                      | `kanban-md list --blocked --json`                           |
| List ready-to-start tasks              | `kanban-md list --not-blocked --status todo --json`         |
| List tasks with resolved deps           | `kanban-md list --unblocked --json`                         |
| Find a specific task                    | `kanban-md show ID --json`                                  |
| Create a task                           | `kanban-md create "TITLE" --priority P --tags T --json`     |
| Create a task with body                 | `kanban-md create "TITLE" --body "DESC" --json`             |
| Start working on a task                 | `kanban-md move ID in-progress --json`                      |
| Advance to next status                  | `kanban-md move ID --next --json`                           |
| Move a task back                        | `kanban-md move ID --prev --json`                           |
| Complete a task                         | `kanban-md move ID done --json`                             |
| Edit task fields                        | `kanban-md edit ID --title "NEW" --priority P --json`       |
| Add/remove tags                         | `kanban-md edit ID --add-tag T --remove-tag T --json`       |
| Set a due date                          | `kanban-md edit ID --due 2026-03-01 --json`                 |
| Block a task                            | `kanban-md edit ID --block "REASON" --json`                 |
| Unblock a task                          | `kanban-md edit ID --unblock --json`                        |
| Add a dependency                        | `kanban-md edit ID --add-dep DEP_ID --json`                 |
| Set a parent task                       | `kanban-md edit ID --parent PARENT_ID --json`               |
| Delete a task                           | `kanban-md delete ID --force --json`                        |
| See flow metrics                        | `kanban-md metrics --json`                                  |
| See activity log                        | `kanban-md log --limit 20 --json`                           |
| See recent activity for a task          | `kanban-md log --task ID --json`                            |
| Initialize a new board                  | `kanban-md init --name "NAME"`                              |

## Core Commands

### list

```
kanban-md list [--status S] [--priority P] [--assignee A] [--tag T] \
  [--sort FIELD] [-r] [-n LIMIT] [--blocked] [--not-blocked] \
  [--parent ID] [--unblocked] --json
```

Sort fields: id, status, priority, created, updated, due. `-r` reverses.
`--unblocked` shows tasks whose dependencies are all at terminal status.
Returns a JSON array of task objects.

### create

```
kanban-md create "TITLE" [--status S] [--priority P] [--assignee A] \
  [--tags T1,T2] [--due YYYY-MM-DD] [--estimate E] [--body "TEXT"] \
  [--parent ID] [--depends-on ID1,ID2] --json
```

Returns the created task object with its assigned ID.

### show

```
kanban-md show ID --json
```

Returns a single task object including body text.

### edit

```
kanban-md edit ID [--title T] [--status S] [--priority P] [--assignee A] \
  [--add-tag T] [--remove-tag T] [--due YYYY-MM-DD] [--clear-due] \
  [--estimate E] [--body "TEXT"] [--started YYYY-MM-DD] [--clear-started] \
  [--completed YYYY-MM-DD] [--clear-completed] [--parent ID] \
  [--clear-parent] [--add-dep ID] [--remove-dep ID] \
  [--block "REASON"] [--unblock] --json
```

Only specified fields are changed. Returns the updated task object.

### move

```
kanban-md move ID STATUS --json
kanban-md move ID --next --json
kanban-md move ID --prev --json
```

Use `-f` to override WIP limits. Auto-sets Started on first move from
initial status. Auto-sets Completed on move to terminal status.
Returns a task object with a `changed` boolean field.

### delete

```
kanban-md delete ID --force --json
```

Always pass `--force` (non-interactive context requires it).

### board

```
kanban-md board --json
```

Returns board overview: task counts per status, WIP utilization,
blocked/overdue counts, priority distribution.

### metrics

```
kanban-md metrics [--since YYYY-MM-DD] --json
```

Returns throughput (7d/30d), avg lead/cycle time, flow efficiency,
aging items.

### log

```
kanban-md log [--since YYYY-MM-DD] [--limit N] [--action TYPE] \
  [--task ID] --json
```

Action types: create, move, edit, delete, block, unblock.

### Global Flags

All commands accept: `--json`, `--table`, `--dir PATH`, `--no-color`.

## Workflows

### Daily Standup

1. `kanban-md board --json` — board overview
2. `kanban-md list --status in-progress --json` — in-flight work
3. `kanban-md list --blocked --json` — stuck items
4. `kanban-md metrics --json` — throughput and aging
5. Summarize: completed, active, blocked, aging items

### Triage New Work

1. `kanban-md list --status backlog --sort priority -r --json` — review backlog
2. For items to promote: `kanban-md move ID todo --json`
3. For new items: `kanban-md create "TITLE" --priority P --tags T --json`
4. For stale items: `kanban-md delete ID --force --json`

### Sprint Planning

1. `kanban-md board --json` — current state
2. `kanban-md list --status backlog,todo --sort priority -r --json` — candidates
3. Promote selected: `kanban-md move ID todo --json`
4. Assign: `kanban-md edit ID --assignee NAME --json`
5. Set deadlines: `kanban-md edit ID --due YYYY-MM-DD --json`

### Complete a Task

1. `kanban-md move ID done --json` — marks complete, sets Completed timestamp
2. `kanban-md show ID --json` — verify status and timestamps

### Track a Bug

1. `kanban-md create "Fix: DESCRIPTION" --priority high --tags bug --json`
2. `kanban-md edit ID --body "Steps to reproduce: ..." --json`

### Track a Dependency Chain

1. Create parent: `kanban-md create "Epic title" --json`
2. Create subtask: `kanban-md create "Subtask" --parent PARENT_ID --json`
3. Or add dependency: `kanban-md create "Task B" --depends-on TASK_A_ID --json`
4. List unresolved: `kanban-md list --blocked --json`

## Pitfalls

- **DO** pass `--json` on every command. Without it, output format depends on TTY and is not parseable.
- **DO** pass `--force` on delete. Without it, the command hangs waiting for stdin.
- **DO NOT** hardcode status or priority values. Read them from `kanban-md board --json`.
- **DO NOT** pipe table output through jq. Use `--json` instead.
- **DO NOT** use `--next` or `--prev` without checking current status. They fail at boundary statuses.
- **DO NOT** pass both `--status` and `--next`/`--prev` to move. Use one or the other.
- **DO** quote task titles with special characters: `kanban-md create "Fix: the 'login' bug" --json`.
