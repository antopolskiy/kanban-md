# Product boundaries for a principal-owner review skill

Date: 2026-09-20. Repository: `/Users/santop/Projects/kanban-md`. Inspected commit: `166245b8db3660f88b013a59d8dacbee2afed876`. The working tree was clean at the start of this audit.

This is a static audit of current local source and documentation. No application commands, tests, shared-board mutations, web requests, issue operations, worktrees, or commits were performed. Existing tests cited below were inspected, not run. File references are relative to this repository and include the inspected line numbers.

The review skill should preserve local task orchestration, compatibility, and deliberate human control without turning those aims into blanket bans on metadata, configuration, automation, or integrations. Several desirable principles describe the direction of the code more accurately than its complete current behavior. The skill must distinguish implemented behavior, documented intent, and proposed policy.

## Custom metadata is not currently preserved

`Task` is a fixed Go struct containing the canonical fields, body, and file path. It has no extension map or retained YAML document. Reading unmarshals frontmatter into that struct; writing marshals only the struct. Unknown frontmatter keys therefore have no retained representation and disappear on a normal typed rewrite. This conclusion follows directly from `internal/task/task.go:11`, `internal/task/file.go:31`, and `internal/task/file.go:45`.

That affects more than an explicit metadata edit. Moves call `task.Write` at `internal/board/mutate.go:134`; edits call `task.WriteAndRename` at `internal/board/mutate.go:454`; consistency repair can rewrite a task at `internal/task/consistency.go:119`. Reading an otherwise consistent file does not itself rewrite it, but the normal command loader also performs consistency repair, at `cmd/root.go:121` and `cmd/root.go:126`.

The same distinction applies to config. `Config` is a typed struct without an extension map, loaded with `yaml.Unmarshal` and saved with `yaml.Marshal`, at `internal/config/config.go:24`, `internal/config/config.go:422`, and `internal/config/config.go:397`. Unknown config keys have no general preservation mechanism either.

Current task JSON is also generated from the typed model. `show` embeds `*task.Task` and adds a direct `children` array, at `cmd/show.go:60`. It cannot expose metadata that the parser discarded.

Review implication: preserving supported user metadata would be a new behavior that needs an explicit contract. A principal-owner skill must not describe it as already supported. It should distinguish semantic value preservation from preservation of YAML comments, ordering, anchors, quoting, or arbitrary syntax. Those are different commitments. It should also distinguish preserving opaque metadata from allowing that metadata to acquire validation rules, query operators, automation, or inheritance.

"Only intrinsic fields" would be a poor rule. The existing task model already includes coordination state such as `claimed_by` and `claimed_at`, lifecycle timestamps, dependencies, class of service, estimates, and blocking reasons at `internal/task/task.go:18`. These fields exist because they support the workflow. Their worth does not depend on an abstract classification as intrinsic. The same workflow test should apply to a proposed field or metadata-preservation feature.

## The mutation core is shared, with meaningful exceptions

The README says invariants belong at the board boundary, at `README.md:887`. Substantial implementation supports that intent:

- CLI create and TUI create both call `board.Create`, at `cmd/create.go:87` and `internal/tui/board.go:780`. Both acquire the local creation lock, at `cmd/create.go:66` and `internal/tui/board.go:752`.
- CLI move and TUI move both call `board.Move`, at `cmd/move.go:96` and `internal/tui/board.go:1306`. That operation validates status, checks the current claim, enforces a target claim requirement and WIP, updates timestamps, writes, and logs, at `internal/board/mutate.go:81`.
- CLI edit and the TUI edit dialog both call `board.Edit`, at `cmd/edit.go:100` and `internal/tui/board.go:807`. The core checks claims, validates relationships, applies status-related WIP checks, writes, and logs, at `internal/board/mutate.go:409`.
- Delete shares `board.Delete`, whose claim check and archive write are at `internal/board/mutate.go:24`. The TUI call is at `internal/tui/board.go:1335`; CLI calls are at `cmd/delete.go:93` and `cmd/delete.go:115`.

Shared operations do not mean identical interaction policy. TUI moves read and pass through an existing task claimant so a human can move a claimed task without replacing its claim. They assign `tui@<hostname>` when an unclaimed task enters a status requiring a claim, at `internal/tui/board.go:1296` and `internal/tui/board.go:836`. This is documented at `README.md:677`. CLI moves require the caller to supply their claimant at `cmd/move.go:87`. The TUI edit dialog and delete path instead use the TUI identity and therefore can fail another claimant's ownership check. A skill should recognize an explicit human-supervision adaptation and inspect whether a proposed change preserves its intended scope.

There are also implementation gaps and separate paths that must not be canonized as product policy:

- TUI `+` and `-` shortcuts reach `executePriorityChange`, which updates the in-memory task and calls `task.Write` directly, at `internal/tui/board.go:335` and `internal/tui/board.go:1242`. This bypasses the claim checks and fresh read used by `board.Edit`. Because `task.Write` itself makes a file writable before writing, read-only claim protection does not close this gap, at `internal/task/file.go:64`. This is concrete drift from the shared-invariant intent.
- `board.Edit` delegates field application to its caller. It does not itself validate every status or priority value. CLI validation is in `cmd/edit.go:180` and `cmd/edit.go:187`; the TUI dialog chooses from configured priorities. Reusing `board.Edit` alone is not proof that a new caller enforces every invariant.
- `edit --status` sets the status at `cmd/edit.go:180`, but neither that path nor `board.Edit` calls `task.UpdateTimestamps`. `board.Move` does so at `internal/board/mutate.go:124`. Status mutation semantics therefore already differ between edit and move. A proposed shared-core change must inspect both paths.
- `board.Create` accepts a status and optional claim but has no `require_claim` check, at `internal/board/mutate.go:254` and `internal/board/mutate.go:303`. The TUI create dialog supplies no claimant at `internal/tui/board.go:772`. Do not silently strengthen the existing move/edit claim rule into a claim invariant on every possible newly created task without treating that as a behavior change.
- External editor support runs a user-selected editor on the Markdown path, then reloads files, at `internal/tui/external_editor.go:19` and `internal/tui/board.go:244`. It is a direct-file editing path, not a mutation transaction. The README explicitly supports direct edits while recommending claim-aware commands for active claims, at `README.md:234`.
- A long-lived TUI keeps its config object on ordinary task reloads, at `internal/tui/board.go:233` and `internal/tui/board.go:934`. Creation loads a fresh config but copies only `NextID` back at `internal/tui/board.go:761`. In contrast, CLI board watch reloads config before rendering at `cmd/board.go:123`. Config freshness is a separate concern from sharing a mutation function.

Concurrency guarantees also need precise language. Creation requires a caller-held lock and fresh config, at `internal/board/mutate.go:249`. `PickAndClaim` explicitly does not serialize concurrent callers, at `internal/board/mutate.go:624`. Claims are cooperative leases, not distributed transactions or security controls, as documented at `README.md:826`. The skill should ask which operation is serialized, which data is fresh, and what failure leaves on disk, rather than claim universal atomicity.

Review implication: require a proposal to identify every write path it affects. Separate domain invariants from intentional interface behavior, and flag existing bypasses as implementation debt. Do not demand identical buttons, flags, defaults, or identity handling merely because the domain mutation is shared.

## Sorting has several independent scopes

| Scope | Current behavior | Evidence |
| --- | --- | --- |
| CLI `list` | Defaults to task ID. Supports ID, title, status, priority, created, updated, and due. `--sort priority` is highest configured priority first; `--reverse` flips it. Sorting follows filtering and precedes limit. | `cmd/list.go:30`, `cmd/list.go:58`, `internal/board/board.go:31` |
| TUI cards | Starts with priority descending. Cycles priority, created, updated, title. Direction is board session state and survives field cycling. The shared comparator sorts visible tasks before assigning them to columns. | `internal/tui/board.go:42`, `internal/tui/board.go:142`, `internal/tui/board.go:362`, `internal/tui/board.go:377`, `internal/tui/board.go:963` |
| Direct child summaries | Always ascending task ID, regardless of current TUI card sort or CLI list flags. | `internal/board/children.go:50`, `internal/board/children.go:78`, `README.md:328` |
| `pick` work selection | Uses configured class order. Within `fixed-date`, compares due dates before priority. Otherwise uses descending configured priority. Filters claim, block, status, tag, direct parent, and dependency eligibility before choosing. | `internal/board/pick.go:21`, `internal/board/pick.go:34`, `internal/board/pick.go:80`, `internal/board/pick.go:110` |
| CLI `board` | A status/priority summary or grouped output. It is not the TUI card board and has no `--sort` flag. | `cmd/board.go:25`, `cmd/board.go:38`, `cmd/board.go:90` |

The low-level comparator uses configured order for statuses and priorities, case-insensitive title ordering, and due dates with missing dates last in its ascending comparison, at `internal/board/sort.go:23`. Reverse currently negates `less`, at `internal/board/sort.go:14`, so equal keys do not retain a strict comparator relation. A skill should not infer a universal tie-breaking or stable reverse-order guarantee from the use of `sort.SliceStable`.

There is also documentation drift in fixed-date selection. The README class table says earliest due date "within its priority tier" at `README.md:857`; `sortPickCandidates` compares due dates before priority at `internal/board/pick.go:88`. The source establishes current behavior. The wording does not establish a new policy to preserve.

Review implication: a request for child ordering, card ordering, or work-selection priority must state which scope it changes. A single global sort option can accidentally change unrelated behavior. Shared comparison code may be appropriate; a universal presentation and scheduling policy does not follow from that reuse.

## Hierarchy storage already supports arbitrary depth

Each task can store one optional parent ID, at `internal/task/task.go:24`. There is no depth field, task type registry, or maximum-level rule in the model. Creation and editing validate referenced IDs and reject cycles, at `internal/board/mutate.go:350`. The parent cycle walk follows parents until a root, missing record, or visited record, with no depth cap, at `internal/board/cycle.go:13`. Existing tests cover multilevel trees, subtree reattachment, and a deeper cycle, at `internal/board/cycle_test.go:35` and `internal/board/cycle_test.go:44`.

This supports acyclic parent chains of arbitrary depth in the product model. It is not a claim of unlimited practical size, recursive display, or enforced validity of every manually edited file. Normal reads validate only required fields at `internal/task/file.go:145`. The cycle helper deliberately terminates when it encounters an unrelated existing cycle, at `internal/board/cycle.go:26`; graph-read failure also leaves cycle validation skipped at `internal/board/mutate.go:377`.

Presentation and queries are shallower:

- `SummarizeChildren` includes only tasks whose immediate parent matches, at `internal/board/children.go:60`. Its test deliberately supplies a grandchild and excludes it from the parent's count, at `internal/board/children_test.go:10`.
- Both CLI show and TUI detail use that helper, at `cmd/show.go:54` and `internal/tui/board.go:2247`. The view resolves one immediate parent and displays a direct-child list. The TUI maintains an unfiltered active-task set so card search does not remove relationship context, at `internal/tui/board.go:944`.
- `list --parent` and `pick --parent` match only immediate children, at `internal/board/filter.go:62` and `internal/board/pick.go:54`.
- Completion roll-up counts terminal direct children and does not update the parent. The helper returns only a summary, at `internal/board/children.go:53`; independent parent status is explicit at `README.md:315`.

Review implication: "add deep hierarchy" may really mean better visibility or navigation for an already representable structure. A new task type or display feature should not quietly impose a hierarchy depth limit, require types for existing relationships, or make parent completion automatic. Each of those is a separate workflow policy. Conversely, existing arbitrary-depth links do not justify claiming a recursive hierarchy UI already exists.

## Defaults and migration rules are deliberate, but not uniform

New boards have six statuses, required claims on `in-progress` and `review`, four priorities, four classes, a standard default class, a one-hour claim timeout, two title lines, and age colors. Empty columns remain visible. These are defined at `internal/config/defaults.go:4`, `internal/config/defaults.go:34`, and `internal/config/defaults.go:61`, then applied at `internal/config/config.go:112`. Output defaults to a human table, with flags and `KANBAN_OUTPUT` overrides, at `internal/output/output.go:22`.

Optionality is specific. WIP limits default to unlimited at `internal/config/config.go:314`; classes can be absent under validation at `internal/config/config.go:235`; mouse reporting is opt-in at `cmd/tui.go:40` and `cmd/tui.go:93`. In contrast, narrow layout is automatic when its threshold is zero, at `internal/config/config.go:66`, and a running TUI always starts a file watcher at `cmd/tui.go:102`.

Config schema version is 11, at `internal/config/defaults.go:25`. Loading applies sequential migrations, rejects newer versions, persists a migrated config, then validates it, at `internal/config/migrate.go:9` and `internal/config/config.go:429`. A nominally observational command can therefore update old config on load.

The distinction between new and existing boards matters:

- The v6-to-v7 migration leaves `require_claim` false on existing statuses, even though new boards enable it for two statuses, at `internal/config/migrate.go:101`. The compatibility test checks this explicitly at `internal/config/compat_test.go:455`.
- The v10-to-v11 migration adds an automatic narrow threshold without changing an existing true hide-empty-columns preference, at `internal/config/migrate.go:146` and `internal/config/compat_test.go:657`.
- Other migrations intentionally introduce behavior. v2-to-v3 adds classes and a timeout; v5-to-v6 adds `archived`; v8-to-v9 changes a stored title-lines value of 1 to 2, at `internal/config/migrate.go:58`, `internal/config/migrate.go:91`, and `internal/config/migrate.go:131`.

Thus "all extras off" and "migrations never change behavior" are both inaccurate. The stronger review question is what users receive on a new board, what an omitted value means, what existing files gain on upgrade, and whether an explicit choice remains distinguishable from a former default. The README's wording that stricter policy is *normally* opt-in and prospective is more accurate than an absolute rule, at `README.md:891`.

Configuration is also bounded. Some keys are writable through `config set`; others require direct file editing, at `cmd/config.go:57` and `cmd/config.go:277`. Status order determines workflow progression and terminal meaning, at `cmd/move.go:142` and `internal/config/config.go:479`. `archived` remains a reserved status name at `internal/config/defaults.go:28`. `handoff` requires a literal `review` status at `internal/board/mutate.go:538`; `fixed-date` gets a specific selection rule at `internal/board/pick.go:88`.

Review implication: "everything configurable" would turn intentional product behavior into a sprawling configuration language. Add configuration when users have a real recurring choice. Document its precedence, persistence, validation, and migration behavior. Generic mechanisms should earn their complexity against a concrete workflow.

## Local authority allows bounded support files and explicit automation

The declared source of truth is `config.yml` plus task Markdown. The README explicitly permits a bounded audit log and a transient lock, at `README.md:234`, and says support files must not become an opaque second source of truth at `README.md:883`.

The code follows that model in several concrete ways. Mutations append local activity entries. Logging errors do not fail commands, and the log is capped at 10,000 entries, at `internal/board/log.go:13`, `internal/board/log.go:35`, and `internal/board/log.go:140`. The log is therefore diagnostic history, not a required event store from which current task state must be reconstructed. Claims and their timestamps live on tasks at `internal/task/task.go:28`.

Local automation is already part of the product:

- Commands repair duplicate IDs, filename mismatches, `next_id`, and claim-related permissions during config loading, at `cmd/root.go:126` and `internal/task/consistency.go:19`.
- Move and delete maintain lifecycle timestamps, at `internal/board/mutate.go:50` and `internal/board/mutate.go:124`.
- `pick` selects and claims work, optionally moving it, at `internal/board/mutate.go:628`.
- `handoff` combines a review move, claim refresh, optional block, note, and release, at `internal/board/mutate.go:515`.
- CLI watch and the TUI react to local file changes, at `cmd/board.go:116` and `cmd/tui.go:102`.
- `context --write-to` explicitly updates a bounded context block in another local file, at `cmd/context.go:17` and `cmd/context.go:66`.
- Skills install into agent directories from embedded local content, at `internal/skill/install.go:17`. The README describes autonomous agents using those skills under human supervision, at `README.md:57` and `README.md:794`.

The inspected non-test code in `cmd` and `internal` contains no HTTP client, webhook, or remote-sync implementation. This is evidence about the current checkout, not a prohibition on future optional adapters. The README promises no required server, account, or SaaS dependency at `README.md:12`, and rejects required external control planes at `README.md:896`.

Review implication: distinguish an integration's existence from its authority. An optional adapter that reads local files or invokes explicit local operations can preserve the product boundary. A dependency that makes normal board operations require remote availability, or introduces a competing hidden owner of task state, would change that boundary. The review must specify local behavior when the adapter is absent, unavailable, or disagrees with the board. There is no basis here for a blanket "no integrations" rule.

"No automation" would reject much of the existing product. A better test asks what triggers the action, which rule it applies, whether the result is inspectable, and who owns consequential workflow decisions. Parent status is currently independent, for example, while timestamp updates are automatic. Treat those as separate choices rather than examples of one universal automation policy.

## Replace simplistic maxims with questions tied to behavior

| Misleading maxim | Why it misleads here | More useful review question |
| --- | --- | --- |
| Only intrinsic fields belong | Existing coordination and lifecycle fields support concrete workflows. Foreign metadata preservation is an interoperability question distinct from first-class semantics. | What user or agent decision needs this data, who owns it, and what semantics does kanban-md promise? |
| Everything should be configurable | The product already combines configurable workflows with fixed semantic conventions and finite CLI options. | Is there a recurring user choice worth the validation, documentation, migration, and interaction cost? |
| All extras must default off | New boards enable claims, classes, colors, automatic narrow layout, and live TUI refresh. Old-board migrations make different choices. | What should a new user experience, and what should change for an existing board? |
| No automation | Selection, claim expiry, consistency repair, timestamps, watchers, and agent workflows already automate work. | Is this a deterministic local consequence, an explicitly invoked workflow, or an unrequested transfer of a user's decision? |
| Generic is always better | Local helpers share real rules, while `review`, `archived`, and `fixed-date` carry specific semantics. A generalized rule engine is not an automatic improvement. | Does the abstraction reduce actual duplicated rules and expose a clearer operation, or merely move complexity into configuration? |
| Zero cost when unused | Even without opening the log, mutations append and scan it; non-skill commands check installed skill staleness. Optional features also carry maintenance and compatibility costs. | What measurable runtime, output, state, setup, and maintenance costs remain for users who do not opt in? |

For the last row, log append invokes truncation scanning at `internal/board/log.go:55` and `internal/board/log.go:63`. Root command pre-run checks skill staleness at `cmd/root.go:43`. No performance measurement was made. These establish nonzero work, not a performance defect.

The principal-owner review should request concrete before/after examples across affected commands and interfaces, then state which judgment follows from existing behavior and which is a recommended product choice. This avoids defending implementation drift as a principle or presenting a preferred future contract as a shipped guarantee.
