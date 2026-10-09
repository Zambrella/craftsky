# Implementation Review: First Search Improvements

## Verdict

Status: Approved

Reviewer: Codex

Date: 2026-10-09

Risk level: Medium

## Summary

The matching implementation follows the approved PostgreSQL design and preserves separate Posts and Projects, current visibility predicates, original query wording and existing hydration. English forms, the three curated equivalences, authored fields, bounded single-word correction, reliable-first ranking and tuple pagination have substantive coverage. The local migration adds two pure helpers without rewriting records, adding infrastructure or handling older posts.

Re-review confirms IR-001 is resolved. The diagnostics test now seeds and reads back both private-content markers in its own isolated database fixture, searches them through both observed handlers, asserts empty pages and checks flushed serialized local/SDK output. No remaining required changes or production privacy defect were identified.

## Findings

None identified in the current implementation.

### Resolved Finding: IR-001

The initial review found that IT-012 asserted absence of private markers it never seeded. The correction adds real moderation and scheduled payloads to the captured observer's fixture and verifies both values exist before execution. Both routes search the fixture-read values and require empty items/cursors; existing serialized sink checks prohibit the markers and retain positive operation-context/nonempty-output checks. The focused privacy/boundary tests and broader required-PostgreSQL search/timeline suite passed. This resolves RULE-003 / AC-016 / IT-012 without a production code change. Execution evidence is recorded in `05-implementation-plan.md` under Review Correction.

## Requirement And Test Traceability

- Requirements implemented: BR-001; FR-001 through FR-006; NFR-001 and NFR-002; RULE-001 through RULE-004. RULE-003 verification includes the corrected same-fixture private-content sink coverage.
- Tests implemented: all 32 active planned IDs have executable coverage or shared regression evidence. IT-012 now includes the private-content sink evidence required by IR-001.
- The 31 independent corpus cases retain useful/negative records, language and rationale. Controlled ranking and pagination cases cover reliable/typo ordering independently of the top-five benchmark.
- Test placement and query-time helper refinements are explained in the implementation plan. Real PostgreSQL tests avoid a duplicate Go matching algorithm; projection convergence invokes the public Tap indexer from the API test package.
- Unplanned behavior: none identified. Tier cursors and malformed-cursor validation follow the plan. Existing request, response, route, auth, Flutter production and indexing interfaces are unchanged.
- Remaining gaps: none identified. IT-007/GAP-002 and performance obligations remain retired; they are not correction work.

## Test Evidence

- Commands reviewed: focused required-database Go tests; broader API `Search|Timeline`; `just test`; `just app-test test/search --no-pub --reporter expanded`; scoped Flutter analysis; `git diff --check`.
- Passing evidence from implementation: full backend race suite with required PostgreSQL/MinIO, all 61 Flutter search tests, 31 relevance corpus cases, focused migrated-schema visibility/convergence tests and scoped Flutter analysis.
- Review-stage validation: read workflow documents, tracked diff and new source/tests/migration/corpus; inspected correction selection, scoring, cursor ordering, privacy boundaries and client tests; reran `git diff --check`, which passed.
- Re-review inspected the corrected test and its execution notes, and reran the whitespace check successfully. Runtime suites were not repeated during re-review: the focused privacy/boundary tests and broader API Search|Timeline suite passed after the correction, with no subsequent source/test change.
- Failing or skipped tests: the new fixture-presence check initially failed because the private records were missing; after adding them, the focused and broader checks passed. No unresolved runtime failures. The prior full backend race and Flutter evidence remains applicable because this correction changes only a Go test and its notes. No performance tests or older-post upgrade harness are required.

## Risk Review

- Risk level: Medium, consistent with the approved documents.
- Eligibility is applied before vocabulary caps and pagination; correction vocabulary is recomputed independently of the cursor. User input is bound to PostgreSQL parameters. Fields are selected from current own public columns; quoted/private payloads are not included in matching SQL.
- Correction is restricted to exactly one edit of one original word, with minimum length four and eight deterministic candidates per word. Phrase candidate eligibility is checked before the cap.
- Ranking uses best alternative/path per concept and deduplicated tags; reliable results precede corrections. Keyset pagination includes tier, score, time and URI.
- Migration up/down operations are scoped to the two new helper functions. Local creation/testing was explicitly approved; production application remains outside authorization.
- No external search/model/PDS dependency, persisted search projection, backfill, new extension, lexicon, dependency or infrastructure change was introduced.
- Raw-query, correction and private-content sink coverage is substantive following resolution of IR-001.
- No commit, push or deployment was requested or performed in this review.

## UI Polish Recommendation

- Recommendation: Not needed.
- Reason: Flutter changes are test-only. The approved UI remains unchanged, including separate tabs and original query text.
- Suggested polish notes: None.

## Handoff Back To TDD Builder

- Required fixes: None.
- Suggested next failing test: None; no correction stage remains.
- Verification to rerun: None for the reviewed state. Rerun relevant checks if implementation changes.
- Ready for merge or handoff: Yes, within the approved scope. Production migration/deployment remains a separate authorized operation.
- UI polish is not needed. No commit or push was requested.
