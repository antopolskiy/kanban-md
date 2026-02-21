# TUI edit flow parity with create (Task #184)

## Context

Task #184 asked for a TUI edit workflow that uses the same step-based interaction as TUI create.

## What was reviewed

- `internal/tui/board.go`
  - Existing `viewCreate` wizard had 4 steps (Title, Body, Priority, Tags).
  - Board shortcuts did not include any edit action.
  - Status bar/help text documented create/move/delete, but not edit.
- `internal/tui/create_test.go`
  - Verified the expected wizard behavior and hint conventions.
- `internal/tui/snapshot_test.go` + golden files
  - Confirmed status/help text is snapshot-covered and must be updated when shortcuts change.
- `README.md` TUI keyboard shortcut section
  - Found mismatch with actual keys (`M` documented, while implementation uses `n`/`p`) and missing create/edit parity details.

## Key decisions

1. Reuse the existing create wizard state and rendering path for edit mode instead of introducing a separate edit dialog system.
2. Add `e` as the board-level shortcut for editing the currently selected task.
3. Keep the same 4-step flow and navigation semantics; only the Enter action label changes from `create` to `save` in edit mode.
4. Prefill edit fields from the selected task and persist via task write + filename rename when title changes, matching CLI edit behavior.
5. Update snapshots and README to reflect the new keyboard behavior.

## Validation strategy

- Added focused tests in `internal/tui/edit_test.go` for:
  - Opening/canceling edit dialog
  - Status bar hint
  - Prefill behavior
  - Save from title step (including filename rename)
  - Full wizard save of title/body/priority/tags
- Ran:
  - `go test ./internal/tui -run TestEdit_`
  - `go test ./internal/tui -run TestSnapshot -update`
  - `go test ./internal/tui`
  - `go test ./...`
- Lint command availability checked; `golangci-lint` binary is not installed in this environment.
