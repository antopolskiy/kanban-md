# Robust testing strategy for TUI mouse support

## Conclusion

Mouse support can be tested to the same standard as the rest of kanban-md.

The application already has the key ingredients:

- model-level TUI tests using synthetic Bubble Tea messages;
- snapshot tests at several terminal sizes;
- a real PTY end-to-end harness for the compiled `kanban-md` binary;
- Linux, macOS, and Windows CI;
- race-enabled unit and E2E tests through `make test`;
- coverage collected from both unit tests and the E2E-instrumented binary.

A disposable proof test confirmed that the existing PTY approach can exercise the actual terminal input path. It successfully:

1. launched a Bubble Tea program with cell-motion mouse support;
2. observed the terminal-mode enable sequences;
3. wrote raw SGR mouse press, release, and wheel bytes into the PTY;
4. received normalized `tea.MouseMsg` values with the expected zero-based coordinates;
5. wrote legacy X10 press, release, and wheel bytes into the PTY;
6. received the expected X10 messages;
7. observed Bubble Tea disabling mouse modes at exit.

The proof command completed successfully:

```text
go test ./e2e -run 'TestTemporaryPTYMouseProtocols|TestMouseProbeHelper' -count=1 -v

--- PASS: TestTemporaryPTYMouseProtocols
```

The probe was removed after verification; only this report is retained.

The main testing requirement is architectural: hitbox calculation must be kept as a small, deterministic layout subsystem that is derived from the same render loop as the visible cards. It should not be scattered across event handlers.

## Important protocol edge: SGR vs X10 releases

Bubble Tea v1.3.10 normalizes both SGR and X10 coordinates to zero-based X/Y values, but release semantics differ:

- an SGR left-button release has `Action == MouseActionRelease` and `Button == MouseButtonLeft`;
- a legacy X10 release has `Action == MouseActionRelease` and `Button == MouseButtonNone`, because X10 does not identify the released button.

Therefore, the application should not implement click handling as:

```go
if msg.Action == tea.MouseActionRelease && msg.Button == tea.MouseButtonLeft {
    openTarget()
}
```

Use a click state machine instead:

1. On an unmodified left-button press inside a target, remember the target identity.
2. On motion outside that target, cancel the pending click.
3. On release, activate only if the pointer is still inside the same target.
4. Accept either `MouseButtonLeft` (SGR) or `MouseButtonNone` (X10) on the matching release.
5. Cancel pending state on resize, reload, view change, or a different button press.

This gives release-based activation without breaking the documented Bubble Tea fallback.

## Testability-oriented implementation shape

The layout code should produce rendered content and hitboxes together.

Suggested internal types:

```go
type mouseRect struct {
    minX int
    minY int
    maxX int
    maxY int
}

type mouseTarget struct {
    kind   mouseTargetKind
    rect   mouseRect
    taskID int
    col    int
    row    int
}

type renderedColumn struct {
    content string
    targets []mouseTarget
}
```

`renderColumn` should append a target at the same moment it appends the corresponding card. `viewBoard` then offsets each target by the column's X position and clamps it to the visible terminal viewport.

This avoids two independent implementations of card layout. If rendering changes card height, indicators, or clipping, the target geometry changes in the same code path.

The target snapshot may be stored on `Board` for the next mouse event. Bubble Tea serializes `Update` and `View`, so the event uses the geometry of the screen that the user actually saw.

Private layout tests can use `package tui` so production internals do not need test-only exported methods.

## Automated test layers

### 1. Pure geometry and hit-testing tests

These should contain the largest edge-case matrix because they are fast and deterministic.

Required invariants:

- every target is fully within the visible terminal bounds;
- fully clipped cards do not have targets;
- partially visible cards have targets clamped to their visible portion;
- targets do not overlap;
- one coordinate resolves to at most one target;
- card targets never include column headers, scroll indicators, blank separators, error lines, search/status bars, or unused terminal space;
- task IDs and column/row indexes match the rendered card;
- an empty column produces no card target;
- a layout with zero width or height does not panic;
- repeated layout with identical state produces identical targets.

Geometry cases:

- terminal widths below the minimum board width;
- terminal widths where `columnWidth()` reaches its minimum and maximum;
- right-side unused space when all columns have reached `maxColWidth`;
- terminal heights from 1 through the smallest usable board;
- a card partially clipped by the bottom viewport;
- error toast present, changing `chromeHeight`;
- up indicator only, down indicator only, and both indicators;
- one-line and multi-line task titles;
- mixed variable-height cards;
- blocked cards and claimed cards with additional content lines;
- `tui.title_lines` values across their supported range;
- hidden empty columns changing column count and X offsets;
- search filters changing visible tasks;
- sort changes moving the same task to a different row;
- Unicode titles containing CJK, emoji, and combining characters;
- reload after a title change alters card height;
- archived or deleted tasks disappearing from the target snapshot.

A deterministic property test should iterate across a grid of widths, heights, title-line limits, scroll offsets, and generated title lengths. A Go fuzz target can additionally seed these cases and assert the invariants above.

### 2. Mouse state-machine/model tests

Use synthetic `tea.MouseMsg` values against `Board.Update`.

Click tests:

- SGR-style left press plus left release on the same card opens it;
- X10-style left press plus buttonless release on the same card opens it;
- release without a preceding press is ignored;
- press outside and release inside is ignored;
- press inside and release outside is ignored;
- press one card and release another is ignored;
- left-button motion leaving the card cancels the click;
- right and middle buttons are ignored;
- modified clicks are ignored;
- double-click does not accidentally perform an action in the next view;
- clicking a non-active column updates selection before opening detail;
- returning from detail preserves the clicked task as keyboard selection;
- clicks on card borders follow the documented inclusive/exclusive boundary;
- clicks on headers, indicators, empty placeholders, status bar, and unused space do nothing.

State invalidation tests:

- resize between press and release cancels the click;
- task reload between press and release cancels it;
- entering help/search/create/edit/move/delete/debug cancels it;
- the pressed task being deleted or filtered out cancels it;
- a new render cannot activate a stale target from the previous layout.

View-routing tests:

- card clicks are accepted only in `viewBoard`;
- Back clicks are accepted only in `viewDetail`;
- all mouse actions are ignored in dialogs and text-entry views for the first release;
- existing keyboard controls work after ignored mouse events.

### 3. Wheel behavior tests

Board wheel tests:

- wheel down scrolls the column under the pointer, not merely the active column;
- wheel over a different column does not change the first column;
- wheel over the status bar or unused right-side space is ignored;
- wheel on an empty column is a no-op;
- repeated wheel up at the top clamps cleanly;
- repeated wheel down at the bottom clamps cleanly;
- trackpad-like bursts of many events do not overshoot or panic;
- variable-height cards remain aligned with hitboxes after every event;
- scrolling an inactive column does not corrupt the active column's selection;
- clicking a newly revealed card opens the correct task;
- hidden-empty-column mode uses the new column geometry;
- resize after scrolling recalculates visible ranges and hitboxes.

Detail wheel tests:

- wheel down advances by the chosen line step;
- wheel up returns toward the top;
- short details remain unchanged;
- long Markdown-rendered details clamp at both ends;
- repeated bursts clamp immediately;
- wheel over the Back action follows the explicitly chosen behavior;
- clicking Back after scrolling returns to the same selected task.

### 4. Snapshot tests

Mouse input itself does not need a snapshot, but user-visible affordances do:

- help view documents `--mouse` behavior and keyboard fallback;
- detail footer contains a visible Back action;
- status/help wording explains wheel support;
- narrow terminal snapshots show a non-broken Back affordance;
- snapshots remain unchanged when mouse support is disabled, except for intentionally documented text.

Golden tests should not contain hidden hitbox markers or control sequences.

### 5. Command tests

Test flag behavior independently from the TUI model:

- `kanban-md tui --mouse` is accepted;
- default launch leaves mouse support disabled;
- the option passed to Bubble Tea is selected only when requested;
- other TUI flags still compose with `--mouse`;
- program initialization errors are unchanged.

The flag should remain a local runtime option, so no config migration or compatibility fixture is needed.

## Required PTY end-to-end tests

The existing harness launches the compiled binary in a real pseudo-terminal. Mouse helpers can write the same byte sequences a terminal would send.

Add these harness capabilities:

- launch the TUI with additional command arguments and configurable rows/columns;
- expose raw output for terminal-mode assertions;
- create an output checkpoint and wait only for output produced after that checkpoint;
- send arbitrary raw bytes;
- send SGR press/release/wheel events using zero-based helper coordinates;
- send legacy X10 press/release/wheel events;
- resize the PTY with `pty.Setsize`;
- avoid fixed sleeps where a rendered-state wait is possible.

The checkpoint is important. The current output buffer accumulates all previous frames, so a plain substring search can pass against stale content. Mouse E2E tests should assert positive, unique content emitted after the action.

### Primary journey 1: click task, inspect, and return

`TestE2E_TUI_MouseClickDetailAndBack`

1. Launch the compiled binary with `tui --mouse`.
2. Assert raw output contains cell-motion and SGR enable sequences.
3. Click a task in a non-active column, proving both X and Y mapping.
4. Wait after an output checkpoint for that task's unique detail heading.
5. Click the visible Back action.
6. Wait after a new checkpoint for the board status bar.
7. Press Enter using the keyboard and confirm the same clicked task opens, proving selection synchronization.
8. Quit normally.
9. Assert raw output contains mouse-mode disable sequences.

Run the click/Back portion as protocol subtests:

- SGR release retaining `MouseButtonLeft`;
- X10 release using `MouseButtonNone`.

This one journey verifies CLI option wiring, Bubble Tea program options, raw parser integration, hitbox lookup, view transition, selection state, Back handling, protocol fallback, keyboard interoperability, and cleanup.

### Primary journey 2: variable-height scrolling and detail scrolling

`TestE2E_TUI_MouseWheelVariableHeightBoardAndDetail`

1. Seed one column with many tasks whose titles create mixed card heights.
2. Give a task near the bottom a long body with unique numbered marker lines.
3. Launch at a deliberately small PTY height.
4. Send multiple wheel-down events over that column.
5. Click the screen location occupied by a newly revealed card.
6. Confirm the detail heading identifies the expected task.
7. Wheel down in detail until a later unique body marker is rendered.
8. Send excessive wheel-down events and verify the view remains responsive at the bottom.
9. Wheel upward and verify an earlier marker is rendered.
10. Click Back and quit.

This journey verifies hovered-column routing, variable card heights, scroll indicators, hitbox regeneration, wheel bursts, detail clamping, and post-scroll Back behavior.

### Additional focused PTY tests

These should remain small:

- default `tui` emits no mouse-mode enable sequence;
- `tui --mouse` remains fully keyboard-operable when no mouse messages arrive;
- resize the PTY after initial render, then click using the new geometry;
- click header/status-bar/blank coordinates and verify no view transition;
- mode-disable sequences are emitted on `q`, `Esc`, and `Ctrl+C` exits where practical.

## CI and platform boundary

The current CI matrix runs Ubuntu, macOS, and Windows.

The PTY TUI suite is already guarded with `//go:build !windows`, so raw ANSI mouse E2E tests will run on Ubuntu and macOS. That matches the existing standard for TUI keyboard E2E coverage.

Windows still runs:

- all pure geometry tests;
- all synthetic `tea.MouseMsg` model tests;
- command flag tests;
- snapshot tests;
- the rest of the non-PTY suite.

Bubble Tea owns Windows console-event translation. Recreating an interactive Windows console inside GitHub Actions would be brittle and would mostly duplicate framework testing. A Windows Terminal smoke test is appropriate before release, but the application's routing and geometry remain automated cross-platform.

## What generic CI cannot prove

The following depend on a real terminal emulator or multiplexer and should be a short manual smoke matrix:

- whether a terminal supports mouse reporting at all;
- how that terminal bypasses mouse capture for native text selection;
- tmux/Zellij forwarding configuration;
- trackpad acceleration and subjective wheel feel;
- Windows Terminal console behavior;
- terminal-specific modifier behavior.

This is not a large untested application surface. If no mouse messages arrive, the application simply receives no mouse input; the keyboard fallback is automated. Terminal selection is performed by the terminal before the application sees the event and cannot be portably asserted from a PTY.

Suggested pre-release smoke matrix:

- macOS terminal;
- Linux terminal;
- one tmux or Zellij session;
- Windows Terminal;
- mouse disabled/default mode;
- `--mouse` mode;
- native text-selection bypass and documented fallback.

## Coverage and merge gate

Current local baseline:

```text
go test -cover ./internal/tui
coverage: 93.3% of statements
```

The existing compiled-binary TUI E2E suite also passed locally:

```text
go test ./e2e -run 'TestE2E_TUI' -count=1
ok .../e2e 26.153s
```

Acceptance criteria for the mouse feature:

- all new geometry and mouse-dispatch branches are directly tested;
- `internal/tui` package coverage does not fall below the existing baseline;
- both primary PTY journeys pass on Linux and macOS;
- SGR and X10 click semantics are covered;
- keyboard-only E2E tests remain unchanged and green;
- `make test` passes with the race detector and merged unit/E2E coverage;
- `golangci-lint run ./...` passes;
- targeted mouse E2E tests pass repeatedly with `-count=10`;
- manual terminal matrix is recorded in the PR or research report.

With these conditions, mouse support would be as well tested as other significant TUI behavior in the project. The only manual remainder is terminal-owned compatibility, not application logic.
