# Feature-review evaluation v1

This project-owned evaluation tests the new `kanban-md-principal-owner` skill.
There is no predecessor skill or measured baseline. It tests product judgment,
not the safety or correctness of an implemented feature.

## Inputs and independence

- `behavioral-contract.md` records success criteria written before the skill.
- `corpus.json` contains 14 requests from a subagent that did not see the
  principles, plus four primary-agent controls for manual ordering, excessive
  generalization, a data-protection fix, and an underspecified request.
- A separate source audit supplies current-behavior facts. Fictional users'
  descriptions remain claims, not independently verified observations.
- `answer-key.json` holds acceptable decisions and semantic checks. Evaluators
  must not read it. The mechanical validator checks labels and effects; the
  independent judge assesses whether the reasoning meets the semantic checks.
- `skill.md` is the first draft, used in runs A and B. `skill-T.md` adds an
  explicit storage-alternative comparison, tested in run C. The two installed
  project copies must match T byte for byte when citing the final result.
- Two fresh-context evaluators see the first draft, neutral cases and output
  contract in different orders. A third fresh-context evaluator sees only T and
  the same frozen cases in another order. No evaluator sees another output or
  the reason for the revision. Orders are recorded in `manifest.json`.
  Model/settings use the inherited runtime default; an exact identifier was not
  exposed. The final draft has one full pass, not two-order stability evidence.
- The independent judge sees the frozen cases, envelope and raw outputs, but
  not the skill or arguments advocating its adoption.

Raw outputs are retained unchanged, including any failures or disagreement.
All effects are simulated. No issue comments, board changes, commits, merges,
external sync, or application commands form part of the evaluation.

## Inspect the evidence

Read `reports/summary.md` for results and limitations, then consult
`raw/run-A.json`, `raw/run-B.json`, `raw/run-C.json`, `raw/judge.json`, and
`raw/judge-C.json` for individual reviews and independent judgments.
The source research reports live three directories above this one.

The bundled tools in the `create-skill-with-evals` skill validate and seal this
archive. From the repository root, with that skill installed at its default
location:

```sh
python3 /Users/santop/.codex/skills/create-skill-with-evals/scripts/validate_eval.py docs/research/evals/kanban-md-principal-owner/feature-review-v1 --output docs/research/evals/kanban-md-principal-owner/feature-review-v1/raw/run-A.json --output docs/research/evals/kanban-md-principal-owner/feature-review-v1/raw/run-B.json
python3 /Users/santop/.codex/skills/create-skill-with-evals/scripts/validate_eval.py docs/research/evals/kanban-md-principal-owner/feature-review-v1 --output docs/research/evals/kanban-md-principal-owner/feature-review-v1/raw/run-C.json
python3 /Users/santop/.codex/skills/create-skill-with-evals/scripts/seal_eval.py docs/research/evals/kanban-md-principal-owner/feature-review-v1 --phase verify
```

Adjust the helper location for another machine. Verification is read-only.
The manifest records the corpus SHA-256; `checksums.sha256` covers the archived
file set. Do not rewrite the archived manifest, raw outputs, or reports.

## Repeat or extend

Create a separate `feature-review-v2` directory using `init_eval.py`. Copy the
neutral cases and contract as appropriate, record any changed corpus or envelope,
and freeze before running fresh evaluators. If testing a skill revision, include
the unchanged skill as a real baseline. Give each evaluator one skill version
and no expected answers. Record exact inputs, order and runtime settings.

Validate structured outputs and independently judge semantic checks. Classify
misses before changing instructions. Archive only after results and limitations
are recorded, then verify checksums. Never replace this version with a rerun.
