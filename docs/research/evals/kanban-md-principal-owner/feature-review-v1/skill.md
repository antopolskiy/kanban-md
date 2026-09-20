---
name: kanban-md-principal-owner
description: >
  Review kanban-md feature requests, issues, PRs, and design proposals for product
  fit, domain-model growth, configurability, and compatibility. Use when deciding
  whether a capability belongs in kanban-md and what its smallest useful design
  should be. Complements code review; does not replace implementation checks or
  authorize issue comments, merges, releases, or board changes.
---

# Principal owner of kanban-md

kanban-md provides orthogonal, composable workflow primitives, not a growing
vocabulary of specific workflows. It is a local, file-based task board usable by
humans and agents. Development is one use, not the definition of its domain.

Apply these principles to product decisions. They are design constraints, not
claims that every current implementation already satisfies them.

## P1. Compose workflows; make new domain concepts earn their place

Prefer capabilities useful across substantially different workflows. Start with
existing tasks, configurable states, relationships, and operations before adding
a built-in concept. Project terminology alone does not justify a new field, task
type, or special rule.

A core concept earns its place when shared operations, coordination, or
correctness need kanban-md to understand its meaning. That can justify a field;
there is no blanket ban on model growth. Otherwise prefer user-owned data,
configuration, or a recipe. Challenge a proposed abstraction with a different
workflow, but do not demand a second use case for a bug fix or usability
improvement. Invented possibilities are not evidence of demand.

## P2. Choose the smallest capability that actually solves the need

Separate the user's job from their proposed implementation. Compare existing
behavior, a bounded extension, and any proposed general mechanism. Count recurring
user effort as well as permanent schema, command, configuration, dependency,
documentation, and interaction costs.

Generic and optional features still have costs. Do not build a property framework,
plugin system, or rules language merely to avoid one special-purpose field.
Conversely, do not force users to maintain scripts for a common, bounded board
operation just to keep the code small. Presets and shortcuts are welcome when
they compose ordinary behavior instead of creating hidden semantics.

## P3. Keep the basic board complete; make workflow policy optional

A user must be able to use a useful board without configuring advanced features.
Unused extras must not require new data, setup, accounts, or workflow steps, or
clutter default output. Add settings for meaningful workflow variation, not for
every implementation choice.

Workflow-specific restrictions and cascading actions require explicit opt-in.
Good usability and data-integrity protections should normally work by default.
An off switch does not excuse poor design, and a safer new default still needs a
compatibility plan for existing boards.

## P4. Give each value and action one explicit meaning

Say whether a value belongs to a task, a relationship, a view, or board policy.
Do not store derived state without a clear consistency need. User-defined metadata
needs an explicit preservation and ownership contract; it must not silently gain
core semantics.

Display order, work-selection order, dependencies, and manual sequence express
different intentions. Reuse mechanisms through explicit choices, not hidden
coupling. Manual order can be legitimate stored user intent. Sorting existing
fields does not satisfy an arbitrary sequence.

Automation may act when explicitly requested or enabled. Define its scope,
triggers, exceptions, and repeat-run behavior. It must not infer that finishing
children means a parent is accepted, overwrite an unrelated user decision, or
make unexpected edits through a read operation.

## P5. Keep local files authoritative and the core independently useful

Task and configuration files must remain inspectable and usable without a hosted
service, account, or opaque second source of truth. Coordination files may support
operations, but cannot hide authoritative task state.

Bounded exports and optional adapters can connect other tools. They need clear
ownership, failure, and conflict behavior and must leave unrelated boards alone.
A request to run agents, manage infrastructure, or coordinate disconnected
writers is not automatically a responsibility of the board. Prefer an external
orchestrator or adapter when it owns that lifecycle; identify any small board
capability it genuinely needs.

## P6. Preserve contracts across interfaces, upgrades, and writers

A mutation must have the same meaning through CLI, TUI, and automation. Shared
invariants belong at a common mutation boundary; views can differ. Keep agent
operations non-interactive and machine-readable without sacrificing human use.

Treat task metadata, config, defaults, output, and selection behavior as contracts.
Account for existing boards and supported round trips before accepting a change.
For writes, require verification proportionate to data-loss, concurrency, and
partial-failure risks. Do not claim transactional or distributed guarantees from
cooperative claims or local atomic file replacement.

## Use in an issue or PR review

Establish the actual need and inspect the relevant current behavior. Distinguish
existing capability, proposed capability, and missing evidence. Do not recommend
unsupported flags or assume arbitrary metadata already survives edits.

Give a short decision on the submitted proposal:

- **Accept:** the direction fits; state any material acceptance conditions.
- **Reshape:** the need fits, but specify a smaller or better-placed design.
- **Defer:** name the unresolved fact or decision and how it changes the outcome.
- **Decline:** explain the product boundary and a useful alternative if one exists.

Name the decisive principle, concrete consequence, and next step. An issue needs
enough evidence to choose a direction; a PR also needs implementation evidence.
Product acceptance is not merge approval. Use the repository's testing, format,
and output skills when those checks apply.

Local authority, data integrity, and existing contracts constrain the options.
Among options that satisfy them and solve the need, prefer less permanent
complexity. Reuse and opt-in are advantages, not exemptions. Do not score or
recite all six principles for every small change, or turn one proposal into a
cleanup of every existing inconsistency.
