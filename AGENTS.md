# Guidelines

- Use semantic commit format, e.g.

```
feat: <short description>

Detailed description of the changes, that can be multiline, but do not artificially create line breaks, as commit messages are automatically wrapped.
- Use bullet points
- When needed
```

- When making changes that affect user-facing behavior (new commands, new flags, changed defaults, new installation methods, etc.), update `README.md` to reflect those changes.

## Backward Compatibility

When modifying `config.yml` schema or task file frontmatter, you must ensure backward compatibility:

### Config changes

1. **Bump `CurrentVersion`** in `internal/config/defaults.go`.
2. **Add a migration function** in `internal/config/migrate.go` that transforms the old format to the new one. Register it in the `migrations` map. The migration must increment `cfg.Version`.
3. **Create a new fixture directory** at `internal/config/testdata/compat/vN/` (where N is the OLD version number) with a representative `config.yml` and sample task files. Copy from the previous version's fixtures before making changes.
4. **Add a compat test** in `internal/config/compat_test.go` that loads the old fixture with the current code and verifies all fields parse correctly after migration.

### Task file changes

- **Adding new optional fields** with `omitempty` is always safe -- old files without the field parse correctly (zero value).
- **Never rename or remove existing fields.** If a field must change, keep the old YAML tag and add the new one, or handle it in a migration.
- When adding new task fields, add a fixture file in `internal/task/testdata/compat/v1/tasks/` that exercises the field (or create a new version directory if the format changes).
- Add a compat test in `internal/task/compat_test.go` verifying the field parses correctly from fixtures.

### Testing

- Run `go test -run Compat ./internal/config/ ./internal/task/` to verify backward compatibility.
- Compat tests must pass for ALL previous fixture versions, not just the latest.

## Using the Kanban Board

This project uses its own kanban board (in `kanban/`) to track work. Use the CLI to manage tasks as you work.

### Workflow

1. **Before starting work**, check the board:
   ```
   go run . list
   go run . list --status backlog
   ```

2. **When picking up a task**, move it to in-progress:
   ```
   go run . move <ID> in-progress
   ```

3. **When finishing a task**, move it to done:
   ```
   go run . move <ID> done
   ```

4. **When starting a new piece of work** that isn't already tracked, create a task:
   ```
   go run . create "Short descriptive title" --priority <low|medium|high|critical> --tags <layer-N>
   ```
   Then add a body with relevant context (reference to proposal docs, scope notes, etc.):
   ```
   go run . edit <ID> --body "Description of what needs to be done. See docs/plans/layer-N-xxx.md"
   ```

5. **When a task is blocked or no longer needed**, update accordingly:
   ```
   go run . move <ID> backlog
   go run . delete <ID>
   ```

### Conventions

- **Tags**: Use `layer-N` tags (e.g. `layer-3`, `layer-4`) to group tasks by roadmap layer.
- **Priorities**: Use `high` for tasks the user explicitly requested, `medium` for planned work, `low` for nice-to-haves.
- **Statuses**: `backlog` → `todo` → `in-progress` → `review` → `done`. Skip statuses when appropriate (e.g. `backlog` → `in-progress` is fine).
- **Task titles**: Short, imperative form (e.g. "Add WIP limits per column", not "WIP limits were added").
- Keep the board current -- move tasks as you work, don't let them go stale.

### Tracking Issues and Ideas

When you encounter problems or have ideas while using the tool, create tasks for them:

- **Bugs**: Something that doesn't work as expected.
  ```
  go run . create "Fix: <description of bug>" --priority high --tags bug
  ```
- **Ideas**: Usability improvements, missing features, or things that feel awkward.
  ```
  go run . create "<description of improvement>" --priority low --tags idea
  ```
- Always add a `--body` explaining what happened, what you expected, and (for bugs) how to reproduce.
- These tasks help us dogfood the tool and build a backlog of real-world improvements.

