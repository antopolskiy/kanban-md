# Hierarchy child-ordering models

**Date:** 2026-08-24
**Question:** Should kanban-md add PR #21's first-class `child_rank` task field, or model child ordering with less task-domain creep?

## Recommendation

Do not merge PR #21 as-is. The use case is valid, but the proposed boundary is not yet convincing.

`child_rank` is described as affecting only detail-view presentation. Persisting a presentation-only choice as a public task property, exposing its sparse integer representation in YAML, CLI flags, table/compact output, and JSON, and defining reparenting behavior turns a narrow view preference into permanent task-domain semantics.

Prefer this sequence instead:

1. Add child ordering as a **view/sort policy over existing task facts**, with task ID as the compatibility-preserving default. Reuse the existing sort vocabulary where possible (`priority`, `due`, `created`, `updated`, `title`, `id`) and consider a dependency-aware topological option.
2. Keep exact arbitrary manual ordering out of the first version. Measure whether derived ordering solves the real boards behind #20.
3. If exact shared order remains necessary, design a **general ordered-collection capability**. Expose relative operations such as `child move --before/--after`, not numeric ranks. Treat the storage token or sidecar representation as an implementation detail and decide explicitly whether order belongs to a view, a parent-child relation, or workflow sequencing.

This is not a claim that rank-like storage is always wrong. It is a claim that kanban-md should not introduce a child-specific public ranking primitive before deciding which of those three meanings it intends.

## What PR #21 models

PR #21 adds an optional positive integer to every task:

```yaml
parent: 201
child_rank: 20
```

The field:

- orders a parent's direct children in detail views;
- is set and cleared through dedicated `create` and `edit` flags;
- is added to task JSON and to the `children` objects returned by `show --json`;
- is rendered in table and compact output;
- requires a parent, is cleared when the parent is cleared, and survives reparenting;
- deliberately does not affect `pick`, priority, WIP limits, or board/list sorting.

The implementation is internally coherent and file-friendly: each child carries its own sparse rank, so reordering one child normally changes one task file. The concern is conceptual scope, not whether the sorting function works.

The core mismatch is this: the PR explicitly defines rank as presentation-only, but stores and exposes it as task-domain data. It also stores a property of the parent-child edge on the child node. That works while a task can have only one parent, but it bakes the current relationship cardinality into the public task schema.

## How comparable systems model ordering

### GitHub sub-issues: ordered relation, relative command

GitHub's sub-issue API reprioritizes an item within a specific parent's sub-issue list. The caller supplies the sub-issue plus either `before_id` or `after_id`; the API does not ask callers to calculate a public numeric rank. This models order as part of the parent-child collection and keeps the position representation opaque.

Source: [GitHub REST API endpoints for sub-issues](https://docs.github.com/en/rest/issues/sub-issues?apiVersion=2022-11-28#reprioritize-sub-issue).

### GitHub Projects: view state and field-derived sorting

GitHub Projects separates issues from project views. A view can save sorting, reordering, filtering, and grouping without adding those choices to the underlying issue. In a board view, applying a field sort disables manual reordering within a column, making the distinction between derived sort and manual position explicit.

Sources: [Managing project views](https://docs.github.com/en/issues/planning-and-tracking-with-projects/customizing-views-in-your-project/managing-your-views), [Customizing the board layout](https://docs.github.com/en/enterprise-cloud@latest/issues/planning-and-tracking-with-projects/customizing-views-in-your-project/customizing-the-board-layout#sorting-by-field-values).

### Linear: mostly display options, with a separate shared manual order

Linear treats the ordering of sub-issues shown under a parent as a display option; changing that `Order by` choice is user-local. More broadly, Linear views can order by status, priority, dates, and other existing properties. It also offers a distinct `Manual` ordering mode; unlike ordinary view preferences, moving items in manual mode updates a workspace-wide manual order.

This is a useful split: sort policy is view state, while exact collaborative manual order is a separate, explicit capability.

Sources: [Linear parent and sub-issues](https://linear.app/docs/parent-and-sub-issues), [Linear display options](https://linear.app/docs/display-options).

### Notion: order is normally a property-based view configuration

Notion stores sorting in a database view and can keep it personal or save it for everyone. Multiple sorts can be composed, and the same underlying items can appear in differently ordered views. Ordering is not automatically promoted to a property on every database item.

Source: [Notion database views, filters, sorts, and groups](https://www.notion.com/help/views-filters-and-sorts#sorts).

### Jira: general rank, relative API

Jira does make Rank a field, but it is a general backlog/board ordering facility rather than a child-specific property. Its API still expresses mutation relationally: move issues before or after another issue, optionally selecting a rank custom field. Users manipulate relative order, not sparse integers.

Sources: [Jira rank a work item](https://support.atlassian.com/jira-software-cloud/docs/rank-an-issue/), [Jira issue-rank REST API](https://developer.atlassian.com/cloud/jira/software/rest/api-group-issue/#api-rest-agile-1-0-issue-rank-put).

### Trello: generic list position

Trello cards carry a generic `pos` within a list, and creation APIs accept intent such as `pos: "top"`. Position is scoped to the containing list and supports the product's core manual-board interaction; it is not a child-hierarchy-specific concept.

Source: [Trello client API introduction](https://developer.atlassian.com/cloud/trello/guides/client-js/getting-started-with-client-js/#create-cards-and-update-information).

### Org mode: source order is canonical, and sequencing is explicit

Org mode gets sibling order from physical outline order. When order should affect workflow rather than just display, the parent can be marked `ORDERED`, which blocks each TODO child until earlier siblings are complete. This is an important contrast: ordered presentation comes from source structure; ordered execution is an explicit domain rule.

Source: [Org mode manual, TODO dependencies](https://orgmode.org/org.pdf#page=64).

## The recurring patterns

Comparable tools use four broad models:

1. **Derived view sort.** Sort by existing task fields. This adds no per-task state and supports multiple views, but cannot express an arbitrary sequence when tasks otherwise compare equal.
2. **Opaque collection position.** Store manual position in a view, list, or parent-child collection and expose relative move operations. This supports exact order but requires persistent state somewhere.
3. **Intrinsic source order.** The order of items in one parent document is canonical. This works naturally for outlines but not for kanban-md's current one-file-per-task layout.
4. **Workflow sequencing.** Use dependencies or an ordered-execution rule when order means “do this before that.” This is domain semantics, not presentation.

There is no model that provides arbitrary, shared, stable manual order without additional state. Avoiding a task property means either deriving order from existing properties or moving the state to a view/collection store.

## Alternatives for kanban-md

### A. Reuse existing sort semantics — recommended first step

Allow detail children and hierarchy-tree siblings to use an existing sort field, while retaining `id` as the default:

```bash
kanban-md show 201 --child-sort title
kanban-md show 201 --child-sort priority --reverse
```

The TUI could inherit its current sort field or have a view-level child-sort option. Internally, kanban-md already sorts tasks by `id`, `title`, `status`, `priority`, `created`, `updated`, and `due`; this extends an existing capability instead of growing the task schema.

Advantages:

- no task-file or JSON schema change;
- different consumers can choose different orderings;
- semantics generalize beyond hierarchy;
- simple, predictable Git behavior.

Limitation: it cannot represent an arbitrary editorial sequence.

### B. Derive execution order from dependencies — recommended option

Add a dependency-aware ordering mode for siblings: topologically order dependencies between children, then use a configured deterministic tie-break such as priority and ID.

This reuses an existing domain fact when “reading order” really means “implementation order.” It also benefits from PR #25 preventing new dependency cycles.

Advantages:

- no new task semantics;
- order stays synchronized with actual blockers;
- generalizes to other dependency views.

Limitation: unrelated siblings remain only partially ordered, and editorial order is not always dependency order.

### C. Store hierarchy order in view metadata

A sidecar could hold ordered child IDs by parent:

```yaml
parents:
  201: [202, 208, 205]
```

This keeps task frontmatter clean and correctly classifies the data as presentation state.

Advantages:

- exact shared manual order;
- no public task property;
- potentially supports multiple named hierarchy views.

Costs:

- creates a central merge-conflict hotspot;
- must repair stale IDs after delete, archive, or reparent;
- reordering and reparenting become multi-file operations;
- introduces a second persistence subsystem for a small feature.

For a Git-native, one-task-per-file tool, these operational costs are significant.

### D. Put an ordered child list on the parent — not recommended

An explicit `children: [...]` array makes source order obvious and resembles an outline. However, `parent` would then be represented twice: once on the child and once on the parent. Every reparent/delete must update multiple files atomically, and hand edits can disagree. Making the parent list authoritative would require a larger hierarchy-model redesign.

### E. General manual-order capability with relative commands — future option

If arbitrary collaborative order proves essential, expose intent rather than representation:

```bash
kanban-md child move 208 --before 205
kanban-md child move 208 --after 202
```

The same abstraction could later support moving tasks within status columns or other ordered collections. Storage could be an opaque fractional token on the relation or a sidecar index, but users and JSON consumers should not need to manage sparse integer ranks.

This follows the GitHub/Jira pattern and avoids committing to `10, 20, 30` as part of the public contract. It still adds persistence semantics, so it should follow—not precede—a decision about ordering scope.

### F. Rename `child_rank` to generic `rank` — not recommended

A generic task rank sounds reusable but becomes ambiguous immediately: rank within which status, parent, filtered view, or project? A task appears in multiple collections simultaneously. Generalizing the name without scoping the value makes the domain model less precise, not more.

## Decision matrix

| Model | New task field | Exact manual order | Multiple views | Git/file behavior | Conceptual fit |
|---|---:|---:|---:|---|---|
| Existing-field sort | No | No | Yes | Excellent | View concern |
| Dependency-derived | No | Partial | Yes | Excellent | Workflow concern |
| View sidecar | No | Yes | Yes | Central conflict/drift risk | View concern |
| Parent child list | Parent field | Yes | No | Multi-file mutation risk | Aggregate concern |
| PR #21 `child_rank` | Yes | Yes | One fixed order | Local edits are good | Presentation stored as task data |
| Relative ordered-collection capability | Implementation-dependent | Yes | Potentially | Design-dependent | Generalizable if scoped |

## Suggested disposition of PR #21

Thank the contributor and keep the use case, but ask to pause the implementation while the product boundary is revised. The most useful next experiment is a smaller PR that:

1. leaves task frontmatter unchanged;
2. adds child sorting using the existing sort vocabulary;
3. optionally adds dependency-aware ordering;
4. applies consistently to CLI detail output and the TUI hierarchy tree;
5. preserves ID order by default.

If users still need exact arbitrary ordering after that, open a dedicated design issue for ordered collections and evaluate a relative `before`/`after` interface. That provides a path to the capability without prematurely freezing `child_rank` into every task and output contract.
