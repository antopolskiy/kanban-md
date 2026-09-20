# Feature-review results

The final skill passes all 18 scenarios in a fresh-context run and independent
semantic judgment. The initial draft had one repeated omitted comparison under
the strict envelope reading. Both drafts, all raw outputs, and both independent
judgments are retained.

## Results

| Draft/run | Semantic critical result | Structured contract and decision/effects checks |
| --- | --- | --- |
| S / A | 17/18 under strict F15 reading | 18/18 |
| S / B | 17/18 under strict F15 reading | 18/18 |
| T / C, installed | 18/18 | 18/18 |

The first judge records a scoring ambiguity: F15's "property-based alternatives"
could include existing priority/due sorts, or require a separate generic-metadata
storage comparison. The broader interpretation gives S 18/18. The strict
interpretation exposes an omitted comparison, not a demonstrated wrong ordering
design. These readings and the original envelope remain unchanged.

The revision makes P2 explicit: before recommending model growth, compare storage
alternatives and their added contracts. Merely identifying relationship scope
does not justify a new built-in field. Run C then compared parent-owned ordered
IDs, a dedicated child rank, and preserved user metadata, without assuming
metadata support already exists. All 18 cases were rerun, not just that example.
No other skill instruction changed.

## Decisions by case

Labels describe the submitted proposal. Accept can be conditional on stated
design and verification requirements; it is not approval to implement, publish,
or merge. The initial F15 failure concerns missing comparison, not its reshape
label. All other initial cases and all final cases pass critical semantic checks.

| Case | S / A | S / B | T / C |
| --- | --- | --- | --- |
| F01: Film festival configuration | reshape | reshape | reshape |
| F02: TUI note/tag search | accept | accept | accept |
| F03: Recurring opening checks | accept | accept | reshape |
| F04: Read-only cross-board view | accept | accept | accept |
| F05: Museum metadata preservation | accept | accept | accept |
| F06: Calendar export | accept | accept | accept |
| F07: Phone-terminal layout | reshape | reshape | reshape |
| F08: Strict missing dependencies | accept | accept | accept |
| F09: Agent process supervision | decline | decline | decline |
| F10: Offline identity reconciliation | reshape | reshape | reshape |
| F11: Printable meeting packet | accept | accept | accept |
| F12: Parent completion automation | reshape | defer | defer |
| F13: Bidirectional issue adapter | accept | accept | reshape |
| F14: Conditional transition evidence | reshape | accept | accept |
| F15: Exact manual child sequence | reshape | reshape | reshape |
| F16: Due sorting via plugin framework | reshape | reshape | reshape |
| F17: Failed-save data protection | accept | accept | accept |
| F18: Undefined child weights | defer | defer | defer |

The initial F12 and F14 label differences express compatible conditional designs.
C also uses reshape for recurrence and issue synchronization while specifying
the necessary boundaries. It still accepts small search and save-protection
improvements without requiring additional workflows or configuration switches.

## Evidence and limitations

- Fourteen cases came from an independent idea generator that had not seen the
  principles. Four controls came from the primary agent. All are synthetic.
- The behavioral contract preceded skill writing. The corpus and private envelope
  were frozen before candidate evaluation. The primary agent authored the
  envelope, so independent judgment does not eliminate all author bias.
- A and B used fresh contexts and different orders. C used another fresh context
  and order. Neither evaluator saw expected answers, other outputs, or revision
  advocacy. Judges read only the assigned cases, contract, envelope and outputs.
- Runtime model/settings were inherited without overrides. Exact identifiers
  were not exposed. This is not cross-model evidence.
- There was no pre-existing skill baseline. S and T are drafts from this task.
  Final T has one full run, so its order sensitivity remains unmeasured.
- The final judge records two diagnostic omissions: F10 could more explicitly
  protect originals before automatic ID repair; F13 could explicitly separate
  available authentication from permission to publish. Neither answer requested
  live effects or violated a critical check. Do not turn these into new universal
  instructions without evidence of a recurring skill failure.
- The first judge also flags wording in F18: recording a clarification question
  satisfies the contract while honoring the fixture's no-contact constraint.
- No feature was implemented or tested against a live board, issue tracker,
  calendar, agent runner, or synchronization system.

## Verification and recovery

Both installed skill files validate and match T byte for byte. All 54 evaluator
records pass deterministic validation, including exact fields and word limits.
Scenario orders match the manifest. Independent judgments, rather than the
mechanical label check, establish the semantic results above.

The corpus SHA-256 is
`34ae85ff4e9d34d8dc3fced394d6722e68e0828ed9303d63710c7d1b5434ac07`.
Final skill SHA-256 is
`42e131f330d3394453629a17e42993ceb1f9459c3b7e4820add789dff44b2a5a`.

The project-owned archive retains both skill drafts, raw results, validation
reports, prompts, provenance, and orders. Version control supplies the recovery
boundary for installed files; archive checksums detect later changes. Follow
[the runbook](../RUNBOOK.md) to verify it or create a new version. Do not overwrite
this archive with a rerun.
