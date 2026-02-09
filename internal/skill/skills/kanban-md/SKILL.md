---
name: kanban-md
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

!`kanban-md board 2>/dev/null || echo 'No board found — run: kanban-md init --name PROJECT_NAME'`

## Rules

- Use `--compact` for listing commands (`list`, `board`, `metrics`, `log`) to get
  token-efficient one-line output.
- Use `kanban-md show ID` (default table format) to read task details — it includes
  the full body and all fields in a human-readable layout. Only add `--json` when you
  need to pipe the output to another tool or parse fields programmatically.
- Always pass `--force` when deleting (`kanban-md delete ID --force`).
- Dates use `YYYY-MM-DD` format.
- Statuses and priorities are board-specific. Check the board state above or run
  `kanban-md board` to discover valid values before using them.
- Default statuses: backlog, todo, in-progress, review, done.
- Default priorities: low, medium, high, critical.

## Decision Tree

| User wants to...                        | Command                                                 |
|-----------------------------------------|---------------------------------------------------------|
| See board overview / standup            | `kanban-md board --compact`                             |
| List all tasks                          | `kanban-md list --compact`                              |
| List tasks by status                    | `kanban-md list --compact --status todo,in-progress`    |
| List tasks by priority                  | `kanban-md list --compact --priority high,critical`     |
| List tasks by assignee                  | `kanban-md list --compact --assignee alice`             |
| List tasks by tag                       | `kanban-md list --compact --tag bug`                    |
| List blocked tasks                      | `kanban-md list --compact --blocked`                    |
| List ready-to-start tasks               | `kanban-md list --compact --not-blocked --status todo`  |
| List tasks with resolved deps           | `kanban-md list --compact --unblocked`                  |
| Find a specific task                    | `kanban-md show ID`                                     |
| Create a task                           | `kanban-md create "TITLE" --priority P --tags T`        |
| Create a task with body                 | `kanban-md create "TITLE" --body "DESC"`                |
| Start working on a task                 | `kanban-md move ID in-progress`                         |
| Advance to next status                  | `kanban-md move ID --next`                              |
| Move a task back                        | `kanban-md move ID --prev`                              |
| Complete a task                         | `kanban-md move ID done`                                |
| Edit task fields                        | `kanban-md edit ID --title "NEW" --priority P`          |
| Add/remove tags                         | `kanban-md edit ID --add-tag T --remove-tag T`          |
| Set a due date                          | `kanban-md edit ID --due 2026-03-01`                    |
| Block a task                            | `kanban-md edit ID --block "REASON"`                    |
| Unblock a task                          | `kanban-md edit ID --unblock`                           |
| Add a dependency                        | `kanban-md edit ID --add-dep DEP_ID`                    |
| Set a parent task                       | `kanban-md edit ID --parent PARENT_ID`                  |
| Delete a task                           | `kanban-md delete ID --force`                           |
| See flow metrics                        | `kanban-md metrics --compact`                           |
| See activity log                        | `kanban-md log --compact --limit 20`                    |
| See recent activity for a task          | `kanban-md log --compact --task ID`                     |
| Initialize a new board                  | `kanban-md init --name "NAME"`                          |

## Core Commands

### list

```bash
kanban-md list [--status S] [--priority P] [--assignee A] [--tag T] \
  [--sort FIELD] [-r] [-n LIMIT] [--blocked] [--not-blocked] \
  [--parent ID] [--unblocked]
```

Sort fields: id, status, priority, created, updated, due. `-r` reverses.
`--unblocked` shows tasks whose dependencies are all at terminal status.

### create

```bash
kanban-md create "TITLE" [--status S] [--priority P] [--assignee A] \
  [--tags T1,T2] [--due YYYY-MM-DD] [--estimate E] [--body "TEXT"] \
  [--parent ID] [--depends-on ID1,ID2]
```

Prints the created task ID and summary.

### show

```bash
kanban-md show ID
kanban-md show ID --json   # only when piping to another tool
```

Default format shows all fields including the body in a readable layout.
Use `--json` only when you need to parse fields programmatically.
For the JSON schema, see [references/json-schemas.md](references/json-schemas.md).

### edit

```bash
kanban-md edit ID [--title T] [--status S] [--priority P] [--assignee A] \
  [--add-tag T] [--remove-tag T] [--due YYYY-MM-DD] [--clear-due] \
  [--estimate E] [--body "TEXT"] [--started YYYY-MM-DD] [--clear-started] \
  [--completed YYYY-MM-DD] [--clear-completed] [--parent ID] \
  [--clear-parent] [--add-dep ID] [--remove-dep ID] \
  [--block "REASON"] [--unblock]
```

Only specified fields are changed. Prints a confirmation message.

### move

```bash
kanban-md move ID STATUS
kanban-md move ID --next
kanban-md move ID --prev
```

Use `-f` to override WIP limits. Auto-sets Started on first move from
initial status. Auto-sets Completed on move to terminal status.

### delete

```bash
kanban-md delete ID --force
```

Always pass `--force` (non-interactive context requires it).

### board

```bash
kanban-md board
```

Shows board overview: task counts per status, WIP utilization,
blocked/overdue counts, priority distribution.

### metrics

```bash
kanban-md metrics [--since YYYY-MM-DD]
```

Shows throughput (7d/30d), avg lead/cycle time, flow efficiency,
aging items.

### log

```bash
kanban-md log [--since YYYY-MM-DD] [--limit N] [--action TYPE] \
  [--task ID]
```

Action types: create, move, edit, delete, block, unblock.

### Global Flags

All commands accept: `--json`, `--table`, `--compact` (alias `--oneline`), `--dir PATH`, `--no-color`.

## Workflows

### Daily Standup

1. `kanban-md board --compact` — board overview
2. `kanban-md list --compact --status in-progress` — in-flight work
3. `kanban-md list --compact --blocked` — stuck items
4. `kanban-md metrics --compact` — throughput and aging
5. Summarize: completed, active, blocked, aging items

### Triage New Work

1. `kanban-md list --compact --status backlog --sort priority -r` — review backlog
2. For items to promote: `kanban-md move ID todo`
3. For new items: `kanban-md create "TITLE" --priority P --tags T`
4. For stale items: `kanban-md delete ID --force`

### Sprint Planning

1. `kanban-md board --compact` — current state
2. `kanban-md list --compact --status backlog,todo --sort priority -r` — candidates
3. Promote selected: `kanban-md move ID todo`
4. Assign: `kanban-md edit ID --assignee NAME`
5. Set deadlines: `kanban-md edit ID --due YYYY-MM-DD`

### Complete a Task

1. `kanban-md move ID done` — marks complete, sets Completed timestamp
2. `kanban-md show ID --json` — verify status and timestamps

### Track a Bug

1. `kanban-md create "Fix: DESCRIPTION" --priority high --tags bug`
2. `kanban-md edit ID --body "Steps to reproduce: ..."`

### Track a Dependency Chain

1. Create parent: `kanban-md create "Epic title"`
2. Create subtask: `kanban-md create "Subtask" --parent PARENT_ID`
3. Or add dependency: `kanban-md create "Task B" --depends-on TASK_A_ID`
4. List unresolved: `kanban-md list --compact --blocked`

## Pitfalls

- **DO** use `--compact` for listing, board, metrics, and log commands — it is the most token-efficient format.
- **DO** use `kanban-md show ID` (default format) to read task details — it is readable and includes the full body.
- **DO** pass `--force` on delete. Without it, the command hangs waiting for stdin.
- **DO NOT** use `--json` unless you are piping output to another tool or parsing fields programmatically. Default and `--compact` formats are sufficient for reading.
- **DO NOT** hardcode status or priority values. Read them from `kanban-md board --compact`.
- **DO NOT** use `--next` or `--prev` without checking current status. They fail at boundary statuses.
- **DO NOT** pass both `--status` and `--next`/`--prev` to move. Use one or the other.
- **DO** quote task titles with special characters: `kanban-md create "Fix: the 'login' bug"`.
