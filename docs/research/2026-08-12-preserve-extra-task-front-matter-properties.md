---
relationships:
  references:
    - internal/task/task.go
    - internal/task/file.go
    - internal/board/mutate.go
    - internal/task/consistency.go
    - internal/tui/board.go
---

# Preserve extra task front-matter properties

## Acceptance boundary

[Issue 16](https://github.com/antopolskiy/kanban-md/issues/16) establishes the
need: tools that share task files must be able to add properties without a
later kanban-md mutation deleting them.

The accepted boundary is semantic YAML values:

- preserve additional properties composed of scalars, lists, and recursively
  string-keyed maps;
- keep kanban-md fields typed and authoritative;
- reserialize the complete front matter as YAML after a mutation; and
- make no preservation guarantee for formatting, comments, key order, anchors,
  aliases, or explicit tags.

Unknown properties remain file-only. Table, compact, and JSON output continue
to expose only the typed task contract.

## Observed data-loss mechanism

`task.Read` previously decoded front matter into the closed `Task` struct. The
YAML decoder ignored unknown mapping keys, and every mutation serialized that
struct through `task.Write`. The first write therefore removed all values that
did not have a field on `Task`.

The loss is centralized in `task.Write`, but it affects every path that reads
and rewrites an existing task:

| Surface | Persistence path |
| --- | --- |
| Edit, claim, release, block, unblock | `board.Edit` → `task.WriteAndRename` → `task.Write` |
| Move | `board.Move` → `task.Write` |
| Archive and delete | `board.Archive` or `board.Delete` → `task.Write` |
| Handoff | `board.Handoff` → `task.Write` |
| Pick and claim | `board.PickAndClaim` → `task.Write` |
| Terminal User Interface mutations | Shared board mutations or `task.Write` |
| Consistency repair | `repairFilenameMismatches` → `task.Write` |

`create` has no prior front matter to preserve. Read-only commands do not
normally write, but configuration loading can trigger consistency repair.

## Storage design

`Task` retains an unexported `map[string]any` containing decoded values for
properties it does not own. Custom YAML marshal and unmarshal methods keep this
state inside `internal/task`:

1. Decode the mapping into the typed task fields.
2. Derive the canonical key set from the `Task` YAML tags.
3. Inspect each explicit additional property and decode only supported values.
4. Encode current canonical values from the typed task.
5. Encode and append the additional values.

The unexported map is absent from JSON output. A task constructed in memory has
no additional values and produces the same canonical YAML as before.

This design retains supported values rather than syntax nodes. Properties that
use anchors, aliases, merges, or explicit tags are outside the preservation
boundary. They do not prevent a task from loading or being updated, but their
representation after an update is not guaranteed. Comments, styles, and source
order have no retained representation, but do not make an otherwise supported
property unsupported.

## Validation boundary

Additional values are retained when their complete syntax tree contains:

- unanchored, implicitly tagged YAML scalar nodes, including null;
- sequences whose items recursively satisfy this boundary; and
- mappings whose keys are unanchored, implicitly tagged strings and whose
  values recursively satisfy this boundary.

An additional property is not retained as a semantic value when its key or any
nested value uses an anchor, alias, merge, explicit tag, non-string mapping key,
or another YAML node shape. Unsupported additional properties do not prevent
the task from loading or being updated. Their representation after an update is
not part of the preservation contract. Duplicate YAML keys remain invalid
through the YAML decoder. Canonical keys never enter the additional-property
map, so a future typed field with the same name becomes authoritative
automatically.

## Compatibility and verification

No task-format version bump or migration is required. Canonical field names and
types do not change, and old files without additional properties retain their
existing behavior.

The verification set covers:

- scalar, sequence, null, and nested string-keyed map values;
- successful load and update with properties containing anchors, aliases,
  merges, explicit tags, or non-string map keys;
- comment and key-order normalization without dropping supported values;
- optional canonical field removal and addition;
- absence from JSON, compact, and table output;
- edit and title rename, move, claim, handoff, archive, consistency repair, and
  Terminal User Interface mutation paths; and
- the version 1 compatibility fixture.

The repository has no `CONTRIBUTING.md` or pull request template. Maintainer
pull requests use conventional commits, user-visible summaries, explicit
validation commands, edge-case tests, README updates, and issue references.
