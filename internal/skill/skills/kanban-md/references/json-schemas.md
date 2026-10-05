# kanban-md JSON Output Schemas

Reference for parsing `show --json` output and error responses.

## Task Object

Canonical task fields are returned by task commands when `--json` is passed.
`show --json` additionally always includes `children`, an array of direct child
records (empty when there are none). Its default child fields are `id`, `title`
and `status`.

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
  "claimed_by": "frost-maple",
  "claimed_at": "2026-02-07T10:35:00Z",
  "class": "standard",
  "body": "Markdown body text",
  "file": "kanban/tasks/001-task-title.md"
}
```

Fields with `omitempty` (absent when zero/null): started, completed,
assignee, tags, due, estimate, parent, depends_on, blocked, block_reason,
body, file.
Claim fields and class are also omitted when empty.

## Explicit property projection

Default task JSON excludes unknown frontmatter. Only `list` and `show` accept
repeatable `--show-property KEY`; it adds `properties` with just the requested
supported string, finite exact number, boolean or null values. For show, the
same selection also applies to each child record. For example, this fragment
from `show --json --show-property kind --show-property reading_order` omits
unrelated canonical fields for brevity:

```json
{
  "properties": {"kind": "milestone"},
  "children": [
    {"id": 2, "title": "Draft chapter", "status": "todo",
     "properties": {"kind": "chapter", "reading_order": 20}}
  ]
}
```

The object is present even when no requested value is available. Missing keys
are omitted; explicit null is included. Unsupported selected values are omitted
with a warning, never serialized as nested foreign data. Numbers are emitted
exactly; consumers should preserve number tokens instead of rounding via float64.
Display/group/child-sort configuration does not select task JSON values.
`list --group-by` produces aggregate records and rejects `--show-property`.

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
