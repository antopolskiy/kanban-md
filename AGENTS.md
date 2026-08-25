# Guidelines

> **Note:** `CODEX.md` and `CLAUDE.md` are symlinks to `AGENTS.md`. Only edit
> `AGENTS.md`; both files stay in sync.

## Universal rules

- Use semantic commit messages:

  ```text
  feat: <short description>

  Explain the change in natural paragraphs without artificial line wrapping.
  - Use bullets when they improve readability.
  ```

- Update `README.md` whenever user-facing behavior changes, including commands,
  flags, defaults, installation methods, or workflows.
- Document research in `docs/research/YYYY-MM-DD-<description>.md`. When
  delegating research, require the subagent to create the report and return only
  its path so the primary agent can read the source directly.

## Workflow routing

- For repository implementation work, use `kanban-based-development`. Board
  tracking and an isolated worktree are the defaults unless the user explicitly
  overrides them.
- For task capture, board changes, triage, or status reporting, use `kanban-md`.
  "Add ticket" means capture a task for later without starting implementation.
  Use `layer-N` roadmap tags, imperative titles, `high` priority for explicit
  user requests and bugs, `medium` for planned work, and `low` for ideas.
- For tests, bug fixes, snapshots, lint, or verification strategy, use
  `test-kanban-md`.
- For `config.yml` schema, task frontmatter, migrations, or compatibility
  fixtures, use `evolve-kanban-md-formats`.
- For table, compact, JSON, output-selection, or renderer changes, use
  `design-kanban-md-output`.
- For releases, tags, publishing, or release notes, use `release-kanban-md`.
