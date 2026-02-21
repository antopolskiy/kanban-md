# TUI hide-empty-columns: config + command override (Task #185)

## Goal

Implement a TUI option to automatically hide empty status columns, with:

- a persistent config setting, and
- a `kanban-md tui` command-line override.

## Findings from codebase review

- TUI columns are currently always built from `cfg.BoardStatuses()` in `internal/tui/board.go`.
- `cmd/tui.go` had no TUI-specific runtime flags.
- TUI config currently includes `title_lines` and `age_thresholds` under `tui`.
- Config schema version was `9`, with migrations up to `migrateV8ToV9`.
- Compat fixtures existed up to `internal/config/testdata/compat/v8/`.

## Design decisions

1. Add `tui.hide_empty_columns` (`bool`) to config schema.
2. Bump config version to `10` and add `migrateV9ToV10` (version bump migration).
3. Add `compat/v9` fixture and v9 compatibility tests.
4. Add `kanban-md tui` flags:
   - `--hide-empty-columns`
   - `--show-empty-columns`
   with conflict validation and CLI precedence over config.
5. Keep empty-board usability by falling back to rendering all columns when every column is empty.

## Validation coverage added

- Config key exposure and set/get wiring (`cmd/config*`, `e2e/config_test.go`).
- TUI flag resolution tests (`cmd/tui_test.go`).
- TUI behavior tests for hiding empty columns and empty-board fallback (`internal/tui/board_test.go`).
- Config migration + compat (`internal/config/migrate_test.go`, `internal/config/compat_test.go`, `compat/v9` fixture).

## Commands executed

- `go test -run Compat ./internal/config/ ./internal/task/`
- `go test ./...`
- `golangci-lint run ./...` (binary not available in this shell)
