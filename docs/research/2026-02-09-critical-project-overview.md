# Critical Project Overview: kanban-md

**Date:** 2026-02-09
**Scope:** Full project review for mission clarity, vision consistency, implementation risks, and forward directions.

## Executive Summary

kanban-md is technically strong and unusually mature for a file-based CLI product: broad end-to-end testing, backward-compat migrations, and clear agent-oriented primitives are already in place. The core execution quality is high.

The main weaknesses are strategic, not foundational code quality:

1. Roadmap drift is now visible (Layer 6 MCP + Layer 5 schema were framed as key milestones but are still absent).
2. Vision language is internally inconsistent across docs (minimal/no rendering vs shipped TUI/watch mode).
3. Multi-agent coordination remains cooperative only; there is no hard concurrency control for contested writes.

This project is in a strong position to become a reference implementation for agent-native local task orchestration, but it needs a tighter product narrative and a focused next milestone set.

## Method

- Reviewed README, roadmap/planning docs, implementation reports, research notes, and core packages under `cmd/` and `internal/`.
- Executed:
  - `go test ./...` (pass)
  - `go test -run Compat ./internal/config/ ./internal/task/` (pass)
  - `go test -cover ./...` (pass; strong coverage in internals, lower in `cmd`)
  - `golangci-lint run ./...` (not available locally in this environment)

## Mission and Vision Assessment

### Stated mission (clear)

- Agent-first, file-based, multi-agent-safe kanban (`README.md:10`, `README.md:18`, `README.md:19`, `README.md:20`).
- No database/server/SaaS, and lean runtime model (`README.md:10`, `README.md:21`).

### Implementation alignment (good)

- Agent ergonomics are real, not aspirational:
  - compact output + env override
  - structured error model
  - claim semantics and `pick`
  - installable agent skills
- Backward compatibility discipline is institutionalized (`internal/config/migrate.go`, compat fixtures/tests).
- Operational quality is high: large e2e suite and stable CI matrix.

### Vision consistency gaps (material)

- README says tool does not "render boards" while it ships board rendering + TUI (`README.md:580`, `README.md:392`).
- Roadmap "What NOT to build" rejects file watchers and TUI board view, but both exist (`docs/plans/proposals.md:33`, `docs/plans/proposals.md:42`, `cmd/board.go:40`, `cmd/kanban-md-tui/main.go:1`, `internal/watcher/watcher.go:1`).
- Roadmap positioned MCP as v1.0 marquee and Layer 7 after Layer 6, but Layer 7 shipped first and MCP is absent (`docs/plans/proposals.md:24`, `docs/plans/proposals.md:25`, `docs/plans/layer-6-mcp-server.md:5`, `docs/research/2026-02-08-layer-7-implementation.md:117`).

## Strengths

1. Strong engineering rigor
- Extensive automated testing (very broad e2e surface and unit coverage in internals).
- Backward-compat schema migration + fixtures for config/task formats.
- Good cross-platform attention (Windows/macOS/Linux CI, path-safe tests).

2. Clear domain model
- Core entities and lifecycle (status, priority, dependencies, blocking, claims, class-of-service, timestamps) are coherent.
- Error taxonomy is explicit and machine-consumable (`internal/clierr/clierr.go`).

3. Practical agent UX
- Compact output mode with documented rationale.
- Batch operations with partial-success semantics.
- Context generation and skill distribution significantly reduce prompt friction.

## Gaps and Potential Issues

## High Priority

1. Strategic roadmap drift and stale narrative
- Planned high-impact features are missing from shipped command surface (no `mcp`, no `schema`).
- This creates expectation debt and weakens product signaling for early adopters.
- References:
  - MCP plan and target release (`docs/plans/layer-6-mcp-server.md:1`, `docs/plans/layer-6-mcp-server.md:5`, `docs/plans/layer-6-mcp-server.md:36`)
  - Schema command plan (`docs/plans/layer-5-agent-integration.md:116`)
  - Current command set has no `schema`/`mcp` (`cmd/`)

2. Cooperative concurrency without hard write safety
- `pick` is "logical atomic" but not protected by lock/CAS.
- Task writes are direct `os.WriteFile` with read-modify-write workflows; concurrent mutation can still race.
- References:
  - Pick flow (`cmd/pick.go:71`, `cmd/pick.go:109`)
  - Task write behavior (`internal/task/file.go:38`, `internal/task/file.go:57`)

## Medium Priority

3. Config command lags config schema
- Config includes claim/class/TUI-related fields, but CLI config accessor set is partial.
- This weakens "agent-native, no-manual-YAML-needed" positioning.
- References:
  - Config schema fields (`internal/config/config.go:31`, `internal/config/config.go:32`, `internal/config/config.go:33`, `internal/config/config.go:51`)
  - Exposed config keys (`cmd/config.go:126`)

4. "Ready" context section hardcodes second status as ready queue
- For custom workflows, this semantic assumption can produce misleading context.
- Reference: `internal/board/context.go:190`

5. Dependency deletion behavior can leave work permanently blocked
- Missing dependency IDs are treated as unsatisfied, so deletes may strand dependents unless cleaned manually.
- Might be correct by policy, but it conflicts with earlier design notes and needs explicit user-facing guidance.
- Reference: `internal/board/filter.go:133`

## Low Priority

6. Context marker replacement lacks malformed-marker guard
- Reversed begin/end marker order is not explicitly validated.
- Reference: `internal/board/context.go:327`

7. Local lint workflow still auto-fixes
- `make lint` runs `golangci-lint --fix`, which can surprise contributors by mutating files during validation.
- Reference: `Makefile` (`lint` target)

## Forward Directions

## 1) Product Direction Decision (immediate)

Choose and state one primary identity:

- **Option A: Agent-native orchestration platform**
  - Prioritize MCP + schema contracts + deterministic machine interfaces.
- **Option B: Minimal local markdown kanban**
  - Keep integrations lighter, de-emphasize protocol/server ambitions.

Current implementation trends toward Option A; docs and roadmap should be updated to match.

## 2) Next Milestone (30-60 days)

If Option A:

1. Ship `schema` command first (small, high leverage).
2. Ship minimal MCP core (`list/show/create/edit/move/delete`) without over-scoping resources/prompts initially.
3. Freeze and publish output/error contract versioning policy.

If Option B:

1. Tighten consistency docs.
2. Improve core collaboration safety (locking/atomic writes).
3. Expand ergonomics around large-board performance and maintainability.

## 3) Concurrency Hardening (short term regardless of strategy)

1. Atomic write via temp file + rename for task/config/log files.
2. Optional advisory lock on pick/claim path for high-contention workflows.
3. Add explicit conflict/error code for detected concurrent mutation.

## 4) Narrative and Docs Cleanup

1. Reconcile README principles with shipped capabilities (`README.md:580` vs TUI/watch features).
2. Update `docs/plans/proposals.md` to reflect actual sequencing and accepted exceptions.
3. Mark deferred layers explicitly as "deferred" or "replanned" to avoid stale commitments.

## Recommended Priority Queue

1. Align docs and roadmap with reality.
2. Implement `schema` command.
3. Add write-safety hardening (atomic writes + optional lock path).
4. Decide on MCP scope and commit/no-commit explicitly.
5. Expand `config` command surface to cover claim/class defaults.

## Final Assessment

kanban-md already has high implementation quality and strong practical utility. The project’s biggest risk is not code correctness; it is strategic ambiguity between "minimal local CLI" and "agent-native orchestration platform."

Resolve that ambiguity, then execute the corresponding next milestone set. If done, the project can credibly own its niche.
