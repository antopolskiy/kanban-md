---
name: evolve-kanban-md-formats
description: >
  Evolve kanban-md config.yml schemas and task Markdown frontmatter without
  breaking existing boards. Use when changing config fields, defaults,
  migrations, CurrentVersion, YAML tags, task metadata, or compatibility
  fixtures. Do not use for CLI output-only changes or ordinary task data edits.
allowed-tools:
  - Bash(go test *)
  - Bash(golangci-lint *)
  - Bash(git diff *)
---

# Evolve kanban-md formats

Treat `config.yml` and task frontmatter as durable user data contracts. Existing
boards must remain readable after an upgrade, and migrations must be explicit,
ordered, and covered by compatibility fixtures.

## Configuration design

- Collocate new or migrated per-column settings with the corresponding entry in
  `statuses`. This applies to settings such as `show_duration`, `require_claim`,
  and WIP limits.
- The existing top-level `wip_limits` map is a compatibility shape, not a model
  for new per-column settings. Migrate it only as part of an intentional schema
  revision.
- Prefer backward-compatible defaults and prospective enforcement. Do not make
  an existing board invalid merely because a new optional policy exists.

## Config schema changes

1. Inspect `internal/config/defaults.go`, `internal/config/migrate.go`, and every
   existing compatibility fixture before editing the schema.
2. Increment `CurrentVersion` in `internal/config/defaults.go`.
3. Add a migration from the previous version in `internal/config/migrate.go`,
   register it in the `migrations` map, and increment `cfg.Version` in the
   migration.
4. Create `internal/config/testdata/compat/vN/`, where `N` is the old version.
   Start from the previous fixture and make it representative of the format
   being migrated, including sample task files when relevant.
5. Add or extend `internal/config/compat_test.go` assertions so the old fixture
   loads through all migrations and exposes the intended current values.
6. Update the generated-config example and user guidance when the serialized
   shape or defaults change.

Never skip intermediate versions or mutate an old fixture to look like the new
format; fixtures document what users already have on disk.

## Task frontmatter changes

- Additive optional fields with `omitempty` preserve old files that omit them,
  but still require a fixture and parsing assertion for the new field.
- Do not rename or remove an existing YAML tag. If semantics must evolve, keep
  reading the old representation and migrate or translate it explicitly.
- Exercise new task fields in `internal/task/testdata/compat/v1/tasks/`, or create
  a new versioned fixture directory when the task format itself becomes
  versioned or requires migration.
- Add assertions in `internal/task/compat_test.go` for parsing, zero values, and
  round-trip behavior where relevant.

## Verification

Run every historical compatibility fixture, not only the new one:

```bash
go test -run Compat ./internal/config/ ./internal/task/
go test ./internal/config/ ./internal/task/
```

Then follow `test-kanban-md` for repository-wide verification and inspect the
serialized fixture/config diffs manually.
