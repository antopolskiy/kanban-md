# Child roll-up demo board

This self-contained board demonstrates direct child visibility for GitHub issue
#11. Task `#1` is a manually managed parent with children in backlog,
in-progress, review, done, and archived states. Task `#7` is a grandchild and
demonstrates that the detail roll-up is intentionally one level deep.

From the repository root:

```bash
# Human-readable detail: 1/4 active children are terminal.
go run ./cmd/kanban-md --dir examples/issue-11-demo show 1

# Include the archived child: 2/5 are terminal.
go run ./cmd/kanban-md --dir examples/issue-11-demo show 1 --archived

# Compact and JSON contracts.
go run ./cmd/kanban-md --dir examples/issue-11-demo --compact show 1
go run ./cmd/kanban-md --dir examples/issue-11-demo --json show 1

# Interactive detail view. Search for exact ID #1, keep the search, then open it.
go run ./cmd/kanban-md --dir examples/issue-11-demo tui
# Press: /  #1<space>  Enter  Enter
```

Expected behavior:

- Default CLI and TUI detail show children `#2` through `#5`, ordered by ID.
- The default roll-up reads `1/4 done`.
- `show 1 --archived` additionally shows `#6` and reads `2/5 done`.
- Grandchild `#7` is absent from task `#1`'s direct-child list.
- Filtering the TUI to parent `#1` does not make its children disappear from
  the detail view.
- Moving children does not move or complete parent `#1`; its status is manual.
