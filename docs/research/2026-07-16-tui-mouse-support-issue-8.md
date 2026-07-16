# TUI mouse support assessment (GitHub issue #8)

## Question

Assess [GitHub issue #8](https://github.com/antopolskiy/kanban-md/issues/8), which proposes optional mouse support for the TUI:

- click a task card to open its detail view;
- click a visible Back action;
- scroll board columns and task details with the mouse wheel;
- preserve keyboard controls and terminal independence;
- defer drag-and-drop and reordering.

## Recommendation

Accept the proposal as a good fit for the project, with a deliberately narrow first release.

The feature strengthens the human-supervision side of kanban-md without changing the agent-facing CLI, file format, claim semantics, or keyboard workflow. The event plumbing is small; the main implementation risk is accurate hit testing across variable-height cards, per-column scrolling, responsive widths, ANSI styling, and the board's small-terminal clipping behavior.

Recommended product decisions:

1. Make mouse support opt-in initially with `kanban-md tui --mouse`.
2. Do not put the setting in `kanban/config.yml`; mouse capture is a local terminal/user preference, not a shared board property.
3. Use Bubble Tea's cell-motion mode, not all-motion mode. The initial scope needs clicks, releases, drag events, and wheel events, but not hover tracking.
4. Open a task on a single left-button release. Before opening, update the active column and row so returning to the board preserves the clicked selection.
5. Make only task cards and a visible `Back` affordance clickable in the first release.
6. Make wheel events operate on the column under the pointer while keeping the active selection visible. One card per wheel event is predictable for variable-height board cards; task detail can scroll by a small fixed number of text lines.
7. Implement a small internal hitbox/layout snapshot rather than adding BubbleZone.
8. Keep drag-and-drop, status-bar actions, dialog controls, hover effects, and card reordering out of scope.

If the opt-in behavior proves reliable across terminals, default-on mouse support with a `--no-mouse` escape hatch can be considered separately.

## Project fit

The README describes kanban-md as agent-first and human-friendly, with the TUI serving observation and supervision. Basic mouse interaction supports that positioning:

- supervisors can inspect a busy board faster;
- keyboard workflows remain unchanged;
- no additional mutation path is introduced;
- no terminal-specific integration is required;
- the core CLI remains minimal and scriptable.

Drag-and-drop would be a materially different feature. It would introduce mutation semantics, claim/WIP/error feedback, and accidental-action concerns. Deferring it is the right boundary.

## Current architecture

The project currently uses Bubble Tea v1.3.10 and starts the TUI with only `tea.WithAltScreen()` in `cmd/tui.go`.

`internal/tui/board.go` has a single top-level `Board` model with view-specific key dispatch. Adding a `tea.MouseMsg` branch and a `handleMouse` dispatcher fits the existing architecture cleanly.

Board geometry is more complex than the issue's event-handling sketch suggests:

- columns have independent `scrollOff` values;
- `activeRow` is shared as the selected row for the active column;
- card heights vary with title wrapping and optional metadata;
- up/down scroll indicators consume lines;
- column widths change with terminal width and hidden empty columns;
- the final board view is vertically clipped at very small terminal sizes;
- the detail view computes and clamps its own line-based viewport.

These constraints are already covered by substantial unit and snapshot tests, including variable-height cards, selection visibility, narrow terminals, stable status-bar placement, and detail scrolling.

## Why mouse support should start opt-in

Bubble Tea mouse mode captures mouse input that terminals would otherwise use for native text selection. There is no universal selection-bypass gesture across all terminals and multiplexers. Shift-drag is common, but documentation should describe it as terminal-dependent rather than guaranteed.

An opt-in `--mouse` flag has several advantages:

- existing TUI behavior and text selection remain unchanged;
- users can test compatibility in their own terminal/multiplexer;
- unsupported terminals naturally remain keyboard-only;
- no shared config migration is required;
- the reliable fallback is simply launching without `--mouse`.

The README should explain that native selection commonly requires holding the terminal's mouse-bypass modifier (often Shift), and that omitting `--mouse` is the portable fallback.

## Bubble Tea behavior

Bubble Tea v1.3.10 provides `tea.WithMouseCellMotion()`. Its documentation says cell-motion mode enables click, release, wheel, and drag events, prefers SGR extended reporting, falls back to the older mouse mode when needed, and automatically disables mouse reporting when the program exits.

No explicit terminal capability probe is needed for the initial feature. If the terminal does not report mouse events, keyboard input continues to work.

Bubble Tea v2 now has a different declarative mouse API and split message types. Mouse support does not need to wait for a v2 migration, but the new code should isolate v1-specific event decoding in a small handler so a later framework upgrade is localized.

Sources:

- [Bubble Tea v1.3.10 mouse option](https://github.com/charmbracelet/bubbletea/blob/v1.3.10/options.go)
- [Bubble Tea v1.3.10 mouse event types](https://github.com/charmbracelet/bubbletea/blob/v1.3.10/mouse.go)
- [Bubble Tea v2 upgrade guide](https://github.com/charmbracelet/bubbletea/blob/main/UPGRADE_GUIDE_V2.md)
- [Bubble Tea issue about mouse capture and native text selection](https://github.com/charmbracelet/bubbletea/issues/162)

## Internal hitboxes vs BubbleZone

Use an internal implementation for the first release.

BubbleZone is useful for deeply nested component trees, but kanban-md currently has one monolithic board model and only two initial target types: task cards and Back. A small internal registry keeps behavior explicit and avoids coupling the feature to a helper's render-marker lifecycle.

There are also project-specific reasons to avoid the dependency:

- the current Bubble Tea v1-compatible BubbleZone release is v1.0.0, while BubbleZone v2 targets Bubble Tea v2;
- BubbleZone v1 creates a background worker and updates zone data asynchronously, with its own documentation warning that bounds may briefly reflect a previous render;
- BubbleZone relies on wrapping rendered content in markers and scanning at the root;
- kanban-md can vertically clip the rendered board at very small terminal heights, which is an awkward edge case for marker-delimited regions;
- the project can derive exact card rectangles from the same width, card-height, visible-range, and scroll-indicator calculations it already owns.

Suggested internal shape:

```go
type mouseTarget struct {
    kind targetKind
    x1, y1, x2, y2 int
    col, row int
}
```

During rendering, build a layout snapshot containing only visible, on-screen targets. On a mouse event, dispatch against that snapshot. Keep target creation next to `renderColumn`/`viewDetail` so rendering and hit testing cannot silently drift apart.

BubbleZone sources:

- [BubbleZone repository and usage](https://github.com/lrstanley/bubblezone)
- [BubbleZone v1.0.0](https://github.com/lrstanley/bubblezone/releases/tag/v1.0.0)

## Interaction details

### Card click

- Handle left-button release rather than press.
- Ignore borders outside registered card rectangles, column headers, indicators, empty-column placeholders, and status-bar text.
- Set `activeCol` and `activeRow` to the clicked task.
- Open the detail view and reset `detailScrollOff`.

Single-click opening is appropriate because the action is non-destructive and matches the issue's primary supervision use case.

### Board wheel

- Determine the column from the pointer position.
- Ignore the status bar and coordinates outside rendered columns.
- Scroll only the hovered column.
- Keep selection and viewport state consistent; do not allow the highlighted task to remain hidden.
- Move by task/card, not raw terminal lines, because cards have variable heights.

### Detail wheel

- Wheel up/down should adjust `detailScrollOff`.
- Reuse the existing clamping performed by the detail renderer.
- A small fixed step, such as three lines per event, will feel more like a pager than one-line keyboard scrolling while remaining deterministic.

### Back action

Render an explicit affordance such as `← Back  q/esc` in the detail footer and register only the `Back` text/button as clickable. Existing `q`, `Esc`, and Backspace behavior should remain.

## Testing strategy

The existing test architecture is well suited to this feature.

Add model-level tests using synthetic `tea.MouseMsg` values for:

- clicking the first and later cards;
- variable-height cards;
- cards after an up-scroll indicator;
- clicks outside cards and below a clipped viewport;
- clicking a card in a non-active column;
- wheel up/down boundaries;
- wheel behavior over inactive and empty columns;
- detail wheel scrolling and clamping;
- clicking Back;
- ignored mouse events in dialogs/help/search/create/edit views;
- unchanged keyboard behavior when mouse support is disabled.

Add command tests for flag resolution and program option selection. If practical, extend the PTY E2E harness with SGR mouse sequences for one click-to-detail flow and one detail-wheel/Back flow. Unit tests should remain the primary cross-platform coverage because the PTY suite is excluded on Windows.

Manual smoke testing should cover a small representative matrix rather than terminal-specific code:

- one macOS terminal;
- one Linux terminal;
- tmux or Zellij;
- a terminal without useful mouse reporting;
- native text-selection bypass and the no-mouse fallback.

## Scope and priority

This is a worthwhile medium-complexity TUI enhancement, not a core roadmap blocker. It is a good contributor-sized feature once the interaction decisions above are recorded on the issue.

The implementation should remain one focused feature:

- no config schema change;
- no Bubble Tea v2 migration;
- no BubbleZone dependency;
- no mutation via mouse;
- no drag-and-drop.

That boundary gives kanban-md a noticeably better supervision experience without weakening its keyboard-first, agent-first design.
