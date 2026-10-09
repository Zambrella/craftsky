# TDD Implementation Plan: First Search Improvements

## Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Review: `03-document-review.md`
- Coding plan: `04-coding-plan.md`

## Implementation Rules

- Link every behavior to requirements; one focused failing test before each implementation increment.
- Run focused tests first; refactor only while green.
- User approved local helper migration creation/testing and retirement of IT-007/GAP-002. No older-post backfill/upgrade harness, production operations or commits.

## Progress

- Read workflow documents: completed.
- Create execution plan and inspect code: completed.
- Final verification and completion notes: completed.

## Test Order

| Step | Test ID | Requirement IDs | Acceptance Criteria | Status |
|---|---|---|---|---|
| 1 | IT-001 | FR-001 | AC-002, AC-003 | completed |
| 2 | UT-001 | FR-001 | AC-002, AC-003 | completed |
| 3 | UT-002 | FR-001 | AC-003 | completed |
| 4 | AT-002 | FR-001, RULE-004 | AC-002, AC-017 | completed |
| 5 | UT-003 | FR-002 | AC-004 | completed |
| 6 | IT-002 | FR-002, FR-004 | AC-004, AC-007, AC-008 | completed |
| 7 | AT-003 | FR-002, FR-004 | AC-004, AC-007, AC-008 | completed |
| 8 | IT-004 | FR-004 | AC-007, AC-008 | completed |
| 9 | UT-004 | FR-003 | AC-005, AC-006 | completed |
| 10 | UT-007 | NFR-002 | AC-012 | completed |
| 11 | IT-003 | FR-003 | AC-005, AC-006 | completed |
| 12 | IT-009 | NFR-002 | AC-012 | completed |
| 13 | AT-004 | FR-003 | AC-005, AC-006 | completed |
| 14 | IT-010 | RULE-002 | AC-015 | completed |
| 15 | AT-006 | RULE-001, RULE-002 | AC-014, AC-015 | completed |
| 16 | REG-003 | RULE-002 | AC-015 | completed |
| 17 | UT-005 | FR-005 | AC-009 | completed |
| 18 | IT-005 | FR-005 | AC-009 | completed |
| 19 | UT-006 | FR-006 | AC-010 | completed |
| 20 | IT-006 | FR-006 | AC-010 | completed |
| 21 | AT-005 | FR-005, FR-006 | AC-009, AC-010 | completed |
| 22 | IT-008 | NFR-001, RULE-003 | AC-011, AC-016 | completed |
| 23 | IT-011 | RULE-003 | AC-016 | completed |
| 24 | IT-012 | RULE-003 | AC-016 | completed |
| 25 | IT-013 | RULE-004 | AC-017 | completed |
| 26 | REG-001 | RULE-001 | AC-014 | completed |
| 27 | REG-005 | RULE-004 | AC-017 | completed |
| 28 | IT-014 | NFR-001, RULE-002 | AC-011, AC-015 | completed |
| 29 | AT-001 | BR-001 | AC-001 | completed |
| 30 | AT-007 | RULE-004 | AC-017 | completed |
| 31 | REG-002 | RULE-001 | AC-014 | completed |
| 32 | REG-004 | RULE-004 | AC-017 | completed |

IT-007: cancelled by user; older-post upgrade coverage unnecessary. Order mirrors coding-plan groups with each ID tracked individually. Existing regression tests may already pass; no artificial red failure.

## Implementation Steps

### Step 1: IT-001
- Write failing test: `TestSearchImprovementsEnglishWordForms`, real PostgreSQL, both tabs, declared-English and unknown-language contrast.
- Run command: focused Go API test with required local test database.
- Confirmed failure: eligible English sock/knit posts were absent with the original simple-only SQL. Initial setup failure corrected by seeding normal image-safety clear state.
- Implement: minimum English form matching.
- Focused EnglishWordForms test passes for both tabs. Interim English path added beside literal matching; This interim path was subsequently replaced by the shared per-concept pipeline while green.

## Completion Checklist

- [x] All Must requirements covered by tests or documented gaps
- [x] All planned Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Docs updated
- [x] Review completed or explicitly skipped

### UT-001, UT-002, AT-002
- Status: completed
- Evidence: NaturalWording and LanguageAndLiteralNames passed on real migrated PostgreSQL. Natural wording was already green; literal/metadata baselines remain green. Only how/to explicitly removed in planner.

### UT-003
- Status: completed
- Evidence: CraftEquivalents failed in all twelve query/tab directions with original SQL; passes after finite nonrecursive concept planning and local score helper. English/filler/literal tests remain green.

### IT-002, AT-003
- Status: completed
- Evidence: CrossFieldConcepts post case failed before tags/alt support and passed afterward. Project fixture corrected to put blue in caption, wool in materials and socks in title. CraftEquivalents covers all bound AT-003 directions; phrase cannot span unrelated fields. Existing disjoint/rank/pagination regression passes.

### IT-004
- Status: completed
- Evidence: ScopedFields passes all eleven independent post/project field cases, including null/missing alt values; this was baseline verification after the cross-field increment.

### UT-004
- Status: completed
- Evidence: OneEditBoundary failed with missing SQL function, then passed all insertion/deletion/substitution/transposition, Unicode, zero-edit and multi-edit cases after the narrow helper.

### UT-007
- Status: completed
- Evidence: CandidateCap was red with no corrected results; passes after viewer/tab/other-concept eligible vocabulary, distinct C-order cap of eight, single replacement plans and reliable/typo tiers. Tuple cursor support lands with the pipeline to keep retrieval internally coherent; detailed cursor coverage follows.

### IT-003, AT-004
- Status: completed
- Evidence: TyposAndProtectedCodes passes both tabs: four edit types, Flaxx Light, zero/two-word/unrelated negatives, DK/K2/KAL and reliable-before-corrected with and without reliable records. Green baseline after bounded pipeline increment.

### IT-009
- Status: completed
- Evidence: RequestAndResultBounds passes 256/257 Unicode runes, limits 100/101/0 and complete 101-record traversal. Existing limits preserved.

### IT-010, AT-006, REG-003
- Status: completed
- Evidence: VisibilityBeforeCandidates passes both tabs and eight real policy states: mute, block each direction, hide/takedown, terminal owner, language and blocked image safety. Hidden words cannot consume candidate cap. Own/no-preference language exceptions and page traversal retained. Setup updated for real source/aggregate foreign keys.

### UT-005, IT-005
- Status: completed
- Evidence: Ranking passes both tabs: primary before tags before alt, reliable before strong corrected, descending time/URI, pattern primary weight, duplicate-tag and like/repost immunity. SQL-owned unit case covered through integration test.

### UT-006
- Status: completed
- Evidence: CursorValidation passes roundtrip and malformed/missing tier, fractional/invalid tier, score, date, URI and query/kind binding. Cursor tuple was implemented with typo pipeline, so these are green preservation assertions.

### IT-006, AT-005
- Status: completed
- Evidence: Pagination passes reliable-only, synonym, typo-only and mixed tiers at limits 1/2/3, twice per dataset, with exact URI order and dual qualification once. Handler wrong-query/tab cursors return invalid_cursor.

### IT-008
- Status: completed
- Evidence: ReadOnlyRetrieval succeeds with default_transaction_read_only=on across literal, English, synonym, typo, empty and added-field queries; full published-row and history snapshots unchanged. Matching wiring has no external search/model/PDS client; outbound absence is code-boundary review, not a fabricated injectable spy.

### IT-011
- Status: completed
- Evidence: PrivateAndQuoteBoundaries passes real moderation snapshot/scheduled payload markers, own-field quote matching, quote-body isolation, project-quote/reply exclusion and disjoint tabs. Scheduled fixture includes owner generation.

### IT-012
- Status: completed
- Evidence: SerializedPrivacy passes actual slog JSON and flushed Sentry MockTransport events for both routes, success/empty/validation/DB-failure paths; query/correction canaries absent, permitted search operation context retained. No diagnostic production changes.

### IT-013, REG-001, REG-005
- Status: completed
- Evidence: HandlerContract passes both handlers, real saved/like hydration with active member fixtures, camelCase success/errors, invalid bounds/cursors, missing-DID handler behavior and zero recent-search rows. PrivateAndQuoteBoundaries supplies tab/reply/own-quote regression; full middleware and recent-search suites remain final verification.

### IT-014
- Status: completed
- Evidence: ProjectionConvergence passes real public Tap create/update/replay/delete for both types, including captions, hashtag facets, image alt, project title/material/tags; obsolete markers disappear, replay stays single URI/CID, delete removes result. No indexer production edits.

### UT-007
- Status: completed
- Evidence: PhraseCandidateCap was meaningfully red because incomplete phrase records consumed all slots. It passes after enforcing full reconstructed-concept eligibility before cap; ordinary cap, typo and equivalence regressions also green.

### AT-001
- Status: completed
- Evidence: RelevanceCorpus passes all 31 independent TD-001 query/tab cases through real handlers. Useful URIs in top five, all designated/concrete family negatives absent over full traversal. Separate six-reliable case retrieves correction seventh exactly once.

### AT-007
- Status: completed
- Evidence: Router-backed Flutter widget and client fixtures pass original crochett blanket across separate tabs and explicit recent-search save payload. Existing states/cursors retained; no Flutter production changes. First test setup lacked GoRouter and was corrected, not treated as product failure.

### REG-002
- Status: completed
- Evidence: Broader required-PG API Search|Timeline suite passes, covering exact hashtag, browse/filter, profiles, suggestions and timeline regressions.

### REG-004
- Status: completed
- Evidence: All 61 Flutter search tests pass, including provider pagination/account fencing, history, separate tabs, states and typo wire spelling. Scoped flutter analyze on both changed files reports no issues.


## Implementation Refinements

- The shared pipeline computes own fields directly from current rows. Migration 000089 contains only two helpers: per-concept scoring and the Unicode one-edit predicate. Image extraction/tag union remain bound-query SQL; there is no stored search projection or backfill.
- Reliable and corrected retrieval landed together with internal tier cursors because the new result order requires coherent pagination. Detailed ranking/cursor/handler tests then checked those boundaries while green.
- Phrase correction candidates must satisfy the complete reconstructed phrase before the eight-candidate cap. A later focused failing test exposed and fixed this gap; ordinary correction/equivalence regressions passed afterward.
- SQL-owned unit IDs are verified through real PostgreSQL, avoiding a duplicate Go stemmer/edit algorithm. Acceptance IDs share focused behavior evidence where their scenarios overlap; the full independent corpus has its own handler test.
- IT-014 lives in the API test package to reuse isolated fixtures, but invokes the real public Tap indexer. No production indexer changes.
- No outbound search/model/PDS dependency exists in the retrieval wiring. IT-008 combines wiring inspection with runtime read-only PostgreSQL and unchanged full-row/history snapshots; it does not claim a fictitious injected outbound spy.
- Existing UI regressions remain green. The router-backed widget fixture checks actual tab navigation. No Flutter production code, generated code, API DTO, route/auth policy, lexicon, dependency or infrastructure changes.

## Verification

- Each recorded red/green loop used real PostgreSQL 16 with TEST_DATABASE_REQUIRED=true; no database skips accepted.
- Broader Search|Timeline API suite: passed.
- Flutter search suite: 61 passed.
- Static analysis of changed Flutter tests: no issues.
- Full backend race suite via just test: passed (exit 0), with required PostgreSQL and MinIO integration suites.
- Diff/whitespace and traceability check: passed. All 32 active test IDs are completed; the 13 Must requirements and 16 active acceptance criteria map to the recorded tests. Tracked diff and new files pass whitespace checks.
- Implementation review: explicitly deferred to the user's next stage choice.
- Commit/push/deploy: not requested and not performed.

## Completion Notes

Implementation is complete within the approved scope. All focused behavior checks, the broader backend race suite, all 61 Flutter search tests and scoped Flutter analysis passed. Older-post upgrade/backfill coverage is retired; normal Tap update/replay/delete convergence remains covered. No performance metrics or measurement gates were added. The helper migration was tested locally only; no production deployment, commit or push was performed. Implementation review awaits the user's next stage choice.

## Review Correction: IR-001 / IT-012 / RULE-003 / AC-016

- Status: completed; awaits implementation re-review.
- Add same-fixture private marker presence assertions, then seed the missing moderation/scheduled payloads and exercise both handlers with captured diagnostics. Preserve production code and existing query/correction checks.
- The original completion claim lacked seeded private-content sink evidence; review found this gap.
- Red evidence: the strengthened fixture-presence assertion failed on the original test with “private-content canaries missing from captured observer fixture.” This exposes missing test setup, not a production privacy defect.
- Green evidence: after seeding both actual private tables, SerializedPrivacy and PrivateAndQuoteBoundaries pass on required PostgreSQL. Both routes search the fixture-read markers, assert empty pages and preserve serialized local/SDK canary absence with nonempty permitted operation context. No production code, migration or Flutter changes.
- Final verification: required-PostgreSQL API `Search|Timeline` regression suite passed (exit 0); whitespace checks passed. No production change or new concern warranted repeating the earlier full backend race suite or unchanged Flutter suites.
- IR-001 is addressed in the test and evidence; `06-implementation-review.md` remains the historical Changes required verdict until the next review stage. No commit, push or deployment performed.
