# kanban-md JSON Output Schemas

Reference for parsing JSON output from kanban-md commands.

## Task Object

Returned by: `list` (array), `show`, `create`, `edit`, `move` (with extra `changed` field).

```json
{
  "id": 1,
  "title": "Task title",
  "status": "in-progress",
  "priority": "high",
  "created": "2026-02-07T10:30:00Z",
  "updated": "2026-02-07T11:00:00Z",
  "started": "2026-02-07T10:35:00Z",
  "completed": "2026-02-07T12:00:00Z",
  "assignee": "alice",
  "tags": ["bug", "frontend"],
  "due": "2026-03-01",
  "estimate": "4h",
  "parent": 5,
  "depends_on": [3, 4],
  "blocked": true,
  "block_reason": "Waiting on API keys",
  "body": "Markdown body text",
  "file": "kanban/tasks/001-task-title.md"
}
```

Fields with `omitempty` (absent when zero/null): started, completed,
assignee, tags, due, estimate, parent, depends_on, blocked, block_reason,
body, file.

## Move Result

Embeds all task fields plus:

```json
{
  "changed": true
}
```

`changed` is `false` when the task was already at the target status
(idempotent move).

## Delete Result

```json
{
  "status": "deleted",
  "id": 1,
  "title": "Task title"
}
```

## Board Overview

Returned by: `board`.

```json
{
  "board_name": "My Project",
  "total_tasks": 12,
  "statuses": [
    {
      "status": "backlog",
      "count": 3,
      "blocked": 0,
      "overdue": 0
    },
    {
      "status": "in-progress",
      "count": 2,
      "wip_limit": 5,
      "blocked": 1,
      "overdue": 0
    }
  ],
  "priorities": [
    {"priority": "low", "count": 2},
    {"priority": "medium", "count": 5},
    {"priority": "high", "count": 4},
    {"priority": "critical", "count": 1}
  ]
}
```

`wip_limit` is omitted when 0 (unlimited).

## Metrics

Returned by: `metrics`.

```json
{
  "throughput_7d": 3,
  "throughput_30d": 12,
  "avg_lead_time_hours": 120.5,
  "avg_cycle_time_hours": 48.2,
  "flow_efficiency": 0.4,
  "aging_items": [
    {
      "id": 7,
      "title": "Implement search",
      "status": "in-progress",
      "age_hours": 72.5
    }
  ]
}
```

Omitempty fields: avg_lead_time_hours, avg_cycle_time_hours,
flow_efficiency, aging_items. Absent when no completed tasks exist.

## Log Entry

Returned by: `log` (array).

```json
{
  "timestamp": "2026-02-07T10:30:00Z",
  "action": "move",
  "task_id": 1,
  "detail": "backlog -> in-progress"
}
```

Action types: create, move, edit, delete, block, unblock.
For move actions, detail is `"old-status -> new-status"`.

## Init Result

```json
{
  "status": "initialized",
  "dir": "/absolute/path/to/kanban",
  "name": "My Project",
  "config": "/absolute/path/to/kanban/config.yml",
  "tasks": "/absolute/path/to/kanban/tasks",
  "columns": "backlog,todo,in-progress,review,done"
}
```

## Error Response

Returned on errors when `--json` is active:

```json
{
  "error": "task not found",
  "code": "TASK_NOT_FOUND",
  "details": {"id": 99}
}
```

Error codes: TASK_NOT_FOUND, BOARD_NOT_FOUND, BOARD_ALREADY_EXISTS,
INVALID_INPUT, INVALID_STATUS, INVALID_PRIORITY, INVALID_DATE,
INVALID_TASK_ID, WIP_LIMIT_EXCEEDED, DEPENDENCY_NOT_FOUND,
SELF_REFERENCE, NO_CHANGES, BOUNDARY_ERROR, STATUS_CONFLICT,
CONFIRMATION_REQUIRED, INTERNAL_ERROR.

Exit codes: 1 for user errors, 2 for internal errors.
