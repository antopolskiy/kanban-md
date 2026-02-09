# Kanban-Based Development

A methodology for autonomous, parallel-safe development using kanban-md to manage tasks and git worktrees to isolate work.

## Multi-Agent Environment

**This board is shared.** Multiple agents and humans may be working on it simultaneously. You are NOT the only one reading or modifying tasks. This means:

- Another agent may claim a task between the time you list it and try to pick it.
- Another agent may be merging into main while you are trying to merge.
- Tasks you saw as available a moment ago may no longer be available.

The **claim** mechanic is the coordination primitive. It prevents two agents from working on the same task. **You MUST claim a task before starting any work on it, and you MUST only pick unclaimed tasks.** Violating this causes duplicate work, merge conflicts, and wasted effort.

## Prerequisites

- Use the installed `kanban-md` CLI for all board interactions
- Ensure you are on the `main` branch before starting

## Agent Identity

Each agent session must generate a unique name to identify itself for claims. At the very start of a session, run:

```bash
awk 'length >= 4 && length <= 8 && /^[a-z]+$/' /usr/share/dict/words | sort -R | head -2 | tr '\n' '-' | sed 's/-$//'
```

This produces a name like `rapidly-almoign` or `fiber-kindly`. **Remember this name in your context** and use it as a literal string in all claim/release commands for the rest of the session. Do not store it in a file or environment variable — those are not persistent or isolated between agents.

Example: if the generated name is `frost-maple`, use `--claim frost-maple` in every claim command.

## Workflow

### 1. Pick and claim a task

Use the `pick` command to atomically find the highest-priority unclaimed, unblocked task and claim it in one step:

```bash
kanban-md pick --claim <your-agent-name> --move in-progress
```

To pick from a specific status column:

```bash
kanban-md pick --claim <your-agent-name> --move in-progress --status todo
kanban-md pick --claim <your-agent-name> --move in-progress --status backlog
```

This is atomic — if another agent claims the task between your list and claim, `pick` handles it safely. No need to list/choose/claim manually.

If no tasks are available, `pick` will tell you. Wait or check if there are blocked tasks that need unblocking.

### 2. Create a worktree

Create a git worktree for the task so work is isolated from `main` and from other agents:

```bash
git worktree add ../kanban-md-task-<ID> -b task/<ID>-<short-description>
cd ../kanban-md-task-<ID>
```

### 3. Implement and test

Work inside the worktree. The task stays at **in-progress** throughout this step.

1. Implement the fix or feature.
2. Run tests: `go test ./...`
3. Run lint: `golangci-lint run ./...`
4. Update golden files if needed: `go test ./internal/tui/ -run TestSnapshot -update`

### 4. Commit and move to review

Once tests and lint pass, commit inside the worktree:

```bash
git add <files>
git commit -m "feat: <description>"
```

Now move the task to **review**. This signals: "code is complete and committed, awaiting merge to main."

```bash
kanban-md move <ID> review --force
```

(Use `--force` because you hold the claim — the move command requires it for claimed tasks.)

**The task MUST stay in `review` until it is merged into main.** Do not move it to `done` yet.

### 5. Merge into main

Switch back to the main working directory and check if the state is clean:

```bash
cd /Users/santop/Projects/kanban-md
git status
```

#### If main is clean

Merge your branch:

```bash
git merge task/<ID>-<short-description>
```

Then clean up the worktree:

```bash
git worktree remove --force ../kanban-md-task-<ID>
git branch -d task/<ID>-<short-description>
```

**Only after the merge is on main**, release the claim and move to **done**:

```bash
kanban-md edit <ID> --release --force
kanban-md move <ID> done --force
```

#### If main is NOT clean

Another agent is currently merging. Do NOT force or overwrite. The task stays in **review**. Instead:

1. Go back and **pick another task** (repeat from step 1 in a new worktree).
2. Complete that task as well.
3. Wait until main is clean, then merge both branches sequentially:
   ```bash
   git merge task/<FIRST-ID>-<desc>
   git merge task/<SECOND-ID>-<desc>
   ```
4. Resolve any conflicts if needed, run tests again to confirm everything passes.
5. Clean up both worktrees and branches.
6. **Only after each branch is merged**, release its claim and move to done.

### 6. Status lifecycle

Statuses have strict meanings. Never skip ahead.

| Status | Meaning | Enters when | Leaves when |
|---|---|---|---|
| `in-progress` | Agent is actively working | `pick --claim --move in-progress` | Tests + lint pass, code committed |
| `review` | Code committed, awaiting merge to main | `move <ID> review --force` | Branch merged into main |
| `done` | Merged into main | `edit <ID> --release --force` then `move <ID> done --force` | Never |

To abandon a task: release the claim and move back to `todo`:

```bash
kanban-md edit <ID> --release --force
kanban-md move <ID> todo --force
```

### 7. Release

After merging and confirming tests pass on main:

```bash
# Tag and push (release workflow triggers automatically)
git tag vX.Y.Z
git push origin main --tags
```

Then write release notes per the project guidelines (see CLAUDE.md / AGENTS.md).

## Rules

### Claiming (most important — prevents duplicate work)

- **Always claim before working.** Never start work on a task without claiming it first. Use `pick --claim` to do this atomically.
- **Only pick unclaimed tasks.** Use `pick` or `list --unclaimed`. Never manually select a task that is already claimed by someone else.
- **Never override another agent's claim.** If a task is claimed, it belongs to that agent. Pick a different task. Do not use `--force` to steal claims.
- **If `pick` fails, pick again.** Another agent got there first. This is normal in a multi-agent environment. Just run `pick` again for the next available task.
- **Release claims when done or abandoning.** Always `edit <ID> --release --force` before moving to `done` or back to `todo`.

### Status discipline

- **`in-progress` means actively working.** Only claimed tasks should be in-progress.
- **`review` means committed, awaiting merge.** The task stays here until the branch is merged into main. Do not skip to `done`.
- **`done` means merged into main.** Only move here after `git merge` has landed on main and tests pass.

### Git safety

- **Never merge into a dirty main.** Always check `git status` first.
- **Always use worktrees** to isolate task work from main and from other parallel tasks.
- **One merge at a time.** If main is occupied, work on something else and come back.
- **Test before merging.** Run `go test ./...` and `golangci-lint run ./...` on main after every merge.
