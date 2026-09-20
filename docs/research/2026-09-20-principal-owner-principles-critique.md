# Critique of proposed product review principles

The direction is sound: prefer orthogonal, composable workflow primitives, keep a usable basic board, and let users combine capabilities for different workflows. The proposed tests are too absolute. Applied literally, they would reject useful parts of kanban-md while admitting a large, configurable workflow engine.

This review uses the current local README and implementation. It does not assess live issues or PRs. Examples of possible additions below are hypothetical.

## Where the proposed rules break

- "Primitives, not workflow nouns" judges vocabulary instead of behavior. Tasks, statuses, claims, parents, and dependencies are domain concepts. Hierarchy can support many workflows; renaming `child_rank` to `rank` does not prove reuse. Ask whether the behavior carries assumptions that other users must inherit.
- "Intrinsic, durable, broad, non-derived, invariant-enforcing" is an excessive conjunction for first-class fields. Claims expire by design. Assignees and due dates can deserve consistent editing, filtering, and display without a special invariant. First-class support may earn its place through frequent shared operations. Invariant enforcement is one strong justification, not the only one.
- Classifying facts, relations, policy, presentation, derived data, and automation helps locate responsibilities, but the categories overlap. Status is stored state; terminal status configuration affects interpretation; completion counts are derived; their visibility is presentation. Classification does not decide whether the combined feature is worth adding.
- "Unused capabilities impose zero cost" cannot be met. Optional behavior still costs documentation, compatibility, tests, and maintenance. The useful promise is no compulsory setup, clutter, or workflow restrictions for users who do not enable it. Review the remaining costs explicitly.
- "Configure policy, fix invariants" can justify unlimited configuration. It can also disguise an opinion as an invariant. Acyclicity follows from choosing a hierarchy; requiring every child to finish before its parent is a workflow decision. Define the meaning first. Expose a policy setting only when variation has a demonstrated need.
- "Composition over named modes" becomes harmful if it excludes useful conveniences. `pick --claim ... --move ...` combines selection, claiming, and movement in one operation. Named presets can also help users start. The objection should be to separate semantics and hidden coupling, not to names or bundled commands.
- "Two materially different demonstrated use cases" is a useful stress test, not an acceptance quota. Two weak examples do not justify a rules engine, while one frequent, costly coordination problem may justify a focused capability. Requiring every addition to lower complexity is also impossible if it means reducing the codebase. Compare total user and maintenance cost with realistic alternatives.
- "Automation informs before it controls" conflates unsolicited control with requested execution. Informational child completion counts are a good default because parents have independent meaning. A user may still explicitly request automatic transitions. `pick` already acts on a user's request. Judge whether the action is explicit, predictable, and consistent with its declared scope.

## Six decision rules

1. **Start with the user problem.** Name the job, its cost today, and why it belongs in a local task board. Prefer an existing command, composition, or recipe when it solves the job adequately.
2. **Choose the smallest reusable behavior.** Prefer capabilities that combine across workflows. Use a different workflow to challenge the design; do not build a general framework merely to avoid one specific feature.
3. **Make semantics earn their place.** Add a built-in field or rule when shared operations or correctness need a common meaning. Keep project-specific meaning in user data or configuration. Explain why the simpler representation fails.
4. **Keep combinations predictable.** Each capability needs a clear responsibility and the same meaning across commands and views. Shortcuts may combine capabilities; incidental display choices must not silently become execution policy.
5. **Keep the basic board usable.** Supply sensible defaults. Extras should require no setup when unused, and workflow-specific restrictions or automatic changes should require an explicit choice.
6. **Count the whole cost.** Compare user effort, discoverability, maintenance, compatibility, and interactions. Prefer the smallest change that meets the need, and state what evidence would justify expanding it later.

Use these in order, with product scope and existing data contracts as gates. After establishing a real benefit, compare complete alternatives. Reuse matters when it reduces repeated concepts and work; it does not excuse poor defaults or unlimited configuration. A small increase in implementation complexity can be justified by a substantial reduction in users' recurring work.

## Concrete decisions the rules support

| Proposal | Decision and reason |
| --- | --- |
| Add `child_rank` as a built-in task field solely to choose how siblings appear | Prefer a reusable sortable property and an explicit view order if that mechanism adequately covers the need. There is no demonstrated need for the core to own sibling-specific ranking semantics. Hierarchy itself is not the objection. |
| Treat generic properties as the automatic replacement for `child_rank` | Require a bounded design first. Properties introduce preservation, editing, value types, sorting, missing values, and output contracts. A general expression language is not justified by the desire to order children. |
| Let a display sort control which task an agent picks | Reject implicit coupling. Reading order and execution order are separate user intentions. A reusable ordering mechanism can serve both through explicit settings. |
| Replace claims with arbitrary metadata because claims are temporary or workflow-specific | Reject. Expiration, selection, and claim-aware mutations rely on shared meaning. A plain metadata value does not supply that behavior. |
| Add a reusable preset that assembles existing statuses and options | Accept if it saves meaningful setup and remains ordinary editable configuration. Reject a preset that creates a separate hidden ruleset. |
| Automatically finish every parent once its children finish | Reject as a universal default. A parent may have its own acceptance work. An explicit automation could be considered on its own benefit and cost. |
| Add a fully configurable transition engine, justified by two workflows | Two examples are insufficient. Compare ordinary commands, dependency checks, and a documented recipe before accepting a new language and its interaction burden. |

The sortable-property alternative is a direction, not an existing capability to assume. The current task struct contains only known fields, and the reader and writer deserialize and serialize that struct. Arbitrary fields therefore do not currently have a preservation contract. Current sorting also switches over a fixed set of fields. Any properties proposal must account for those changes before its cost can be compared honestly with a dedicated field.

## Local evidence

- [README](../../README.md) documents configurable statuses and priorities, independently managed parent status, derived child counts, cooperative claims, combined `pick` operations, and the existing product principles.
- [Task representation](../../internal/task/task.go) shows the current built-in fields, including claims, assignee, due date, parent, and dependencies.
- [Task file handling](../../internal/task/file.go) shows deserialization and serialization through the task struct.
- [Sorting](../../internal/board/sort.go) shows the supported field switch and configuration-based status and priority ordering.
- [Task selection](../../internal/board/pick.go) shows selection semantics that extend beyond display sorting.
