# Release CI OOM: wrapTitle preallocation

## Context

Release workflow for tag `v0.32.0` failed in CI (`release` workflow run `22256394404`) during race tests.

## Failure details

- Failing step: `make mod build test`
- Failing package: `internal/tui`
- Error: `fatal error: runtime: out of memory`
- Stack trace pointed to:
  - `internal/tui.wrapTitle` in `board.go`
  - called from `detailLines` via `wrapTitle(..., noLineLimit)`.

## Root cause

`wrapTitle` (and `wrapTitle2`) allocated output with:

- `make([]string, 0, maxLines)`

When `maxLines` was `noLineLimit` (`1<<31 - 1`), this attempted enormous preallocation under CI race-test memory constraints.

## Fix

1. Added bounded helper `wrapLinesCap(maxLines int)`.
2. Switched `wrapTitle` and `wrapTitle2` to use bounded capacity instead of `maxLines` directly.
3. Added tests:
   - `TestWrapLinesCap_NoLineLimit`
   - `TestWrapTitle_NoLineLimitWrapsWithoutHugeAllocation`

## Validation

- `go test -race ./internal/tui -run 'TestBoard_DetailTitleWraps|TestWrapTitle_NoLineLimitWrapsWithoutHugeAllocation|TestWrapLinesCap_NoLineLimit'`
- `go test ./...`

Both passed locally before retagging.
