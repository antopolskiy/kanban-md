# Test Patterns

Read only the section relevant to the affected surface.

## CLI commands

- Unit tests: `cmd/*_test.go` exercise individual command logic and wiring.
- End-to-end tests: `e2e/*_test.go` run the built binary through complete user
  workflows.
- Shared E2E helpers such as `runKanban` and `runKanbanEnv` live in
  `e2e/helpers_test.go`.

Prefer an E2E reproducer when the defect depends on argument parsing, exit codes,
filesystem effects, environment variables, or the interaction of several
packages.

## TUI behavior

- Behavioral tests: `internal/tui/board_test.go` simulate messages and keypresses
  and inspect `View()`.
- Snapshot tests: `internal/tui/snapshot_test.go` compare rendered output with
  golden files under `internal/tui/testdata/`.
- `setupTestBoard(t)` creates four tasks across statuses at 120x40.
- `setupManyTasksBoard(t)` creates eighteen tasks at 100x30 for scrolling cases.
- `sendKey(b, "j")` and `sendSpecialKey(b, tea.KeyEsc)` simulate input.
- `assertGolden(t, "name", output)` compares rendered output with a golden file.
- `containsStr(haystack, needle)` checks text without ANSI sequences.
- `addLongBodyToTask(t, cfg, taskID, lineCount)` creates scrollable task bodies.

For size-specific defects, send
`tea.WindowSizeMsg{Width: W, Height: H}` through `Update`. Verify line counts,
content, and widths on the raw string returned by `View()`.

Update TUI snapshots with:

```bash
go test ./internal/tui -run TestSnapshot -update
```

Then inspect every changed golden file.

## Internal packages

- Follow the table-driven patterns in each package's `*_test.go` files.
- Use `t.TempDir()` for filesystem behavior.
- Use `strings.Builder` when testing output renderers.
- Prefer public behavior assertions over implementation-detail coupling.

Schema/frontmatter compatibility tests belong to the
`evolve-kanban-md-formats` workflow. Output renderer contracts also require the
`design-kanban-md-output` guidance.
