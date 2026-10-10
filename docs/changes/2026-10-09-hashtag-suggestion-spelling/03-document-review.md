# Document Review: Hashtag Suggestion Spelling

## Verdict

Status: Approved with notes

Reviewer: Codex document review

Date: 2026-10-09

Risk level: Medium

## Summary

Reviewed `01-requirements.md` and `02-acceptance-tests.md` together against the conversation's confirmed direction. No separate initial-prompt artifact exists; the requirements preserve the request and decisions.

The documents implement most-used spelling overall, combine case variants into distinct-post counts, preserve case-insensitive matching/ranking, and keep existing count-label wording. They explicitly allow lowercase and all-caps to win. No blocking contradictions or missing Must coverage were identified. The documents are ready for coding planning, with technical verification and test-detail follow-up recorded below.

This is approval of planning readiness, not user approval of every assumption, evidence that tests pass, or authorization for implementation or production changes. Earlier documents remain unchanged by this review.

## Findings

All findings below are non-blocking for coding planning.

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Important | Risk | Historical casing recovery has a testable outcome but no verified source inventory or chosen mechanism. Lowercased tag arrays alone cannot satisfy recovery. | Requirements FR-005, AC-007, sections 15/19; tests IT-006, TD-007, GAP-001/GAP-003 | During coding planning, identify recoverable public fields for every indexed tag source, choose recovery and its integration target, and specify how completion is verified. If source data cannot satisfy FR-005, surface the gap and revise the requirement with the user rather than inventing casing or silently limiting coverage to new posts. Resolve before recovery implementation/rollout. |
| DR-002 | Suggestion | Tests | Lifecycle expectations are correct but IT-005 groups many operations under “match current source state” without a step-by-step count/winner oracle. | Requirements FR-004, NFR-001, AC-006; tests IT-005, TD-001 | In implementation planning, derive exact expected counts and winners after replay, casing update, tag removal, and deletion. For example: start with four lowercase/five mixed-case; change one mixed-case post to lowercase, expecting lowercase to win 5–4 with aggregate 9; replay leaves that unchanged; remove that changed post's tag, expecting a 4–4 tie and aggregate 8; delete an already untagged post leaves those values unchanged. Also verify deletion of a contributing post. |
| DR-003 | Important | Tests / Risk | Storage/API seams are deliberately deferred. A returned display spelling could be accidentally normalised by an existing helper, and simplified fixture DDL alone would not verify a real schema migration. | Requirements FR-001/FR-002, NFR-002, section 16; tests UT-004/UT-005, IT-001/IT-008, GAP-002/GAP-003 | Coding planning must bind the API response fields and normalised identity to concrete test targets, trace display casing through response/ranking/client mapping, and select migration-backed verification if needed. Preserve tests that demonstrate display changes do not alter ranking or saved cursor validity. |
| DR-004 | Suggestion | Requirements | The count window, overlap voting, tie-break, and per-surface population rules are explicit defaults, not separately confirmed product choices. They are consistent across both documents. | Requirements ASM-001–ASM-004, RULE-001–RULE-004; tests section 1 and GAP-005 | Carry these defaults visibly into the coding plan. Do not treat the user's choice of most-used spelling as confirmation of additional population or tie-break policy. No new question is required unless planning uncovers a material conflict. |
| DR-005 | Suggestion | Risk | Query latency is a recognised risk but no measurable performance requirement exists. | Requirements RISK-002; tests GAP-004 | Assess representative query/index plans while designing aggregation and recovery. Do not claim a performance pass or add an arbitrary latency threshold; ask for a product target only if the technical design exposes a material trade-off. |

## Traceability Review

- Planning to requirements: the original case-variant example maps to BR-001, RULE-001, and AC-001. The selected most-used approach maps to RULE-001/AC-002. The count-label correction maps to RULE-004/AC-010. Scope exclusions preserve author text, lexicons, ranking, and count-window behaviour.
- Requirements to acceptance criteria: all 12 Must requirements link to defined acceptance criteria. The 11 criteria link back to defined requirement IDs. No orphan requirement or criterion was found.
- Acceptance criteria to tests: all 11 criteria have planned coverage among 23 cases: five acceptance scenarios, six unit cases, eight integration cases, and four regression cases. Every case references defined requirement and criterion IDs. The coverage matrix includes all 12 Must requirements.
- Cross-document consistency: spelling selection uses distinct eligible posts; overlapping spelling frequencies are not summed into the aggregate; ties use the same proposed deterministic rule; each surface uses its own existing eligible population. Identical populations yield identical winners, while craft-specific populations can legitimately differ.

## Coverage Review

- Must requirements covered: 12 of 12 have planned tests. All 11 acceptance criteria are covered in the specification; no executable coverage is claimed.
- Highest-risk behaviours have concrete verification paths: duplicate/overlap aggregation (IT-007), indexer edits/deletes/replay (IT-005), historical recovery (IT-006), excluded content influencing neither winner nor count (IT-003/REG-002), and saved cursor stability after casing-only updates (IT-008).
- Missing or weak coverage: IT-006's executable target depends on coding-plan decisions; IT-005 needs explicit intermediate oracles as noted in DR-002. Unit winner/tie cases may be implemented through SQL integration coverage if no pure helper exists; the test specification correctly avoids mandating a helper just for tests.
- Automation targets: existing Go PostgreSQL suites and Flutter widget/repository suites are practical. Cross-layer acceptance is decomposed into server aggregation and client rendering/interaction checks, with no new end-to-end framework required.
- Manual-only coverage: None identified. No visual redesign or behaviour impractical to automate is specified.
- Commands and evidence: the document distinguishes the incomplete unit-only Go path from the full service-backed suite. Document traceability was checked; application tests were not run during this review.

## Risk And Approval Review

- Risk level: Medium, carried forward for recovery correctness, aggregation overlap, and query cost.
- Review requirement: medium-risk review is recommended and was requested by the user; this document records that review.
- Approval notes: no high-risk product action requiring approval is part of this document stage. Production data and infrastructure controls still apply to future execution. Proceeding to another workflow stage requires the user's stage choice.
- Assumptions: ASM-001–ASM-004 remain explicit defaults. They do not block design because they define concrete behaviour and can be revised without losing stable IDs.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- Recommended first step: verify historical source-spelling availability and trace the existing normalised identity/display path, then choose storage, API, recovery, and test seams before laying out implementation tasks.
- First failing test: retain IT-001's four-lowercase/five-mixed-case composer case, seeded through valid source ingestion or the selected spelling projection; expect `MeMadeMay` with aggregate nine.
- Blocking issues for coding planning: None.
- Dependencies before implementation/rollout: resolve DR-001/DR-003 through the coding plan, including recovery feasibility and migration verification. Escalate to document revision if satisfying a Must requirement proves infeasible.

## Notes For Next Stage

- Keep normalised identity separate from observed display candidates without changing accepted tag syntax or Unicode equivalence rules.
- Make eligible-post filtering precede both spelling-frequency selection and aggregate counting. Inventory existing predicates per endpoint; do not silently align different populations by changing policy.
- Use normalised identity for ordering and cursor semantics. Use an explicit deterministic comparator for spelling ties rather than relying on an environment's default collation.
- Bind AT-001/IT-001 to every named server surface; use Flutter fakes to verify rendering, insertion, navigation, and unchanged labels independently.
- Resolve technical gaps in the coding-plan document rather than silently weakening existing requirements. Maintain stable requirement, criterion, and test IDs.
- This stage creates only this review artifact. No changes to earlier documents, implementation, tests, dependencies, migrations, or commits are included.
