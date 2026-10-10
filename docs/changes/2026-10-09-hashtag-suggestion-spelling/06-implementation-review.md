# Implementation Review: Hashtag Suggestion Spelling

## Verdict

Status: Approved

Reviewer: Codex

Date: 2026-10-09

Risk level: Medium

## Summary

The implementation satisfies the revised requirements. Case-insensitive identities combine counts while each surface selects the most-used observed spelling from its own eligible posts. Query ranking, pagination, lifecycle, count labels and composer insertion retain their intended behavior.

The user explicitly approved rejecting ambiguous recognized source fields. FR-006/AC-012 and the coding-plan correction now record that exception to the original source-parity contract. The shared raw-token checker rejects conflicting aliases and duplicate exact keys before projection; the historical migration fails visibly on detectable conflicts; a database constraint protects writes that bypass application validation.

No remaining implementation blocker was identified. This review approves local implementation readiness, not deployment or execution of the production migration. Only this review artifact was changed during review.

## Findings

None identified.

Prior finding disposition:

- IR-001: resolved. SQL and Flutter use the generated Go Unicode lowercase mapping for identity comparisons. Real-indexer and pagination regressions cover an actual platform normalization disagreement.
- IR-002: resolved for the accepted source contract. Single case-folded aliases are preserved across post facets, structured tags, pattern fields and material fields; migration recovery and all discovery stores are covered.
- IR-003: resolved through the explicitly approved validation policy. Both input orders, same/different identities, identical values, escaped aliases and duplicate exact keys are rejected. Historical conflicts block recovery without automatic repair or omission. The former claim that the defect was solely historical has been superseded in the execution notes.

## Requirement And Test Traceability

- Requirements implemented: all 13 Must requirements, BR-001, FR-001–FR-006, RULE-001–RULE-004 and NFR-001–NFR-002, have mapped implementation and test evidence.
- Tests implemented: the original 23 test IDs remain covered at their documented seams. UT-007 covers raw source validation under FR-006/FR-004; IT-009 covers historical ambiguity, failure rollback and explicit fixture resolution under FR-006/FR-005; IT-010 covers direct indexing, database safeguards and durable ingestion under FR-006/FR-004/FR-002.
- Contract alignment: requirements, acceptance tests and coding plan explicitly record the user-approved exception. The generated spelling projection remains appropriate for unambiguous sources. Unknown fields and future feature payloads retain best-effort handling; one alias and multiple independent hashtag features remain valid.
- Existing behavior: a rejected direct indexer call leaves the existing post intact. An invalid authoritative update through the transactional ingestion pipeline clears prior serving state under the existing policy while retaining its durable source. Tests distinguish these paths rather than introducing a new invalid-update policy.
- Unplanned behavior: none identified within the revised contract. No route, wire shape, lexicon, OAuth, permissions, count wording or UI layout change was introduced by the correction.
- Remaining gaps: no required local test gap. Historical JSONB cannot expose exact duplicate keys already overwritten; this information limit is explicit in the revised requirements. The historical gate detects surviving conflicting aliases and does not claim universal original-token reconstruction.

## Test Evidence

- Implementation evidence reviewed: the final full `just test` service-backed Go race run passed in one run with exit 0, including PostgreSQL and MinIO. This supersedes the earlier split passing evidence and accurately retains the earlier intermittent MinIO failure in the execution history.
- Permanent correction tests: raw validation covers 24 conflicting-field cases on both create and update, with positive single-alias/unknown-payload/delete cases. Historical migration coverage has 26 fixtures across recognized source boundaries. Direct-write and durable-ingestion integration tests exercise the actual database and production indexer/dispatcher.
- Review rerun: `TEST_DATABASE_URL=<local development PostgreSQL> TEST_DATABASE_REQUIRED=true go test ./internal/db ./internal/index -run 'TestHashtagSpellings|TestSourceValidatorRejectsAmbiguousHashtagFields|TestCraftskyPost_(RejectsAmbiguousTagSources|DurableAmbiguityRejection|HashtagFieldCasing|HashtagUnicodeSpelling|HashtagSpellingLifecycle)' -count=1` passed; database 4.875 s, indexer 6.287 s.
- Independent review diagnostic: a temporary program outside the repository applied the exact migration in a rolled-back local PostgreSQL transaction. All 14 cases agreed between the raw Go checker, shared source validator and SQL ambiguity checker: canonical fields, single aliases, Unicode field folding, independent features, unknown root/feature fields, conflicting identities, reversed order, spelling-only conflicts, identical values, escaped aliases, Unicode index conflicts and structured tags. No repository source/tests or production records were changed.
- Flutter evidence reviewed: seven suites, 59 tests, and targeted analysis passed in the earlier correction. They were not rerun for this backend-only correction/review because those files are unchanged.
- Formatting and `git diff --check` are clean. The full Go suite was not repeated during review because no implementation files changed; the focused rerun addresses the corrected behavior directly.

## Risk Review

- Risk level: Medium, for migration locking/cost and historical source integrity.
- Migration controls: the preflight reports a clear failure before enabling the spelling column, preserves source rows and rolls back schema work. A CHECK constraint prevents surviving conflicting aliases on later writes; down migration removes the column, constraint and helper in dependency order. New raw exact duplicates are caught before JSONB conversion.
- Source handling: the ambiguity checker is scoped to recognized hashtag-source fields and boundaries. It does not replace lexicon validation or reject unrelated unknown payloads. Errors contain a fixed reason without source values; no diagnostic logging was added.
- Durable ingestion: Tap retains its source in JSON, preserving duplicate keys and field order for worker revalidation. The tested invalid-update behavior changes only the serving projection; no PDS record is deleted or rewritten.
- Cost evidence: the final 20,000-row rolled-back fixture measured preflight 1,269.811 ms, constraint validation 1,219.252 ms and generated-column addition 2,929.160 ms, about 5.42 s combined. Query execution was 76.452 ms, planning 0.415 ms, without temporary-file spill. These local observations do not establish production lock duration or latency.
- Rollout notes: production integrity and volume remain unverified. Any detected historical conflict requires explicit resolution before migration retry. JSONB cannot recover overwritten exact duplicate keys; retained authoritative source may assist investigation where available. Review of production migration scope and duration remains normal release work.
- Approval notes: local changes are uncommitted and undeployed. This review authorizes no production inspection, migration, deployment, commit or push.

## UI Polish Recommendation

- Recommendation: Not needed.
- Reason: existing rows, chips, labels and navigation are reused; no visible polish issue was identified.
- Suggested polish notes: None.

## Handoff Back To TDD Builder

- Required fixes: None.
- Suggested next failing test: None required for the approved scope. Future changes to recognized tag sources should update Go and SQL ambiguity rules together and extend boundary/parity coverage.
- Verification to rerun: rerun relevant gates if implementation changes after this review. Before production rollout, review source integrity and migration cost against the intended deployment data; no production action is part of this review.
