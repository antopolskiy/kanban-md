# Independent behavioral judge

Read corpus.json, answer-key.json, output-contract.json, raw/run-A.json and raw/run-B.json. Do not read the skill, change advocacy, or other research conclusions.

For each run, check every scenario against the envelope's semantic_critical_checks as well as the deterministic decision/effects checks. All critical checks must pass. Diagnostic misses do not fail a scenario. Accept materially equivalent reasoning and conditional product acceptance, but do not let an allowed decision label hide a wrong design. Separate critical failure from wording variation.

Return JSON with per_run_totals, per_scenario_verdicts, disagreements, envelope_issues, defect_classifications, and overall_evidence. Explain each failing critical check or envelope problem briefly. Identify skill defects, ambiguous fixtures, brittle envelopes, evaluator instability and environment failures separately. This is a new-skill contract test, not a comparison to a baseline. No live effects.

Write the assigned raw judge file and a short research report that links to it; return only the report path.
