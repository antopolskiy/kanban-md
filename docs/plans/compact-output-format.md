# Plan: Add Compact Output Format + Revert to Table Default

## Context

When Claude Code runs `kanban-md list`, stdout is non-TTY so the CLI returns pretty-printed JSON — verbose, token-expensive, and unnecessary for agent consumption. Research shows a one-line compact format (`git log --oneline` style) uses ~70% fewer tokens than JSON and ~40% fewer than padded tables. We want to:

1. **Revert TTY auto-detection** — table is the unconditional default
2. **Add `FormatCompact`** — `--compact`/`--oneline` flags + `KANBAN_OUTPUT=compact` env var
3. **Update all docs** so agents use compact and this decision doesn't get reversed

## Setup

- Create a git worktree on a feature branch (e.g., `feat/compact-output`)
- All code/doc edits happen in the worktree
- All kanban board operations (`kanban-md create/move/edit`) run from `/Users/santop/Projects/kanban-md` (main directory)

## Steps

### 1. Core: `internal/output/output.go`

- Add `FormatCompact Format = 3` constant
- Remove `isTerminalFn` variable and `golang.org/x/term` import
- Change `Detect(jsonFlag, tableFlag bool)` → `Detect(jsonFlag, tableFlag, compactFlag bool)`
- Add `"compact"` and `"oneline"` cases to `KANBAN_OUTPUT` switch
- Remove TTY fallback — default returns `FormatTable`

### 2. Unit tests: `internal/output/output_test.go`

- Update all `Detect()` calls to 3 args
- Remove `TestDetectDefaultTTY` and `TestDetectDefaultPiped`
- Add: `TestDetectDefaultIsTable`, `TestDetectCompactFlag`, `TestDetectEnvCompact`, `TestDetectEnvOneline`, `TestDetectJSONFlagOverridesCompact`

### 3. New file: `internal/output/compact.go`

Five compact renderer functions:

| Function | Format |
|----------|--------|
| `TaskCompact(w, tasks)` | `#3 [backlog/high] Title @alice (tag1, tag2) due:2026-03-01` |
| `TaskDetailCompact(w, task)` | Same header + timestamps/body on indented lines |
| `OverviewCompact(w, overview)` | `Board (N tasks)` + `  status: count` lines |
| `MetricsCompact(w, metrics)` | `Throughput: 3/7d 12/30d \| Lead: 2d 0h \| ...` |
| `ActivityLogCompact(w, entries)` | `2026-02-08 12:00:05 create #1 Detail` |

Rules: no padding, no headers, omit empty optional fields.

### 4. New file: `internal/output/compact_test.go`

~16 tests covering all 5 renderers — all fields present, optional fields omitted, empty inputs, multi-record output. Follow `table_test.go` patterns (strings.Builder, fixed times, DisableColor not needed since compact has no styling).

### 5. Flags: `cmd/root.go`

- Add `flagCompact bool` global var
- Register `--compact` and `--oneline` (both bind to `flagCompact`)
- Update `outputFormat()` → `output.Detect(flagJSON, flagTable, flagCompact)`

### 6. Commands: add compact branch to 5 read commands

Each gets this pattern inserted between JSON and table branches:
```go
if format == output.FormatCompact {
    output.TaskCompact(os.Stdout, tasks)  // or appropriate compact renderer
    return nil
}
```

Files: `cmd/list.go`, `cmd/show.go`, `cmd/board.go`, `cmd/metrics.go`, `cmd/log.go`

Mutation commands (`create`, `edit`, `move`, `delete`, `init`, `config`, `context`) — no changes needed, they already use `Messagef` for non-JSON output and compact falls through correctly.

### 7. E2E tests: `e2e/cli_test.go`

- **Update** `TestPipedOutputDefaultsToJSON` → `TestDefaultOutputIsTable` (bare command now returns table, not JSON)
- **Add**: `TestCompactOutputList`, `TestCompactOutputShow`, `TestCompactOutputBoard`, `TestCompactOutputMetrics`, `TestCompactOutputLog`, `TestOnelineAlias`, `TestCompactEnvVar`

### 8. Documentation updates

**README.md** (3 locations):
- Line 17: Update "Built for automation" bullet — mention compact format
- Lines 384-402: Rewrite "Output format" section — table default, compact for agents, `--json` for scripting
- Line 460: Update "Predictable output" design principle

**SKILL.md** (agent instructions):
- Rules: recommend `--compact` instead of bare commands
- Global Flags: add `--compact`/`--oneline`
- Pitfalls: recommend compact, deprecate bare `--json` guidance
- Decision tree: consider adding `--compact` to listing commands

**CLAUDE.md**:
- Add "Output Format Design" section documenting the decision and rationale, so future changes don't blindly revert

### 9. Final cleanup

- `go mod tidy` (term still used in `cmd/delete.go`, so it stays)
- `go test ./...`
- `golangci-lint run ./...`

## Verification

1. `go test ./internal/output/...` — unit tests pass
2. `go test ./e2e/...` — e2e tests pass
3. `golangci-lint run ./...` — zero lint issues
4. Manual check: `go run . list` shows table, `go run . list --compact` shows one-line format, `go run . list --json` shows JSON, `KANBAN_OUTPUT=compact go run . list` shows compact

## Key files

- `internal/output/output.go` — format detection
- `internal/output/compact.go` — NEW, compact renderers
- `internal/output/compact_test.go` — NEW, compact tests
- `internal/output/output_test.go` — detection tests
- `cmd/root.go` — flag registration
- `cmd/list.go`, `cmd/show.go`, `cmd/board.go`, `cmd/metrics.go`, `cmd/log.go` — format branching
- `e2e/cli_test.go` — integration tests
- `README.md`, `SKILL.md`, `CLAUDE.md` — documentation
