# PR #12 exploratory layout and interaction testing

Date: 2026-07-24  
PR: `antopolskiy/kanban-md#12`  
Commit tested: `e271f89bbe70e2d7e02625863b0b06bb969041c2`  
Task: `#235`  
Agent: `curarine-pulsojet`

## Purpose

Explore the final PR interactively before adding more tests. The emphasis was
on combinations that the existing model-level narrow tests do not exercise:

- real PTY rendering and input parsing;
- live terminal resize;
- configured, automatic, and forced narrow modes;
- `hide_empty_columns` with live reload and search;
- keyboard, mouse, CRUD, detail, and modal interactions;
- Unicode labels and titles;
- many columns and double-digit position indicators;
- very small width and height boundaries.

No production or test code was changed during this pass.

## Existing coverage assessment

The PR has substantial `internal/tui` coverage for:

- automatic/configured/forced narrow selection;
- active-column header rendering;
- keyboard and model-level mouse tab switching;
- compact-bar navigation at widths 1–14;
- display-cell-safe Unicode truncation;
- card hit-target offsets;
- scroll calculations;
- snapshots at representative narrow widths;
- mouse-layout fuzzing.

The E2E PTY suite currently has no narrow-specific test. Its TUI scenarios
start with the default 120-column terminal, which renders the five-column
fixture in wide mode. Config E2E tests prove that `tui.narrow_threshold` can be
stored and read, but not that a real TUI process applies it.

This leaves the integration boundaries under-tested:

1. Cobra/config to Bubble Tea model wiring;
2. PTY resize and narrow/wide transitions;
3. terminal mouse coordinates against the rendered narrow strip;
4. CRUD and modal views inside a narrow viewport;
5. dynamic column sets caused by `hide_empty_columns`, search, and watcher
   reloads.

## Manual test setup

I built the PR binary and launched it in real PTYs with `TERM=dumb` and
`NO_COLOR=1`. I changed PTY dimensions with `stty` plus `SIGWINCH`, sent
keyboard input directly, and sent SGR mouse press/release sequences.

Two temporary boards were used:

- A five-column board with long titles, Unicode (`🧪`, CJK), empty columns,
  tasks in multiple statuses, and `hide_empty_columns`.
- A twelve-column board using Unicode status names (`α一` through `μ十二`) with
  one task per column.

Widths exercised:

`200, 120, 80, 60, 50, 48, 40, 14, 8, 7, 4`

Heights exercised:

`24, 12, 4, 3, 2, 1`

## Behaviors that worked

### Real narrow navigation

- Tab and Shift+Tab changed columns correctly.
- `h/l` navigation remained consistent.
- Detail opened for the selected task and returned to the board.
- Search filtered tasks and could be committed or cleared.
- At width 14, SGR clicks on both compact-bar arrows changed to the indicated
  column.
- Live resize between compact and abbreviated tab strips preserved a usable
  board.

### Unicode and many-column compact rendering

The twelve-column Unicode board rendered correctly:

```text
  α...  ▸ 1/12
◂ κ... ▸ 10/12
◂ λ... ▸ 11/12
◂ μ...   12/12
```

At width 7 the middle compact control rendered as:

```text
◂11/12▸
```

The next arrow remained clickable. CJK active status names were preserved when
they fit without metadata.

### Configuration threshold

With `tui.narrow_threshold: 200`, a real 120-column TUI rendered narrow. After
resizing to exactly 200 columns it rendered wide, confirming the intended
strict `< threshold` boundary and CLI-to-model wiring.

### Narrow CRUD smoke test

At 48x24:

- creating a task succeeded;
- editing opened the correct task;
- the move picker moved the task and persisted the new status;
- detail and cancellation paths remained responsive.

### Stress results

```text
go test ./internal/tui \
  -run 'TestNarrow_|TestCompactNarrowHeader_|HideEmpty|Search|Create' \
  -shuffle=on -count=50
PASS

go test ./internal/tui \
  -run '^FuzzMouseLayout$' \
  -fuzz='^FuzzMouseLayout$' \
  -fuzztime=20s
PASS — 153,437 executions
```

## Findings

### 1. Active column identity is lost when hidden columns change

Severity: high-value regression target; moderate user impact.

With `hide_empty_columns: true`, `loadTasks` rebuilds `b.columns` and preserves
only the numeric `activeCol`. It does not preserve the active status name.

#### Reproduction

1. Create non-empty `backlog`, `todo`, `in-progress`, and `done` columns.
2. Start a 50-column TUI with `hide_empty_columns: true`.
3. Select `todo` (visible index 1).
4. Externally move the last backlog tasks to `done`.
5. Wait for the watcher reload.

Observed:

- `backlog` disappears;
- the visible-column count changes from four to three;
- automatic layout changes from narrow (`50 < 4*14`) to wide
  (`50 >= 3*14`);
- active index 1 now identifies `in-progress`;
- Enter opens the in-progress task, even though the user had selected `todo`.

The hidden debug view confirmed:

```text
Active col: 1
Columns:    3
Column:     in-progress
Selected:   #4 In progress claimed
```

Adding `backlog` again before the active column caused the inverse jump:
`in-progress` changed to `todo` because index 1 was retained.

The index-preservation behavior predates this PR, but automatic single-column
rendering makes it materially more disruptive: in narrow mode the previous
status disappears entirely, and the same reload can also switch layout modes.

#### Test to add

Add a PTY E2E test:

`TestE2E_TUI_NarrowHideEmptyReloadPreservesActiveStatus`

- Launch at 50x24 with `hide_empty_columns`.
- Tab from backlog to todo.
- Mutate the task files through the CLI so backlog becomes empty.
- Wait for settled watcher output.
- Press Enter.
- Assert detail opens `Task #3: Todo first`, not the in-progress task.

Also add a smaller model test that calls `ReloadMsg` and asserts that a status
which still exists remains active even when preceding statuses are removed or
inserted.

This test should fail on the current PR.

### 2. A no-result search can reveal hidden columns and change layout mode

Severity: behavior decision needed; likely medium UX issue.

`hide_empty_columns` counts tasks after applying the search filter. When the
filter has zero matches, `filtered` is empty and the all-columns fallback is
used. That fallback is intended to keep a truly empty board creatable, but it
also runs for a non-empty board whose search has no matches.

#### Reproduction

1. Enable `hide_empty_columns`.
2. Use a board with four non-empty visible statuses and one empty status.
3. Open the TUI at width 60.
4. Confirm the four-column board is wide (`60 >= 4*14`).
5. Search for `ZZZ`, which matches no task.

Observed:

```text
backlog 0  todo 0  in-progress 0  review 0  done 0
```

All five columns reappear and the board changes to narrow because
`60 < 5*14`. Clearing the search returns to four columns and wide mode.

Filtering to a single result can also collapse `activeCol` to zero; clearing
the filter then treats index zero as the first status rather than restoring
the status selected before search.

#### Tests to add

First decide the intended behavior:

- Preferred: a no-result search should retain the pre-search visible status
  set and active status, showing zero matching cards without a layout-mode
  switch.
- Alternative: all columns may reappear, but the behavior should be explicit
  and the previously active status should be restored when search is cleared.

Then add:

- a model test for no-result search with hidden empty columns;
- a PTY E2E test that searches from a non-first active status, clears the
  search, presses Enter, and verifies selection/status restoration.

### 3. Modal views do not fit a practical narrow terminal

Severity: moderate usability issue; mostly pre-existing layout debt exposed by
the new narrow workflow.

At 48x24, the main narrow board and CRUD operations are functional, but several
modal views exceed the viewport:

- Create/edit body step: the long hint determines dialog width, clipping the
  right border and the end of `esc:cancel`.
- Create/edit priority step: its hint is even longer.
- Delete confirmation: a long task title is neither wrapped nor truncated, so
  the right border and title tail are clipped.
- Mouse help: the golden output is 31 lines, so a common 24-row terminal hides
  the top seven rows, including the heading and primary navigation entries.
- At width 14, create and move dialogs remain operable but borders, labels, and
  hints are heavily clipped.

Example at 48 columns:

```text
╭───────────────────────────────────────────────
│  Create task in todo  Step 2/4: Body
...
│  tab:next  shift+tab:back  enter:create  esc:c
╰───────────────────────────────────────────────
```

Long-title delete confirmation:

```text
│    #2: A very long backlog title intended to w
```

#### Tests to add

Unit/snapshot tests are more reliable than the current raw PTY buffer for exact
modal geometry:

`TestDialogViewsFitViewport`

For widths `48, 40, 20, 14` and heights `24, 12`:

- open every create/edit wizard step;
- open move and delete dialogs with long Unicode content;
- open normal and mouse help;
- assert every rendered line is at most `b.width` display cells;
- assert the rendered view is at most `b.height` lines, or that an explicit
  scroll/continuation affordance is present;
- assert critical actions (`enter`, `esc`, confirmation keys) remain visible.

The current implementation will fail these invariants.

After fixing the model layout, add one PTY smoke test at 48x24 to ensure the
body step remains interactive through a real terminal.

## Recommended E2E additions

### Tier 1: highest value

1. `TestE2E_TUI_NarrowHideEmptyReloadPreservesActiveStatus`
   - Reproduces finding 1 and should fail now.

2. `TestE2E_TUI_ConfiguredNarrowThresholdAndResizeBoundary`
   - Set threshold 200.
   - Start at 120 and assert only the active column's task is rendered.
   - Resize to 200 and assert tasks from other columns appear.
   - Resize back and verify the same active task/status remains selected.

3. `TestE2E_TUI_NarrowMouseCompactNavigationAndDetail`
   - Start at 14x12 with `--narrow --mouse`.
   - Click the visible next arrow.
   - Double-click the active card at the row below the compact strip.
   - Assert the expected detail opens.
   - Exercise both SGR and X10 where representable.

4. `TestE2E_TUI_NarrowCRUDKeyboardFlow`
   - Start at 48x24 with `--narrow`.
   - Create, edit, move, open detail, and delete/cancel.
   - Verify persisted task state through the CLI.
   - This should pass now and protects the integration path.

### Tier 2: behavior and layout coverage

5. Hidden-column search restoration E2E after the intended UX is chosen.

6. Dialog viewport invariant tests for width and height.

7. Narrow scroll E2E:
   - mixed card heights;
   - wheel and keyboard navigation;
   - resize while scrolled;
   - selected task remains visible and Enter opens it.

8. WIP and active-tab metadata:
   - active WIP count changes tab width;
   - switching into/out of a WIP column may cross the compact fallback
     boundary;
   - navigation and mouse targets remain correct.

### Tier 3: property/stateful stress

9. Extend fuzz/state-machine coverage to vary:

   - `forceNarrow`, configured threshold, and `hideEmptyColumns`;
   - column counts from 1 through at least 20;
   - Unicode status names and double-digit task/WIP counts;
   - Tab/Shift+Tab, search edits, sort, reload, and resize sequences;
   - task changes that insert/remove columns before the active status;
   - mouse clicks on every emitted tab target.

Useful invariants:

- no panic;
- `0 <= activeCol < len(columns)` when columns exist;
- active status survives a reload when that status still exists;
- visible arrows lie inside their direction targets;
- targets remain within the viewport and do not overlap incorrectly;
- rendered board lines do not exceed terminal width;
- selected task remains visible after resize/reload when still present.

## Suggested implementation order

1. Add the failing active-status preservation model and E2E tests.
2. Fix status preservation in `loadTasks`.
3. Decide and specify no-result search behavior under
   `hide_empty_columns`; add the corresponding tests.
4. Add passing PTY coverage for configured threshold, compact mouse
   navigation, and narrow CRUD.
5. Add failing modal viewport invariants and address dialog fitting in a
   separate change, because it is broader pre-existing TUI layout work.
6. Add the stateful/property stress test after the state semantics are fixed.

## Conclusion

The final PR narrow renderer itself held up well under compact widths, Unicode,
double-digit positions, live resize, mouse input, and CRUD. The most valuable
missing coverage is not another compact-string unit test; it is PTY-level
coverage at the boundaries between narrow rendering and mutable application
state.

The first test worth adding is the hidden-column live-reload E2E reproduction.
It exposes a real selection error. The modal viewport family is the next clear
failure area, while threshold, compact mouse, and CRUD E2E tests should be
added as passing integration guards.
