---
name: test-kanban-md
description: >
  Plan and run verification for kanban-md changes, including test-driven bug
  fixes, Go package tests, CLI end-to-end tests, TUI behavior and snapshots,
  lint, and precommit checks. Use when implementing or reviewing a change,
  fixing a bug, adding tests, updating snapshots, or deciding what evidence is
  required. Do not use as the source of release or schema-migration policy.
allowed-tools:
  - Bash(go *)
  - Bash(golangci-lint *)
  - Bash(make *)
  - Bash(git diff *)
---

# Test kanban-md

Choose the smallest fast feedback loop while developing, then widen verification
in proportion to the change before declaring it complete.

## Required discipline

- Fix bugs with test-driven development: first reproduce the exact failure in a
  test and confirm it fails for the expected reason, then implement the fix.
- Match the test layer to the behavior. Prefer package tests for internal logic,
  command tests for CLI wiring, E2E tests for complete binary workflows, and TUI
  behavioral or snapshot tests for rendered interaction.
- Run focused tests during implementation, the affected package suite after the
  change, and the full suite before completing production-code work.
- Run lint after Go changes and resolve findings introduced by the change.
- Update golden files only after confirming the new rendering is intended, then
  inspect the resulting diff rather than accepting it blindly.
- Report the exact commands run and distinguish passing checks, known baseline
  failures, and checks that were not run.

## Bug-fix loop

1. Identify the affected package or interface and study nearby test patterns.
2. Add the narrowest test that reproduces the reported behavior.
3. Run that test and confirm the pre-fix failure is meaningful.
4. Implement the smallest sufficient fix.
5. Run the reproducer again and confirm it passes.
6. Run the affected package or E2E suite.
7. Run the full repository tests and lint.
8. Review snapshots, generated files, and the final diff for unintended changes.

Do not weaken an assertion merely to make a failing test pass. If the expected
behavior is unclear, resolve that product decision before encoding it.

## Common verification commands

```bash
# One test or package during development
go test ./path/to/package -run TestName -count=1
go test ./path/to/package -count=1

# Repository-wide checks
go test ./...
golangci-lint run ./...

# Full branch validation used by the repository
make precommit
```

Use `make precommit` before integration when the change warrants the full build,
race-enabled coverage, E2E, lint, and clean-worktree checks. If it fails because
of a pre-existing baseline, verify the current diff separately and report the
baseline precisely; do not silently present a partial check as a full pass.

For package locations, helpers, and TUI snapshot commands, read
[references/test-patterns.md](references/test-patterns.md).
