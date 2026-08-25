---
name: release-kanban-md
description: >
  Release kanban-md through its tag-triggered GoReleaser workflow, monitor CI,
  recover safely from failures, and publish user-facing GitHub release notes.
  Use when the user asks to release, tag, publish, or prepare release notes for
  kanban-md. Do not use for ordinary commits or unreleased changelog edits.
allowed-tools:
  - Bash(git *)
  - Bash(gh *)
  - Bash(make *)
  - Bash(go *)
  - Bash(golangci-lint *)
---

# Release kanban-md

Carry an explicitly requested release from preflight through a verified GitHub
release. A request to inspect, plan, or draft a release does not authorize
pushing a tag or editing a live release; perform remote mutations only when the
user has asked to release or publish.

## Release invariants

- A version tag triggers the GitHub Actions `release` workflow, which uses
  GoReleaser to create the GitHub release and its artifacts.
- Never run `gh workflow run release`; tag push already triggers the workflow
  and a manual dispatch can create a duplicate build.
- Never run `gh release create`. Let GoReleaser create the release, then update
  its title and notes with `gh release edit` after CI succeeds.
- Never move, delete, or reuse a tag that has been pushed. After a non-transient
  failure, fix the cause and choose a new, higher version.
- Decide the version autonomously using semver: patch for fixes, minor for
  backward-compatible features, and major for breaking changes. Do not ask the
  user to confirm the version.
- A release is complete only when the workflow is green, the generated release
  exists, its human-written notes are published, and the final release is
  verified.

## 1. Preflight

Work from a clean, up-to-date `main` containing exactly the changes intended for
release. Do not discard, overwrite, or silently stash unrelated user changes.

```bash
git branch --show-current
git status --short
git fetch origin --tags
git status --short --branch
git tag --sort=-version:refname | head -20
gh release list --limit 100
```

If `main` is dirty, diverged, or missing intended work, resolve that through the
normal project development workflow before releasing. Run the release-appropriate
verification, normally:

```bash
make precommit
```

Do not tag a revision that fails required tests or lint.

## 2. Inspect changes and choose the version

Identify the previous release tag and review the user-visible changes, including
outside contributions that may need attribution:

```bash
git log vPREVIOUS..HEAD --oneline
git diff vPREVIOUS..HEAD --stat
git diff vPREVIOUS..HEAD
```

Choose `vX.Y.Z` according to semver and confirm that it does not already exist
locally or remotely. The tag must identify the verified `main` commit.

## 3. Tag and push

Push `main` and the exact new tag. Do not use `--tags`, which can publish other
local tags unintentionally.

```bash
git push origin main
git tag vX.Y.Z
git push origin vX.Y.Z
```

## 4. Watch the release workflow

Find the run associated with the new tag and watch it to completion:

```bash
gh run list --workflow release --limit 10
gh run watch <RUN_ID>
```

Confirm the selected run is for `vX.Y.Z`. Do not publish release notes before
the workflow is green.

If the workflow fails:

1. Inspect the failed logs with `gh run view <RUN_ID> --log-failed`.
2. If the failure is clearly transient, rerun only the failed jobs with
   `gh run rerun <RUN_ID> --failed`, then watch the run again.
3. If code, tests, lint, configuration, or packaging must change, stop the
   release attempt. Fix and merge the problem through the normal development
   workflow, verify `main`, choose a new higher semver, and push a new tag.

Do not delete the failed tag or partially generated release unless the user
explicitly requests that separate cleanup.

## 5. Write and publish release notes

Once CI is green, read
[references/release-notes.md](references/release-notes.md) and draft the notes
from the full diff, commits, and relevant pull-request authors. Release notes
must explain what users can now do, not merely restate commit messages.

Publish from a notes file so shell quoting cannot corrupt Markdown:

```bash
gh release edit vX.Y.Z \
  --title 'vX.Y.Z "Codename" — Short Theme' \
  --notes-file <RELEASE_NOTES_FILE>
```

## 6. Verify the finished release

```bash
gh release view vX.Y.Z
gh run view <RUN_ID>
```

Confirm the workflow conclusion is successful, the release title and notes are
correct, and GoReleaser attached the expected artifacts. Report the version,
release URL, workflow result, and any recovery actions to the user.
