# Product Hunt Launch Draft

## Product name

kanban-md

## Tagline (60 chars)

File-based Kanban for AI coding agents and humans

## Short description

kanban-md is a CLI/TUI Kanban board where tasks are Markdown files in your repo. It is built for multi-agent coding workflows with atomic `pick --claim`, compact output, classes of service, and no server setup.

## Full description

kanban-md helps AI agents and humans coordinate development work without the overhead of a hosted tracker.

Instead of a remote API/database, every task is a Markdown file with YAML frontmatter. That makes tasks easy to diff, review, and merge in normal Git workflows.

It is designed for parallel agent execution:

- Atomic `pick --claim` to avoid two agents grabbing the same task
- Claim expiry so stale claims do not block progress
- Compact output mode for token-efficient agent loops
- Classes of service + WIP limits for predictable flow control

You can use it entirely from CLI, and there is also a separate TUI binary (`kanban-md-tui`) for interactive board navigation.

## First comment draft (maker comment)

I built kanban-md because I kept hitting coordination issues when multiple coding agents worked on the same repository.

Most existing project tools are API-first and produce heavy responses that are expensive in agent context windows. I wanted something deterministic, Git-native, and fast for local/CI workflows.

Key things that made a difference for us:

- `pick --claim` + claim expiry for multi-agent safety
- file-based tasks that review cleanly in PRs
- compact output for lower token usage in agent loops

Would love feedback from anyone running multi-agent coding workflows, especially on what should be improved in claim semantics, dependency handling, or TUI ergonomics.

## Media checklist

- Hero screenshot: `assets/tui-screenshot.png`
- CLI screenshot: `assets/cli-screenshot.png`
- Demo GIF: generate from `assets/demo.tape` / VHS pipeline
- Canonical URL: https://antopolskiy.github.io/kanban-md/

## Launch-day checklist (PST)

1. Confirm personal Product Hunt account has posting access.
2. Upload screenshots/GIF and finalize maker comment.
3. Schedule launch for target PST day/time.
4. Publish and monitor comments during first 6-12 hours.
5. Share organically in relevant channels (no vote-exchange/incentives).
