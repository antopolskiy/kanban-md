# kanban-md design-principles orientation

**Date:** 2026-08-25

**Task:** #276 — Define application design principles

**Scope:** Local repository history, current user-facing behavior, package boundaries,
compatibility policy, tests, and recent design research. No external research was
needed for this orientation.

## Executive summary

kanban-md already contains a short four-item `Design principles` section in the
README. The section dates back to the first documented version of the project and
currently says:

1. Agent-first, human-friendly.
2. Files are the API.
3. No hidden state.
4. Minimal by default.

These are useful origin signals, but not yet sufficient contributor guidelines.
Two have become factually stale or too absolute as the product matured:

- The board now has an `activity.jsonl` mutation log and a `.lock` file, so “no
  lock file” is false. Neither is authoritative hidden state, which suggests that
  the durable principle is closer to **no hidden authoritative state**.
- Task files remain the durable interoperability boundary, but actively claimed
  task files are deliberately made read-only on Unix-like systems, and kanban-md
  owns the serialization of documented front-matter fields. “Files are the API”
  therefore needs an explicit ownership and mutation contract.

The implementation and recent design reviews reveal a richer, more durable core:

- local files are the source of truth and remain inspectable with ordinary tools;
- agent workflows are first-class, while human supervision remains a product
  requirement rather than an afterthought;
- mutations converge on shared domain operations and protect workflow invariants;
- concurrency mechanisms are cooperative, explicit, and recoverable;
- defaults are permissive and familiar, while stricter policy is opt-in;
- persisted schema and machine-facing output are compatibility boundaries;
- view concerns should not casually become permanent task-domain fields;
- the product should solve local orchestration without becoming a hosted service
  or general workflow-automation engine.

The most important brainstorming question is not “what principles can we invent?”
but “which of these observed values are constitutional, and how should they decide
future contribution disputes?”

## Product orientation

### Primary identity

The current README describes kanban-md as an “agents-first file-based Kanban” for
parallel AI-agent workflows with human supervision. Its differentiating bundle is:

- one Markdown file per task;
- local configuration in YAML;
- a single Go binary with no database, server, account, or SaaS dependency;
- token-efficient CLI output and structured JSON;
- atomic task selection and cooperative claims;
- embedded agent skills describing the expected operating workflow; and
- a built-in TUI for observing and manipulating the same board.

This is narrower than a general issue tracker and broader than a passive Markdown
viewer. It is a local orchestration tool whose persistence model is intentionally
legible to humans, scripts, Git, and agents.

### Two product participants, not one

The product repeatedly designs for two participants:

- **Agents** need non-interactive commands, compact context, deterministic output,
  explicit errors, atomic coordination operations, and installable instructions.
- **Humans** need readable defaults, discoverable commands, an interactive board,
  safeguards around destructive actions, and enough visibility to supervise
  concurrent work.

“Agent-first” therefore does not mean “agent-only.” The current product is strongest
when the same domain operation works through both the CLI and TUI, with presentation
adapted to each participant.

### Deliberate non-capabilities

The README and historical planning decisions consistently avoid:

- a required database or background server;
- hosted synchronization and accounts;
- notification systems and external-service integrations in the core;
- network-dependent operation; and
- automatic workflow decisions such as deriving and writing parent status from
  child state.

Git, ordinary files, and external tools remain the extension mechanisms. The built-in
TUI, watcher, log, metrics, and skills show that “minimal” cannot simply mean “few
features.” A more accurate interpretation is **a bounded core with no infrastructure
tax and no external control plane**.

## Architectural orientation

The package graph reflects a clear, useful separation:

- `cmd/` is the command adapter and presentation-routing layer.
- `internal/board/` owns board-level operations and shared mutation semantics used
  by the CLI and TUI.
- `internal/task/` owns the task model, validation, Markdown/YAML persistence,
  filename conventions, and consistency repair.
- `internal/config/` owns board configuration, validation, discovery, versioned
  migrations, and compatibility fixtures.
- `internal/output/` owns table, compact, and JSON representations.
- `internal/tui/` is an interactive view/controller over the same config, task, and
  board operations.
- `internal/filelock/` and task-file permissions provide process and cooperative
  coordination safeguards.
- `internal/skill/` distributes the operational protocol to coding agents.
- `internal/watcher/` observes file changes; it is not a second data store.

The June 2026 refactor moved mutation behavior from command-specific code into the
internal API. That is an architectural statement: invariants belong at the shared
domain boundary, not independently in every UI.

### Persistence and coordination model

The durable board consists of:

- `config.yml`;
- `tasks/*.md`; and
- an append-only `activity.jsonl` audit log.

A transient `.lock` coordinates contested creation/config updates. Claims are
stored on tasks and act as cooperative leases; claimed files are also made read-only
on Unix-like systems as an accidental-edit safeguard. Consistency repair reconciles
IDs, filenames, `next_id`, and task-file permissions.

This model suggests an important distinction for future wording:

- **authoritative state** must be local, inspectable, portable, and recoverable;
- **derived or coordination state** may exist, but must not become an opaque second
  source of truth.

### Compatibility is institutionalized

Config has evolved through eleven schema versions. Each version has an explicit
migration chain and fixtures proving that older boards still load with current code.
Task fields are extended additively, with tests for older task files. The project
also treats exact compact output, JSON shape, defaults, and TUI snapshots as behavior
that can require compatibility evidence.

This is stronger than a general preference for “not breaking users.” It is a design
constraint: persisted files and machine-facing representations are public contracts.

## Historical design principles and how they aged

### 1. Agent-first, human-friendly

**Still strong.** Compact output, `pick --claim`, structured errors, batch behavior,
skills, table output, TUI workflows, and narrow-terminal support all reinforce it.

**Needed clarification:** features should be evaluated in both agent and human
contexts, but exact parity of presentation is not necessary. Shared semantics matter
more than identical interfaces.

### 2. Files are the API

**Still foundational, but underspecified.** Task files are directly inspectable,
Git-friendly, and usable by other tools. Config discovery and watcher behavior make
the filesystem a live integration boundary.

**Needed clarification:** kanban-md owns documented fields and may normalize their
YAML serialization. External metadata needs an explicit preservation contract.
Direct edits are supported, but claimed-file protection means the command layer is
the safe mutation API during coordinated work.

### 3. No hidden state

**The intent survives; the literal wording does not.** The `.lock` file and activity
log are present. They do not create a hidden database, and the lock is transient.

**Likely durable replacement:** no hidden authoritative state; state that affects
behavior must be local, inspectable, explainable, and repairable.

### 4. Minimal by default

**Directionally useful, semantically vague.** The product now includes a substantial
TUI, watchers, metrics, activity logs, hierarchy views, file protection, and skills.

**Likely durable replacement:** keep the core bounded and local; use simple,
permissive defaults; add capability only at the smallest coherent domain boundary;
avoid infrastructure and automation that the files, Git, or another tool can own.

### 5. Predictable output (an original principle that disappeared)

The first README included “Predictable output” as a principle. It was later replaced
by the more distinctive “Agent-first, human-friendly,” but the implementation still
treats output predictability as essential. Table is deliberately always the default,
including in non-TTY environments; compact and JSON are explicit, and flag/env
precedence is documented and tested.

This value may belong under the agent-first principle or may deserve its own principle
about deterministic, scriptable interfaces.

## Implicit principles visible in recent design decisions

### Prefer semantic contracts over accidental representation contracts

The extra-front-matter reviews separate semantic value preservation from lossless
YAML syntax preservation. They recommend preserving supported values while allowing
comments, quoting, anchors, and layout to normalize, rather than turning kanban-md
into a general-purpose round-trip YAML editor.

This is a reusable rule: promise the smallest contract that solves the user problem,
and do not accidentally make an implementation representation permanent.

### Keep domain state, relation state, and view state distinct

The child-ordering review rejected a public `child_rank` field because a presentation
choice would have become task-domain data and appeared in YAML, JSON, flags, and
outputs. It preferred existing sort policies first, then a general ordered-collection
abstraction if exact shared order proves necessary.

This is one of the clearest prospective contributor rules in the repository:
**classify a new concept before choosing where to persist it**.

### Make stricter behavior opt-in and prospective

Config migrations preserve old behavior. Existing boards do not automatically gain
claim requirements. Proposed task types remain optional and permissive; new guardrails
would reject future invalid mutations without making old data unreadable or silently
rewriting it.

This points to a broad policy: reads remain available, legacy data remains inspectable,
and new restrictions should normally constrain future writes rather than rewrite
history.

### Preserve human control over workflow state

Hierarchy roll-ups are informational. Recent type/hierarchy design explicitly keeps
parent status manual and treats inferred status, if ever added, as advisory first.
Claims coordinate ownership but expire and can be released; they are not security
boundaries.

The product helps participants make and coordinate decisions without silently making
workflow decisions for them.

### Design concurrency as a first-class domain concern

Atomic `pick`, advisory file locking, expiring claims, required-claim statuses,
claimed-file protection, WIP validation, and self-healing consistency checks all
exist because multiple writers are normal, not exceptional.

Contributions that add read-modify-write behavior should therefore explain their
concurrency boundary and failure/recovery behavior.

### Keep outputs proportional to their consumer

The table/compact/JSON trio intentionally serves three different needs: interactive
reading, token-efficient agent context, and full structured scripting. The removal of
TTY auto-detection was deliberate because environmental cleverness made output less
predictable for agents.

This suggests a general rule: explicit modes and stable defaults beat surprising
context detection.

### Prefer shared invariants over duplicated UI behavior

CLI and TUI have different interaction models, but create/edit/move/pick/handoff
semantics converge on shared internal operations. Recent contributor review emphasized
the mutation and consistency-repair paths, not merely the visible command being edited.

New behavior should be enforced once at the lowest correct shared boundary, then
rendered appropriately at each surface.

## Product tensions the principles should resolve

### Files as API versus safe coordinated mutation

If arbitrary direct editing is always first-class, read-only claimed files look like
a contradiction. If commands are the only API, the interoperability story weakens.
A likely resolution is:

> Files are the durable data and integration contract; commands are the coordination-
> safe mutation contract when workflows or claims are active.

### Minimal core versus useful integrated experience

The built-in TUI and skills are now central to the product, so “minimal” should not
be used as a blanket argument against integrated features. A principle needs a sharper
test: does the feature strengthen local task orchestration without requiring a new
service, hidden store, or unrelated platform capability?

### Agent-first versus general CLI conventions

Some conventional CLI choices are suboptimal for agents, as the TTY/JSON reversal
showed. The project needs to state whether agent context cost and non-interactive
predictability may deliberately outweigh conventional auto-detection. Current evidence
says yes, while preserving human-readable defaults.

### Extensibility versus schema discipline

Users and integrations want extra metadata, types, ordering, and display controls.
Every first-class field expands the task schema, outputs, migration burden, and future
name-collision surface. The recent research consistently prefers explicit namespaces,
sparse extensible objects, and delaying permanent fields until the concept's ownership
is clear.

### Self-healing versus surprising mutation

The CLI repairs ID and permission inconsistencies automatically and persists config
migrations. That supports resilience, but it also means some reads can write. The
principles should identify the boundary: deterministic, explainable repairs that
restore existing invariants are acceptable; speculative reinterpretation of user data
is not.

### Flexible boards versus opinionated Kanban semantics

Statuses and priorities are customizable and restrictions are often opt-in, while WIP,
classes of service, claims, and flow metrics encode meaningful Kanban concepts. The
tool is not merely a generic Markdown-record manager. A principle should clarify that
configuration may rename and tune the workflow without dissolving the product into an
arbitrary schema/database system.

## Candidate principle territories for brainstorming

These are deliberately territories, not final wording:

1. **Local, inspectable truth** — authoritative state lives in portable files; derived
   state is explainable and rebuildable.
2. **Agents act, humans remain in control** — optimize autonomous execution and human
   supervision together.
3. **Coordination is a domain feature** — concurrency, claims, atomicity, and recovery
   are part of feature design.
4. **One semantic core, multiple interfaces** — CLI, TUI, and future consumers share
   invariant-enforcing operations.
5. **Stable contracts, explicit evolution** — persisted schemas and machine outputs
   evolve additively or through migrations.
6. **Model the right thing in the right place** — distinguish task facts, relations,
   workflow policy, and view preferences before persisting a field.
7. **Permissive defaults, explicit guardrails** — unchanged behavior unless the board
   opts into stricter policy; legacy data remains readable.
8. **Predictability over magic** — stable defaults and explicit modes beat environment-
   dependent behavior.
9. **Bounded product scope** — strengthen local orchestration; avoid hosted services,
   opaque control planes, and automatic workflow ownership.
10. **Recover rather than destroy** — soft deletion, migration, repair, and clear
    diagnostics preserve user work whenever possible.

Ten principles would probably be too many for the main guideline. Several territories
can be combined. A practical final set is likely five to seven principles, each with:

- a memorable title;
- a one-sentence normative rule;
- one or two consequences for contributors; and
- an explicit “this does not mean …” boundary where misinterpretation is likely.

## Questions for the maintainer

1. Is the product's primary identity best stated as **local agent orchestration**,
   **file-based Kanban**, or a deliberate combination of both? Which side wins when
   they conflict?
2. Should task Markdown be treated as a stable public schema for external writers, or
   primarily as a human-readable persistence format whose safe writes go through the
   CLI?
3. Is backward compatibility a constitutional principle, or an engineering policy that
   may yield before v1.0 when a simpler model is materially better?
4. Does “minimal” mean a small feature set, zero infrastructure/runtime obligations, a
   narrow domain, or all three?
5. How strongly should the application resist workflow automation? Is “advisory before
   automatic” the durable boundary?
6. Should recoverability be elevated to a top-level principle, given soft delete,
   self-healing IDs, migrations, and the emphasis on preserving hand-edited data?
7. When agent ergonomics conflict with established human CLI conventions, should agent
   predictability normally win, as it did for output detection?
8. Which principles should be hard review gates versus directional preferences?

## Local evidence reviewed

- Current `README.md`, especially product positioning, commands, output behavior,
  multi-agent workflow, and the existing design-principles section.
- Initial README at commit `b0380c8` and the principles reconciliation at `8b0a935`.
- `AGENTS.md`, including compatibility, output, TDD, and board-workflow requirements.
- Core packages under `cmd/`, `internal/board/`, `internal/task/`, `internal/config/`,
  `internal/output/`, `internal/tui/`, `internal/filelock/`, `internal/skill/`, and
  `internal/watcher/`.
- Config migration chain and compatibility fixtures through schema version 11.
- Unit and end-to-end tests for output formats, concurrency, file protection,
  consistency repair, lifecycle behavior, and cross-interface semantics.
- Recent research on task types, hierarchy child ordering, front-matter preservation,
  YAML library boundaries, claimed-file protection, and local history recovery.
- Historical critical project overview at commit `2e4c950` and token-efficient output
  research at commit `f51cff1`.
- Recent contribution history and the upstream contribution expectations recorded at
  commit `6fb53fa`.

## Orientation conclusion

The project is not starting from a blank philosophical page. Its code and review
history already make repeated choices in favor of local inspectability, explicit
coordination, stable contracts, permissive evolution, bounded scope, and human control.

The next phase should turn those repeated choices into a short constitutional document
that helps a contributor answer questions such as:

- Does this new field represent task truth or only one view?
- What happens when two agents mutate it concurrently?
- What contract does it add to files, JSON, compact output, and migrations?
- Is the behavior opt-in, predictable, and recoverable?
- Does it advance local agent orchestration without turning kanban-md into another
  service or general-purpose workflow engine?

Those questions are more actionable than the current four slogans and provide a strong
basis for collaborative brainstorming.
