# kanban-md JSON Output Schemas

Reference for parsing `show --json` output and error responses.

## Task Object

Returned by: `show --json` (also by other commands when `--json` is passed).

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
