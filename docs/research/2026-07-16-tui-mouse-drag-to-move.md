# TUI mouse drag-to-move assessment

## Conclusion

Yes. kanban-md can support press-drag-release moves without rendering a card that follows the pointer.

The interaction is best described as **drag-to-move** or **drop-to-status**:

1. press the left button on a task card;
2. move the pointer toward another status column;
3. release anywhere in that column's board area;
4. invoke the same validated `board.Move` operation used by keyboard TUI moves.

The destination card or vertical coordinate has no ordering meaning. Only the destination column/status matters.

Bubble Tea's `WithMouseCellMotion()` is sufficient. It reports button press, release, wheel, and movement while a button is held. A release event also includes coordinates, so the final destination can still be resolved even if intermediate motion messages are sparse.

This should be implemented as a follow-up to basic click/wheel support, not silently folded into the first mouse release. The initial feature establishes hitbox accuracy and terminal compatibility; drag-to-move adds mutation, error feedback, and accidental-action concerns. The underlying mouse state machine should nevertheless be designed so the follow-up is a natural extension.

## Why the current move architecture works

`internal/board.Move` already centralizes the required behavior:

- validates that the target status exists;
- verifies claim ownership;
- treats a move to the current status as an idempotent no-op;
- enforces `require_claim`;
- enforces status and class-of-service WIP limits;
- updates lifecycle timestamps;
- applies an automatic claim when requested;
- writes the task;
- records the activity log.

The TUI's `executeMove` also already supplies `tuiClaimant()` and `SetClaim` when the target status requires a claim.

A drop must reuse this path rather than editing `Task.Status` or writing the Markdown file directly.

Expected behavior:

- dropping an unclaimed task into a `require_claim` status auto-claims it as the TUI user, matching keyboard moves;
- dropping a task claimed by another actor fails through normal claim validation;
- dropping into a full WIP-limited column fails and leaves the task in its source column;
- dropping a blocked task succeeds like existing TUI moves; `board.Move` returns
  a warning, although the current TUI does not surface `MoveResult.Warnings`;
- dropping into the source column does not write or log a move.

## Interaction state machine

The click state machine proposed for basic mouse support should become a small pointer gesture state machine.

Suggested state:

```go
type pointerGesture struct {
    active       bool
    taskID       int
    sourceStatus string
    sourceTarget mouseTarget
    pressX       int
    pressY       int
    hoverStatus  string
    crossedColumn bool
    layoutVersion uint64
}
```

### Press

On an unmodified left-button press inside a visible card:

- store the task ID and source status;
- store the card target and current layout version;
- update keyboard selection to the pressed task;
- do not open or mutate anything yet.

Presses on headers, scroll indicators, empty placeholders, the status bar, or unused terminal space do nothing.

### Motion while held

Cell-motion events can update `hoverStatus`:

- over another valid board column: set that column as the potential destination;
- over the source column or outside the board: clear the destination;
- optionally highlight the destination column header and show a status hint.

No card needs to follow the pointer. A destination highlight is still valuable feedback:

```text
Move #42 → review — release to move
```

Motion is useful for feedback but should not be required for correctness. The release coordinates are authoritative.

### Release

Resolve the release against the current visible layout.

- Release inside the original card, without having crossed into another column: treat it as a normal click and open detail.
- Release in a different valid board column: move the stored task ID to that column's status.
- Release elsewhere in the source column: cancel; do not open and do not move.
- Release over the status/search bar, blank separator, unused right-side space, or outside the board: cancel.
- Release after the gesture was invalidated: ignore.

Accept both Bubble Tea release forms:

- SGR: `Action == MouseActionRelease`, `Button == MouseButtonLeft`;
- legacy X10: `Action == MouseActionRelease`, `Button == MouseButtonNone`.

The stored press state identifies the gesture as a left-button drag when X10 cannot identify the released button.

### Cancel and invalidate

Clear the gesture on:

- window resize;
- task/config reload;
- search or sort changing the layout;
- view transition;
- `Esc`;
- a different button press;
- the task disappearing, being archived, or being filtered out;
- a layout-version mismatch;
- terminal focus loss, if focus reporting is added later.

The cursor does not need to remain inside the original card during the drag. Leaving the original card is exactly what enables a cross-column drop.

## Destination geometry

Card hitboxes are required for the press, but a drop should target a **column region**, not another card.

A destination column region should include:

- the column header;
- visible card space;
- empty space below the last card;
- the empty-column placeholder.

It should exclude:

- the board's blank separator and status/search bar;
- an error toast;
- unused terminal space to the right when columns have reached `maxColWidth`;
- horizontally clipped/non-rendered columns;
- archived statuses, which are not part of `BoardStatuses()`.

This makes empty status columns easy drop targets and reinforces that the operation changes status rather than order.

At extremely small terminal heights, a column needs at least one actually rendered row to be a destination. Invisible geometry must never accept a drop.

## Click-versus-drag behavior

No timer is necessary.

The terminal reports cells, not continuous pixels, and moving into a different column is already a strong deliberate gesture. Recommended interpretation:

- same original card on release: click;
- different column on release: move;
- same source column but a different location: cancel.

This avoids:

- long-press latency;
- time-dependent/flaky tests;
- accidental detail opening after a failed same-column drag;
- interpreting vertical movement as reordering.

Optionally, `crossedColumn` can prevent a gesture that briefly entered another column and returned to the original card from opening detail. Cancelling that release is safer than opening unexpectedly.

## Mutation execution details

The move should use the task ID stored at press time, not whichever task happens to be selected at release time.

The current `executeMove(targetStatus)` obtains `selectedTask()`. For drag support, extract a helper shaped like:

```go
func (b *Board) executeMoveTask(taskID int, targetStatus string) (tea.Model, tea.Cmd)
```

Keyboard operations can pass the selected ID; drag operations pass the pressed ID. This prevents a reload, selection change, or future mouse behavior from moving the wrong task.

After a successful drag move:

- reload the board;
- select the moved task in the destination column;
- keep that task visible;
- clear the drag indicator;
- optionally show a short non-error confirmation in the status bar.

After a failed drag move:

- reload to reflect current disk state;
- keep/select the task in its actual column when still visible;
- show the existing error toast;
- clear all drag state.

### Concurrent changes

The file watcher should cancel an active drag on `ReloadMsg`. This prevents applying a drop against geometry that is known to be stale.

There remains a small race if another process changes the task between the final reload and the move. `board.Move` re-reads the task and validates its current claim and target constraints, so it will not blindly write the stale in-memory card. If stricter source-status compare-and-swap semantics become important, they should be added to `board.Move` as a general concurrency feature rather than implemented only for mouse gestures.

## Error and safety behavior

Drag-to-move is a mutation, so failure must be obvious.

Required cases:

- **claimed by another actor:** task remains in place; error toast identifies the claim conflict;
- **WIP limit reached:** task remains in place; error toast reports the limit;
- **class WIP limit reached:** same behavior;
- **invalid/stale destination:** cancel without mutation;
- **same status:** cancel/no-op without a success message;
- **write failure:** task remains based on the reloaded disk state; error shown;
- **blocked task:** match existing TUI behavior. If move warnings are surfaced
  later, keyboard and mouse moves should use the same presentation.

No confirmation dialog is necessary for an ordinary status move. Keyboard `n`, `p`, and `m` already move without confirmation. The deliberate cross-column gesture plus clear destination highlighting provides equivalent intent.

Drag-to-move should remain available only when mouse mode is explicitly enabled.

## Automated testing

The testability work for basic mouse support covers the transport and geometry. Drag adds a focused state-machine and mutation matrix.

### Model/state tests

- press and release on the same card opens detail;
- press a card and release in each different status column moves to that status;
- release in an empty column moves correctly;
- release over another card changes only status and does not reorder;
- release in the same column outside the card cancels;
- release over status bar, separator, error toast, or unused space cancels;
- press outside a card and release in a column does nothing;
- motion over destinations updates only the hover indicator;
- motion is not required when release lands in another column;
- entering then leaving another column and releasing on the source follows the chosen cancellation rule;
- SGR left release works;
- X10 buttonless release works;
- right/middle/modified gestures do nothing;
- resize, reload, search, sort, or view change cancels;
- stale layout versions cannot move a task;
- the stored task ID is moved even if keyboard selection changes;
- successful move follows the task to its destination;
- failed move restores/selects the task in its actual source column.

### Mutation/error tests

- drop into `require_claim` auto-claims using `tuiClaimant()`;
- drop of a task claimed by another actor fails;
- drop into a full status WIP limit fails;
- drop into a full class WIP limit fails;
- class bypass of a column WIP limit succeeds;
- lifecycle timestamps match keyboard moves;
- one activity-log entry is written on success;
- no log entry or task write occurs on cancellation/same-column release.

### Geometry tests

- every rendered board column has one non-overlapping drop region;
- no drop region extends into the status bar/error toast;
- empty columns remain valid targets;
- hidden empty columns have no target;
- `maxColWidth` leaves unused right-side space untargeted;
- narrow and short terminals accept drops only in rendered regions;
- resize produces new regions and invalidates the old gesture.

### PTY end-to-end journey 1: successful drag

`TestE2E_TUI_MouseDragMovesTask`

1. Launch the compiled binary with `tui --mouse`.
2. Press a card in backlog.
3. Send one or more held-button motion events into an empty destination column.
4. Release in the destination column.
5. Verify through `kanban-md show` that the task status changed.
6. Press Enter and verify the destination task opens, proving selection followed it.
7. Return and quit.

Run protocol subtests:

- SGR press/motion/release;
- X10 press/motion/buttonless release.

### PTY end-to-end journey 2: rejected drag

`TestE2E_TUI_MouseDragMoveRejected`

Use subtests for the two most important safety failures:

- destination WIP limit is full;
- source task is claimed by another actor.

For each:

1. start the drag and release in the target column;
2. wait for a unique error message emitted after the drop;
3. verify through `kanban-md show` that status and ownership did not change;
4. verify the TUI remains responsive and keyboard navigation still works.

### Additional PTY cases

- drag directly from press to release without intermediate motion;
- release outside the board and confirm no mutation;
- resize between press and release and confirm cancellation;
- release into an empty column at a small terminal height;
- repeat targeted drag tests with `-count=10`.

## Scope recommendation

Support drag-to-move, but stage it:

### Phase 1: basic mouse support

- card click to detail;
- Back click;
- board/detail wheel scrolling;
- internal hitboxes and column regions;
- press/release gesture state designed for extension.

### Phase 2: drag-to-move

- destination-column highlight;
- drop-to-status mutation through `board.Move`;
- success/failure selection behavior;
- mutation-focused model and PTY tests;
- documentation that vertical drop position does not reorder cards.

Keeping the mutation in a follow-up makes review and regression diagnosis substantially easier. It also preserves issue #8's stated non-goal for the first implementation while confirming that the desired interaction is architecturally supported.
