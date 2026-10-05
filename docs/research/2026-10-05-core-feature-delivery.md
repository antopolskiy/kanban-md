# Core feature delivery and user guide

The user authorized moving PR 18's contribution to maintainer-controlled branches, implementing the accepted core recommendations, verifying them, and delivering a standalone HTML user guide with real terminal screenshots and technical tooltips/callouts. The user chose core features and report only, excluding a VS Code extension. No worktrees or Kanban tracking. Use the existing checkout; preserve local main's eight earlier commits. Root owns branch switches and remote writes. No merge, release, original-PR closure, or issue comments are authorized.

## Starting revisions and publication

- Preserved main: `e334563d9c84ff883970077d29a1e2e9cb6e28fd`, clean, eight commits ahead of origin/main.
- Upstream main: `6f01678748ac44027b58ca98ce62a680ec899963`.
- Contribution: PR 18 head `f756a9d52251842004b9f751e12e6cc0ef12c642`.
- Persistence branch: `codex/preserve-task-properties`, starting at the exact contribution head. Publish one corrected contribution PR to upstream main, retaining the contributor's commits.
- Core features branch: `codex/composable-workflow-views`, created from the accepted persistence candidate. Publish a dependent PR targeting the persistence branch so its diff shows only the new capabilities.
- The charter is committed on local main before dispatch; the owner prompts name that immutable revision. Main's prior commits and this local research checkpoint must stay out of the published code branches.

## Product and integration invariants

Foreign metadata survives unrelated writes, or the affected write refuses before destroying task data. Canonical task fields stay authoritative. Preservation alone does not expose every extra field in output. Selection and display are explicit; presentation order never changes pick, priority, dependencies, claims, WIP, or completion. Existing boards retain usable defaults and supported output. Depth is derived, navigation is session state, and read-only interactions do not rewrite task files. All data examples are synthetic demo boards.

## Pre-dispatch lane charters

### Persistence owner

Identity `preservation_owner`. Base `f756a9d52251842004b9f751e12e6cc0ef12c642`; branch `codex/preserve-task-properties`; checkout `/Users/santop/Projects/kanban-md`. Implement the accepted design in `2026-10-02-pr18-completion-owner.md`. Allowed production scope is `internal/task/task.go`, `internal/task/frontmatter.go`, the narrow TUI priority failure rollback, and README preservation guidance. Tests/fixtures may cover task, board, TUI, CLI E2E and historical compatibility. No property querying, new schema, recovery-policy rewrite, or unrelated cleanup.

First reproduce precision loss and silent omission at the contribution head, then implement opaque value retention and bounded alias safety. Keep ordinary reads separate from write admissibility; preserve existing actionable failure for required unsafe consistency repairs. Run focused-to-full tests, historical compatibility, lint, and precommit. Commit only owned files. Handoff exact SHA and evidence in `2026-10-05-preservation-handoff.md`. Root obtains independent review before publishing.

### TUI owner

Identity `workflow_tui_owner`. Initial design-only phase reads the contribution and pinned main; implementation base will be the accepted persistence SHA. Own tag search, direct relation navigation/back history/ancestor context, and derived exact-depth filtering/text depth cues. Allowed touch set is TUI behavior, dedicated relation/depth helpers, affected board read-only helpers, behavioral/snapshot/PTTY E2E tests. Root owns final README/help reconciliation. No recursive descendant configuration, hover reporting change, new domain taxonomy, or metadata authoring UI. Preserve existing key behavior where compatible, missing-parent IDs, archived context, filtering independence, and task ID selection. Produce a design checkpoint before editing and atomic focused-tested commits after clearance.

### Property design owner

Identity `property_design_owner`. Design-only phase, no production edits. Propose one bounded scalar property access/authoring contract, exact filtering/grouping, explicitly selected output/display, and numeric direct-child sorting. Use the revised combined comparison in `2026-10-02-issues-metadata-ordering-design.md`. Compare costs with canonical type/rank without assuming a framework is required. Root will settle API/config syntax and touch ownership before implementation.

No canonical `type`, `child_rank`, schema/rules language, hidden parent-clear cleanup, or pick coupling. Use historical config migrations for any chosen view settings. Default output remains stable; selected-value output is explicit. Numeric comparison must preserve exact values and distinguish numeric-looking strings. Report the concrete shared interface, migration/defaults, typed scalar authoring syntax, JSON selection, tests and conflict boundaries in `2026-10-05-property-design.md`.

### Capture owner

Identity `terminal_capture_owner`. Own only local report-support files under `docs/research/2026-10-05-delivery/`. Build a repeatable headless capture tool using installed ttyd and headless browser runtime, or a simpler real PTY renderer. Run the real binary in a synthetic temporary board. Target a 1280 by 800 or 1440 by 900 laptop-sized terminal with readable monospace text; preserve terminal output and control sequences accurately. Do not operate visible user apps. No VM needed unless simpler capture fails.

Produce progress-reporting scripts, an evidence manifest tying each screenshot to a binary revision, terminal rows/columns, viewport, commands and key actions, and PNGs. First prove the capture tool against an existing binary; final images must be recaptured after accepted integration. No fabricated app output or image generation. No app source edits or dependency changes. Report only the evidence report path after tool handoff.

### Report author

Start after integrated features and screenshots are available. Own the first complete self-contained report at `docs/reports/kanban-md-workflow-guide.html`, with embedded screenshots, ordinary-user explanations, accessible hover/focus/tap definitions, permanent glossary, and technical details in disclosures. Read the interactive-report skill and required references/template. Root reviews HTML and claims, runs validation and headless browser interaction tests, and delivers the artifact.

## Sequence and conflict policy

Persistence implementation and independent review precede any property implementation. TUI and property designs may run read-only in parallel. Capture tooling can proceed independently in ignored research files. In the shared checkout only one production owner may edit a central file at a time. TUI code and property rendering both touch board.go, so root sequences those edits and owns reconciliation. Stage only explicit lane files. Only root runs branch switches, pushes, PR creation/attachment, and full integration gates.

## Acceptance and evidence

Use meaningful pre-fix regressions for persistence and missing user behavior. Test package contracts, historical config/task fixtures, CLI E2E and real PTY interactions. Validate full Go tests, lint and make precommit on the integrated commit. Independently review high-risk persistence and final composition against exact revisions. Record finding dispositions, acceptance SHAs, fixes and test commands here or linked reports. Run a midpoint headless demo once persistence is accepted; capture final screenshots only from the accepted integrated binary. HTML validation and tooltip/theme/print/narrow-layout interactions must pass before delivery.

Rollback points are the preserved local main and each accepted atomic branch checkpoint. There is no deployment or release. Keep original contribution attribution. The final report must distinguish supported implemented behavior from proposed/deferred designs and state that screenshot captures ran in a background terminal, not on the user's physical MacBook.
