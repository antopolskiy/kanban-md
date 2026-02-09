# Guidelines

> **Note:** `CODEX.md` and `CLAUDE.md` are symlinks to `AGENTS.md`. Only edit `AGENTS.md` — both files are always in sync.

- Use semantic commit format, e.g.

```
feat: <short description>

Detailed description of the changes, that can be multiline, but do not artificially create line breaks, as commit messages are automatically wrapped.
- Use bullet points
- When needed
```

- When making changes that affect user-facing behavior (new commands, new flags, changed defaults, new installation methods, etc.), update `README.md` to reflect those changes.

- Whenever any research is done (including by the subagents), it should be documented as a report in docs/research/YYYY-MM-DD-<description>.md. In case of subagents, the need to be instructed to create the report file and on completion return only the path to the report, so that the main agent can read and analyze it.

## Releasing

To release a new version, tag and push. The release workflow triggers automatically on tag push — do NOT also run `gh workflow run release`, as that causes a duplicate build.

**IMPORTANT: Never create releases locally with `gh release create`.** Always let goreleaser handle release creation through the CI pipeline. After tagging and pushing, wait for the workflow to complete, then use `gh release edit` to update the release notes.

```
git tag vX.Y.Z
git push origin main --tags
```

### Release notes

After the release workflow completes, edit the GitHub release with human-written notes (not commit logs). Use `gh release edit` to update the release created by goreleaser.

**Process:**

1. Diff against the previous version to understand what changed:
   ```
   git diff vPREVIOUS..vNEW --stat
   git log vPREVIOUS..vNEW --oneline
   ```
2. Write release notes from the user's perspective — what can they do now that they couldn't before?
3. Publish with `gh release edit vX.Y.Z --title 'vX.Y.Z "Codename" — Short Theme' --notes "..."`.

**Format:**

```markdown
One to three sentence TL;DR of what this release is about and why users should care.

## New: Feature name

Brief explanation of what it does and why it matters.

\```bash
# 1-2 practical examples showing real usage
kanban-md <command> <flags>
\```

## New: Another feature

...

## Changed: Behavior change

Explain what changed and what users need to know. Include migration notes if applicable.

## Upgrading

Any steps users need to take, or "No action needed" with explanation of auto-migration.

**Full diff:** [`vPREVIOUS...vNEW`](https://github.com/antopolskiy/kanban-md/compare/vPREVIOUS...vNEW)
```

**Guidelines:**
- Title format: `vX.Y.Z "Codename" — Short Theme` (e.g. `v0.8.0 "Iron Gate" — Claim Enforcement`). The codename is a 1-2 word evocative nickname that creates a memorable association with the release — something vivid and fun that works as a mnemonic (e.g. "Quiet Storm", "Paper Trail", "Red Line"). It should loosely relate to the theme but doesn't need to be literal.
- Start with a short TL;DR paragraph (no heading) summarizing the release for someone skimming
- Use `## New:` for new commands/features, `## Changed:` for behavior changes, `## Fixed:` for bug fixes
- Every feature section should include a code example
- End with `## Upgrading` section and a full diff link
- Write for users, not developers — focus on what they can do, not what files changed

## Backward Compatibility

When modifying `config.yml` schema or task file frontmatter, you must ensure backward compatibility:

### Config design principles

- **Collocate column settings with column definitions.** Any configuration that is per-column (e.g. `show_duration`, `require_claim`, WIP limits) must live inside the status/column entry in the `statuses` list — never as a separate top-level list or map. This keeps related settings together and avoids drift between column names and their config.

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

## Output Format Design

The default output is **table**. TTY auto-detection was intentionally removed — agents run in non-TTY environments and were getting verbose JSON by default, wasting tokens and context window.

Three formats are available:
- **table** (default): Human-readable padded columns. Best for interactive terminal use.
- **compact** (`--compact`/`--oneline` or `KANBAN_OUTPUT=compact`): One-line-per-record format modeled after `git log --oneline`. ~70% fewer tokens than JSON. Designed for AI agent consumption.
- **json** (`--json` or `KANBAN_OUTPUT=json`): Full structured JSON. Use for scripting and piping to `jq`.

This is a deliberate design decision. Do not revert to TTY auto-detection without understanding the agent token cost implications. See `docs/research/2026-02-08-token-efficient-output-formats.md` for the research behind this choice.

## Fixing Bugs (Test-Driven Development)

When fixing bugs, use test-driven development: write a failing test that reproduces the bug first, then implement the fix.

### Process

1. **Read the bug description** and identify which package is affected (CLI command or TUI).
2. **Study existing test patterns** in the relevant package to match style and helpers.
3. **Write a failing test** that reproduces the exact bug behavior. Run it to confirm it fails.
4. **Implement the fix** in the production code.
5. **Run the failing test** to confirm it now passes.
6. **Run the full test suite** to check for regressions.
7. **Run lint** (`golangci-lint run ./...`) and fix any issues.
8. **Update golden files** if snapshot tests are affected (`go test ./... -run TestSnapshot -update`).

### Where to look for test patterns

**CLI bugs** (`cmd/` package):
- Unit tests: `cmd/*_test.go` — test individual command logic
- E2E tests: `e2e/cli_test.go` — run the compiled binary with `runKanban(t, ...)` helper
- Test helpers: `runKanban`, `runKanbanEnv`, `setupProject` in `e2e/cli_test.go`

**TUI bugs** (`internal/tui/` package):
- Behavioral tests: `internal/tui/board_test.go` — simulate keypresses and check View() output
- Snapshot tests: `internal/tui/snapshot_test.go` — compare View() output against golden files in `testdata/`
- Test helpers:
  - `setupTestBoard(t)` — creates a temp board with 4 tasks across statuses, 120x40 terminal
  - `setupManyTasksBoard(t)` — creates 18 tasks for scroll testing, 100x30 terminal
  - `sendKey(b, "j")` — simulates a keypress and returns the updated Board
  - `sendSpecialKey(b, tea.KeyEsc)` — simulates special keys (esc, enter, etc.)
  - `assertGolden(t, "name", output)` — compares against `testdata/name.golden`
  - `containsStr(haystack, needle)` — substring check without ANSI codes
  - `addLongBodyToTask(t, cfg, taskID, lineCount)` — modifies a task to have a multi-line body

**Internal packages** (`internal/output/`, `internal/task/`, etc.):
- Each package has `*_test.go` files with table-driven tests
- Use `strings.Builder` for testing output renderers
- Use `t.TempDir()` for file system tests

### Tips

- For TUI bugs, the key insight is that `View()` returns the raw string — check line counts, substring presence, and line widths to verify correct behavior.
- For bugs that only manifest at specific terminal sizes, use `b.Update(tea.WindowSizeMsg{Width: W, Height: H})` to set the size.
- When a fix changes golden file output, update them with `-update` flag: `go test ./internal/tui/ -run TestSnapshot -update`.
- Verify golden file contents look correct after updating — don't blindly accept changes.

## Using the Kanban Board

This project uses its own kanban board (in `kanban/`) to track work. Use the CLI to manage tasks as you work.

### Workflow

1. **Before starting work**, check the board:
   ```
   go run ./cmd/kanban-md list
   go run ./cmd/kanban-md list --status backlog
   ```

2. **When picking up a task**, move it to in-progress:
   ```
   go run ./cmd/kanban-md move <ID> in-progress
   ```

3. **When finishing a task**, move it to done:
   ```
   go run ./cmd/kanban-md move <ID> done
   ```

4. **When starting a new piece of work** that isn't already tracked, create a task:
   ```
   go run ./cmd/kanban-md create "Short descriptive title" --priority <low|medium|high|critical> --tags <layer-N>
   ```
   Then add a body with relevant context (reference to proposal docs, scope notes, etc.):
   ```
   go run ./cmd/kanban-md edit <ID> --body "Description of what needs to be done. See docs/plans/layer-N-xxx.md"
   ```

5. **When a task is blocked or no longer needed**, update accordingly:
   ```
   go run ./cmd/kanban-md move <ID> backlog
   go run ./cmd/kanban-md delete <ID>
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
  go run ./cmd/kanban-md create "Fix: <description of bug>" --priority high --tags bug
  ```
- **Ideas**: Usability improvements, missing features, or things that feel awkward.
  ```
  go run ./cmd/kanban-md create "<description of improvement>" --priority low --tags idea
  ```
- Always add a `--body` explaining what happened, what you expected, and (for bugs) how to reproduce.
- These tasks help us dogfood the tool and build a backlog of real-world improvements.
