# Release Notes Guide

Use this guide only after the tag-triggered release workflow succeeds and
GoReleaser has created the GitHub release.

## Research the release

Compare the new version with the previous release and inspect enough pull-request
context to identify outside contributors accurately:

```bash
git diff vPREVIOUS..vNEW --stat
git log vPREVIOUS..vNEW --oneline
gh release list --limit 100
```

Check the full release-title history before choosing a codename. Codenames must
be unique across all kanban-md releases.

## Title

Use:

```text
vX.Y.Z "Codename" — Short Theme
```

The codename is a unique, evocative one- or two-word mnemonic loosely related to
the release theme, such as "Quiet Storm", "Paper Trail", or "Red Line". Keep the
short theme concrete and user-facing.

## Body

Start with a one- to three-sentence TL;DR paragraph without a heading. Then use
only the section types that apply:

- `## New:` for new commands and capabilities.
- `## Changed:` for behavior changes and migration-relevant differences.
- `## Fixed:` for user-visible bug fixes.

Write from the user's perspective: explain what they can do now, why it matters,
and anything they must change. Do not turn the commit log or file list into the
release notes.

Add attribution in a section heading only when someone other than the maintainer
or release author materially contributed to that item:

```markdown
## New: Live TUI search (by @github-handle)
```

Use multiple handles when multiple outside contributors materially shaped the
same item. Verify handles from the relevant pull requests rather than guessing.

Every new-feature section must include one or two practical command examples:

````markdown
## New: Feature name

Explain what it does and why it matters.

```bash
kanban-md <command> <flags>
```
````

End with an upgrading section and full comparison link:

```markdown
## Upgrading

Describe required steps. If none are required, say "No action needed" and
briefly explain automatic migration or backward compatibility.

**Full diff:** [`vPREVIOUS...vNEW`](https://github.com/antopolskiy/kanban-md/compare/vPREVIOUS...vNEW)
```

Before publishing, verify the version numbers, comparison URL, contributor
handles, examples, migration guidance, codename uniqueness, and Markdown
rendering.
