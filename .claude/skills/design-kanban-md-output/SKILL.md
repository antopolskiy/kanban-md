---
name: design-kanban-md-output
description: >
  Preserve and evolve kanban-md table, compact, and JSON output contracts. Use
  when changing output defaults, format-selection flags or environment
  variables, renderers, fields, grouping, errors, or token efficiency. Do not
  use for TUI-only rendering or task/config YAML schemas.
allowed-tools:
  - Bash(go test *)
  - Bash(golangci-lint *)
  - Bash(git diff *)
---

# Design kanban-md output

CLI output serves humans, agents, and scripts. Treat the default, format
selection, field names, record boundaries, and error shapes as user-facing
contracts rather than incidental presentation.

## Format roles

- **table** is the default in both TTY and non-TTY environments. It is padded,
  readable output for interactive use.
- **compact** is selected with `--compact`, `--oneline`, or
  `KANBAN_OUTPUT=compact`. It uses one record per line, avoids repeated JSON keys,
  and is optimized for agent context.
- **json** is selected with `--json` or `KANBAN_OUTPUT=json`. It is the complete
  structured form for parsing, pipes, and integrations.

TTY auto-detection was intentionally removed because agents commonly execute in
non-TTY environments and previously received verbose JSON by default. Do not
restore implicit TTY-dependent selection without new evidence and an explicit
compatibility decision.

## Contract rules

- Preserve explicit format flags and environment-variable behavior. Conflicting
  selectors must fail or resolve through the documented precedence, never
  silently vary with terminal state.
- Keep compact output one-line-per-record and sufficiently self-describing for
  agents. Avoid repeated labels or decorative padding that materially increases
  token use.
- Keep JSON complete and machine-oriented. Field removals, renames, type changes,
  and error-shape changes require compatibility analysis; additive fields still
  require documentation and tests.
- Keep table output readable and stable enough for humans, while avoiding tests
  that prevent harmless spacing improvements unless alignment itself is the
  contract.
- Do not publish precise token-savings claims without a current, reproducible
  measurement against the formats being compared.
- Update `README.md`, command help, examples, and embedded skill references when
  output behavior changes.

## Implementation and verification

Centralize semantic data before rendering so table, compact, and JSON surfaces
do not drift. Test renderer behavior in `internal/output/` and complete command
selection in `e2e/`.

At minimum, cover:

- the default format in non-TTY execution;
- each explicit flag and supported `KANBAN_OUTPUT` value;
- record contents and ordering for the changed command;
- JSON fields/types and structured errors when affected;
- compact one-record-per-line behavior;
- `--no-color` or `NO_COLOR` when styling changes.

Follow `test-kanban-md` for the broader verification ladder. For the product and
token-cost rationale, read
`docs/research/2026-08-25-kanban-md-design-principles-orientation.md`.
