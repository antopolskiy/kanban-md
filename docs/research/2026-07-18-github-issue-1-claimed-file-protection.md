# GitHub issue #1: claimed-file protection

## Question

Does the current implementation address [GitHub issue #1](https://github.com/antopolskiy/kanban-md/issues/1), which raised the risk that an agent could edit task Markdown directly and bypass another agent's claim?

## Finding

Yes. Release [v0.35.0 "Lockstep"](https://github.com/antopolskiy/kanban-md/releases/tag/v0.35.0) shipped an OS-level guardrail in commit [`0b7e412`](https://github.com/antopolskiy/kanban-md/commit/0b7e41239b055c4b8704858b1a6365f53b17eeb8): actively claimed task files are made read-only on Unix-like systems.

This implements the hardening mechanism discussed in the issue without relying on a Claude-specific `PreToolUse` hook:

- `internal/task/file.go` writes claimed task files with mode `0444`.
- kanban-md temporarily makes a claimed file writable for CLI mutations, writes it, and restores read-only mode while the claim remains active.
- Releasing a claim makes the file writable again.
- Board consistency checks repair stale file modes and restore writability after claim expiry.
- The README documents the behavior and its limits under **Multi-agent workflow > Claims**.

This directly reduces the accidental-overwrite scenario from the issue: another same-user agent attempting an ordinary direct write to a claimed Markdown file receives a permission error.

## Verification

The implementation has unit and end-to-end coverage for claiming, releasing, direct-write rejection, claimant CLI edits, renames, expired claims, and permission self-healing.

Verified on 2026-07-18:

```text
$ go test ./internal/task -run 'Test(Write|RepairFilePermissions)' -count=1
ok github.com/antopolskiy/kanban-md/internal/task

$ go test ./e2e -run '^TestFileProtection_' -count=1
ok github.com/antopolskiy/kanban-md/e2e
```

## Limitations

The protection is intentionally a guardrail rather than a security boundary. A process running as the same OS user can still change permissions, rename, or delete the file. Unix permission enforcement also does not apply on Windows. These limits are documented in the README and v0.35.0 release notes.

## Recommendation

Comment on issue #1 with the v0.35.0 implementation and limitation details, then close it as completed. The implementation addresses the reported accidental direct-edit bypass while preserving the project's plain-Markdown design.
