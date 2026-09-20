# Principal-owner design principles

Date: 2026-09-20. Starting commit: `166245b8db3660f88b013a59d8dacbee2afed876`.

## Decision

Create a project-local `kanban-md-principal-owner` skill for product decisions in
feature proposals, issues, and PRs. Keep it separate from implementation,
testing, output, migration, and release procedures. The skill chooses a product
direction; it does not authorize publishing a review or merging a PR.

The authoritative project copy is
[SKILL.md](../../.agents/skills/kanban-md-principal-owner/SKILL.md), mirrored for
Claude discovery at
[SKILL.md](../../.claude/skills/kanban-md-principal-owner/SKILL.md). AGENTS routes
product reviews to it; the README gives a short contributor-facing summary.
Neither copy is embedded into the skills installed into users' projects.

The user endorsed composable workflow primitives and capabilities useful across
different workflows. Those remain the starting point. They do not imply that
every feature needs two customers or that anything called generic is preferable.

## Principles

1. Compose workflows; make new domain concepts earn their place.
2. Choose the smallest capability that actually solves the need.
3. Keep the basic board complete; make workflow policy optional.
4. Give each value and action one explicit meaning.
5. Keep local files authoritative and the core independently useful.
6. Preserve contracts across interfaces, upgrades, and writers.

Reviews should return accept, reshape, defer, or decline, with the decisive
principle, concrete consequence, and next step. An issue needs evidence to choose
a direction. A PR also needs implementation evidence. A reviewer should not
recite all six rules or create a scoring exercise for a small fix.

## What changed after criticism

- New fields are not categorically wrong. Shared editing, selection,
  coordination, or validation can justify common meaning. The burden is to
  explain why a simpler representation fails.
- Generic properties are a proposed capability, not an existing escape hatch.
  Preservation, value types, sorting, missing values, and compatibility all cost
  something. A property framework must compete with a bounded alternative.
- Exact manual order represents stored intent. Sorting existing due dates or
  priorities is only a solution when that is the order the user actually wants.
  Display order must not silently control work selection.
- Optionality cannot eliminate maintenance cost. The useful promise is no
  required setup, data, workflow restrictions, or output clutter for unused
  extras. Workflow policies need opt-in; integrity and usability should not
  require users to discover an off-by-default fix.
- Configuration needs a real user choice. It is not a goal to expose every
  implementation decision as a setting.
- Explicit, bounded automation and optional integrations are compatible with a
  local board. Agent process supervision and service-specific lifecycle
  management do not become core responsibilities merely because they are useful.

The independent [critique](2026-09-20-principal-owner-principles-critique.md)
documents counterexamples to the earlier stronger formulations.

## Current behavior versus policy

The independent [source audit](2026-09-20-principal-owner-product-boundaries.md)
finds that unknown task metadata is discarded by typed rewrites, child summaries
sort direct children by ID, and the model already permits multilevel parent
links. Narrow-screen layout also already exists. A review must inspect the
specific missing operation rather than infer a missing domain concept.

The audit also identifies separate mutation paths, status timestamp differences,
load-time repairs, and a fixed-date selection documentation discrepancy. These
are static observations, not new test results. No application code or unrelated
documentation was changed to fix them in this task. The skill expressly treats
the principles as design constraints rather than claims of universal current
compliance.

## Evaluation design

The [frozen evaluation](evals/kanban-md-principal-owner/feature-review-v1/RUNBOOK.md)
is project-owned. It contains 14 independently generated future requests and
four controls. The idea generator saw the product but not the principles or
expected dispositions. A separate source audit supplied current-behavior facts.
The primary agent wrote the private acceptance envelope before evaluation.

Two fresh-context evaluators receive the first skill draft and cases in different
orders, without the private envelope, previous conversation, or each other's
answers. An independent judge checks both reasoning and decisions. A third
fresh-context evaluator tests the revised draft against the same 18 cases in a
new order, followed by a separate independent judge. Mechanical validation alone
cannot establish good product judgment.

This is a new-skill contract test with no predecessor baseline. Requests are
synthetic, not evidence of customer demand. The two passes use the inherited
runtime model, so they do not establish cross-model reliability. All effects are
simulated; no real issue, PR, board, or external service is changed by a test.

Corpus SHA-256:
`34ae85ff4e9d34d8dc3fced394d6722e68e0828ed9303d63710c7d1b5434ac07`.

## Evaluation results

The initial two passes score 17/18 under the strict reading of the storage
comparison requirement. Both correctly protect manual order and separate it
from work selection, but neither develops the alternative storage comparison.
The independent judge also records envelope ambiguity: a broader reading gives
both 18/18. This is an omitted comparison, not a proved wrong feature decision.

P2 was tightened to require comparing storage alternatives and their added
contracts before recommending model growth. The corpus and answer envelope were
not changed. A fresh full rerun of the revised skill passes 18/18 critical
semantic checks under a separate independent judge. All 54 evaluator records
pass the structured output and decision/effects checks.

The final review of manual child ordering compares parent-owned ordered IDs,
a dedicated child rank, and preserved metadata. It does not assume generic
metadata already works, or let display order silently change `pick`.

The [case-by-case report](evals/kanban-md-principal-owner/feature-review-v1/reports/summary.md)
preserves initial misses, label differences, two final diagnostic omissions, and
the limits of the evidence. In particular, the final revision has one full
fresh-context pass, not demonstrated cross-model or cross-order reliability.

## Repository verification and scope

- Ran `quick_validate.py` from the skill-creator skill on both installed copies.
- Used `cmp` to confirm both project copies match the evaluated final draft.
- Ran `validate_eval.py` with all three raw evaluator outputs and checked their
  orders against the manifest. Independent judges checked semantic reasoning.
- Checked local Markdown links, `git diff --check`, and archived-file checksums.
- Did not run Go tests, lint, or `make precommit`. This change contains only
  documentation, skill instructions, and evaluation evidence. The documentation
  commit bypasses the Go-oriented pre-commit hook; no application verification
  result is implied.

Work remains on `main`. No worktree, Kanban item, live issue comment, PR change,
release, or push is part of this task. Application behavior is unchanged.
