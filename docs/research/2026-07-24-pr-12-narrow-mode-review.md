# PR #12 narrow-mode review

Date: 2026-07-24

PR: [antopolskiy/kanban-md#12](https://github.com/antopolskiy/kanban-md/pull/12)

Author: `@nkrebs13`

Submitted head: `7746b536338d5db5c38ec7d6d512a36d4f8b380c`

Submitted base: `3866c357ff888a4b593603189dc7d3cde3dce302`

## Verdict

Request changes before merge.

The narrow TUI implementation itself behaved well under the supplied suite, repeated targeted tests, race detection, fuzzing, PTY-level activation/navigation checks, cross-compilation, and a merge test against current `main`. Two configuration-integration omissions should be fixed before merge: the config schema version was not advanced, which allows an older v10 binary to silently delete the new field, and the project-wide `config` command does not expose the new key.

## Findings

### High: advance the config schema before persisting `tui.narrow_threshold`

The PR adds `TUIConfig.NarrowThreshold` at `internal/config/config.go:68`, but leaves `CurrentVersion` at 10 and adds no v10-to-v11 migration, v10 compatibility fixture, or compatibility assertion.

This is observable data loss, not only a process mismatch:

1. Start with a version 10 config containing `tui.narrow_threshold: 200`.
2. Run the pre-PR/current-main binary, which also considers version 10 current.
3. Execute any config write, such as:

   ```bash
   kanban-md config set board.description downgrade-save
   ```

4. The command succeeds and rewrites `config.yml`, but `narrow_threshold` disappears because the older binary accepted the same schema version and ignored the unknown field while decoding.

The repository compatibility contract requires a schema bump for config changes specifically to prevent older binaries from accepting and rewriting a newer schema. The fix should bump `CurrentVersion` to 11, register a v10-to-v11 migration that increments the version, copy/add the v10 fixture directory, and add the corresponding compatibility test. The migration can preserve the zero value because `0` already means automatic mode.

### Medium: expose the new key through `kanban-md config`

`addExtendedConfigAccessors` and `allConfigKeys` in `cmd/config.go` include every other TUI setting but omit `tui.narrow_threshold`. Reproduction against the PR binary:

```text
$ kanban-md config get tui.narrow_threshold
unknown config key "tui.narrow_threshold"
```

The key is also absent from table/JSON `kanban-md config` output, and `config set` cannot configure the documented option. Add a readable/writable integer accessor, include it in `allConfigKeys`, and add unit/E2E coverage for valid and invalid values.

## Verification performed

### GitHub CI

The PR's `build` workflow run `30058894167` passed on all three jobs:

- macOS: passed in 2m24s
- Ubuntu: passed in 2m40s
- Windows: passed in 7m03s

### Exact submitted head

- `make mod build test`: passed. This ran the unit and E2E suites with `-race` and merged coverage.
- `go vet ./...`: passed.
- `golangci-lint v2.10.1 run ./...`: passed with zero findings.
- `go test -run Compat ./internal/config/ ./internal/task/`: passed.
- Narrow, mouse-layout, and snapshot tests repeated 20 times: passed.
- Mouse-layout fuzz target for 20 seconds: passed after 103,402 executions.
- Linux amd64 CGO-disabled cross-build: passed.
- Windows amd64 CGO-disabled cross-build: passed.

### Integration with current local `main`

The PR merged cleanly into current local `main` (`8b3ff0d`), whose only delta from the PR base is a research document. `make mod build test` passed on the resulting integration commit with race detection enabled.

### Additional review probes

Temporary review-only tests (removed after execution) verified:

- `--narrow` forces the single-column layout at 120 columns through a real PTY.
- `tui.narrow_threshold: 200` activates narrow mode at 120 columns through a real PTY.
- Tab and Shift+Tab switch to the next and previous columns through the real TUI process.
- Rendered lines and mouse hit targets remain within the terminal for widths 5 through 120 and for first, middle, and last active columns.
- Width 4 does not panic, but the compact header can exceed the terminal width because the shared truncation helper enforces a four-cell minimum before style padding. This is an impractical boundary and was not treated as merge-blocking.

## Other observations

- `git diff --check` reports trailing spaces in `board_view_60.golden`. These are fixed-width terminal snapshot padding, not a logic defect; the snapshot comparison passes.
- No existing PR reviews, inline threads, or discussion comments were present when this review was performed.
