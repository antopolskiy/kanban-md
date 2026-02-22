# TUI text-input package selection for create/edit flow (Task #196)

## Objective
- Improve task create/edit typing behavior using existing terminal UI components instead of the custom ad-hoc editors.
- Prefer a package from the same ecosystem for minimal risk and consistent Bubble Tea integration.

## Packages evaluated

- `github.com/charmbracelet/bubbles/textarea`
  - Multi-line editor with cursor navigation, custom keymaps, paste support, clipping/wrapping.
  - Public API exposes `Model`, `Update`, `View`, `SetValue`, `SetWidth`, and `SetHeight`.
  - Good fit for the create/edit body field.

- `github.com/charmbracelet/bubbles/textinput`
  - Single-line editor with standard editing keys (cursor movement, insert/delete at cursor, etc.) and validation hooks.
  - Good fit for the create/edit title and tags fields.

## Suggested implementation direction
1. Keep current create/edit flow and step model as-is.
2. Replace custom string/cursor editing for title/tags with `textinput.Model`.
3. Replace custom body editor with `textarea.Model` (at least for create/edit body step).
4. Sync existing validation (`createTitle`, `createBody`, `createTags`) from the component values at execution time.
5. Preserve existing step transitions (`Tab` / `Shift+Tab` / `Enter`), but delegate all other key handling to the component's `Update`.

## Why this is preferable
- Reuses well-tested components from the Bubble Tea ecosystem already listed as dependencies (`github.com/charmbracelet/bubbles`).
- Avoids implementing selection/movement/shortcut behavior from scratch.
- Gives a cleaner path for multiline body editing without manual cursor/line management.

## Acceptance criteria for follow-up
- Title/body/tags editing supports standard cursor movement, insertion/deletion, and completion behavior in both create and edit flows.
- Body editing no longer shows brittle line-growth behavior from manual single-line rendering.
- Existing tests around create/edit tasks remain green, with added coverage for the new component-backed input path.
