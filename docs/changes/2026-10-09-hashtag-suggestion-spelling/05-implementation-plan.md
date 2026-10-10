# TDD Implementation Plan: Hashtag Suggestion Spelling

## Inputs

- Requirements: `01-requirements.md`.
- Tests: `02-acceptance-tests.md`.
- Document review: `03-document-review.md`, Approved with notes.
- Coding plan: `04-coding-plan.md`.
- User invoked `implement-tdd` on 2026-10-09. No commits or production operations requested.

## Implementation Rules

- Implement only linked requirements; retain ASM-001–ASM-004.
- Write a focused failing behavior test before implementing that behavior.
- Run the smallest relevant test first; never refactor while red.
- Preserve source records, normalized identities, surface eligibility, ranking, cursors, and labels.
- Do not edit migrations until explicit migration approval required by the invoked skill is received.
- No production database access or migration execution is included.

## Progress

| Item | Status | Notes |
|---|---|---|
| Read workflow documents | completed | Loaded requirements, tests, review, and coding plan from this worktree. |
| Create execution plan | completed | This document mirrors coding-plan groups at individual test-ID granularity. |
| Inspect code and tests | completed | Existing API seed helpers, facet query, testdb isolation, and local test runner inspected. |
| Final verification | completed | All Go packages passed across the full race run and the MinIO package rerun; 59 focused Flutter tests, targeted analysis and formatting checks passed. See the correction-pass evidence for the transient failure. |
| Update implementation notes | completed | Evidence, requirement coverage, test-seam changes and rollout limits recorded and read back. |
| Commit | cancelled | Not requested. |

## Test Order

The order follows section 9 of `04-coding-plan.md`. IDs grouped there share behavior evidence; each remains separately tracked here. IT-001 starts with the composer example, then gains the remaining surfaces and JSON evidence. UT-001/UT-002 are covered at SQL integration level. Existing protections that already pass will be recorded without artificial failures.

| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State | Status |
|---|---|---|---|---|---|
| 1 | IT-001 | BR-001, FR-002, RULE-001, RULE-003 | AC-001, AC-002, AC-004, AC-009 | Missing behavior or existing regression protection | completed |
| 2 | AT-001 | BR-001, FR-002, RULE-001 | AC-001, AC-004 | Missing behavior or existing regression protection | completed |
| 3 | UT-003 | FR-004, RULE-002 | AC-006, AC-008 | Missing behavior or existing regression protection | completed |
| 4 | UT-006 | RULE-001, FR-001 | AC-002, AC-003 | Missing behavior or existing regression protection | completed |
| 5 | IT-006 | FR-005, NFR-001 | AC-007, AC-006 | Missing behavior or existing regression protection | completed |
| 6 | UT-001 | RULE-001 | AC-001, AC-002 | Missing behavior or existing regression protection | completed |
| 7 | UT-002 | RULE-003 | AC-009 | Missing behavior or existing regression protection | completed |
| 8 | AT-002 | BR-001, FR-002, RULE-001 | AC-002 | Missing behavior or existing regression protection | completed |
| 9 | IT-005 | FR-004, NFR-001 | AC-006 | Missing behavior or existing regression protection | completed |
| 10 | IT-007 | RULE-002 | AC-008 | Missing behavior or existing regression protection | completed |
| 11 | IT-002 | FR-001, NFR-002 | AC-003 | Missing behavior or existing regression protection | completed |
| 12 | IT-003 | FR-002, RULE-004 | AC-004, AC-010 | Missing behavior or existing regression protection | completed |
| 13 | AT-003 | FR-001, NFR-002 | AC-003 | Missing behavior or existing regression protection | completed |
| 14 | REG-002 | RULE-004 | AC-010 | Missing behavior or existing regression protection | completed |
| 15 | UT-004 | FR-001 | AC-003 | Missing behavior or existing regression protection | completed |
| 16 | IT-008 | NFR-002 | AC-011 | Missing behavior or existing regression protection | completed |
| 17 | REG-001 | FR-001, NFR-002 | AC-003, AC-011 | Missing behavior or existing regression protection | completed |
| 18 | IT-004 | FR-003 | AC-005 | Missing behavior or existing regression protection | completed |
| 19 | AT-005 | FR-003 | AC-005 | Missing behavior or existing regression protection | completed |
| 20 | REG-003 | FR-003 | AC-005 | Missing behavior or existing regression protection | completed |
| 21 | UT-005 | FR-002 | AC-004 | Missing behavior or existing regression protection | completed |
| 22 | AT-004 | FR-003, RULE-004 | AC-005, AC-010 | Missing behavior or existing regression protection | completed |
| 23 | REG-004 | RULE-004 | AC-010 | Missing behavior or existing regression protection | completed |

## Implementation Steps

### Step 1: IT-001 / AT-001

- Focused test: `TestHashtagSuggestionsMostUsedSpelling` in `appview/internal/api/search_store_test.go`.
- Fixture: nine distinct public records with valid UTF-8 facet byte ranges; four lowercase and five mixed-case spellings. Indexed identity remains `memademay`.
- Expected response: exactly one `MeMadeMay` item with `postsLast28Days` nine.
- Command: from `appview/`, `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:16235/craftsky_dev?sslmode=disable TEST_DATABASE_REQUIRED=true go test ./internal/api -run '^TestHashtagSuggestionsMostUsedSpelling$' -count=1`.
- Confirmed failure: focused test ran against local PostgreSQL and failed with actual `Tag:"memademay", PostsLast28Days:9` versus expected `Tag:"MeMadeMay", PostsLast28Days:9`. This is the intended missing behavior, not a setup failure.
- Implemented: approved migration 000089, source-parity checks, and composer/search/top-hashtag eligible CTE aggregation.
- Refactor: none while red.
- Test execution: initial sandbox attempt could not access the Go build cache; the approved rerun completed and produced the meaningful failure above.
- Environment: started only this worktree's local PostgreSQL container; isolated test schemas are disposable. The later approved migration was exercised only in isolated test schemas; no production operations executed.

### Remaining steps

Executed results are recorded below. Final verification is complete. The coding plan requires projection parity before discovery queries use it; the first behavior test may remain red during that dependency's test loop. No unrelated test batch is authorized by that dependency.

## Approval Gate

The supplied implement-tdd skill says: “Stop for explicit approval before touching high-risk areas: auth, permissions, billing, payments, migrations, destructive actions, privacy, security, or compliance.”

The next implementation dependency is the planned local up/down migration adding the immutable public-record spelling extractor and stored generated `craftsky_posts.tag_spellings` column. The user explicitly approved authoring and testing this migration on 2026-10-09 ("Yes. Go ahead."). Production execution remains outside scope.

## Completion Checklist

- [x] All Must requirements covered by tests or documented gaps
- [x] All planned Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Docs updated
- [x] Implementation review explicitly deferred to the next workflow stage; not automatically performed

Implementation and local verification are complete. Separate implementation review is deferred to the next user-selected stage.

## Execution Evidence

- UT-003 / FR-004, RULE-002 / AC-006, AC-008: `TestHashtagSpellingsRetainsEligibleSources` failed because the extractor did not exist, then passed after migration 000089. Exact project sources, exact deduplication, and non-project exclusion verified.
- UT-006 / RULE-001, FR-001: `TestHashtagSpellingsFacetParity` passes for accented/uncased text, UTF-8 byte lengths and Unicode whitespace; also verifies Go DecodeFacets equivalence for unknown/null/malformed facets, invalid ranges, fractional and overflowing offsets. These dependency checks were added before changing the query.
- IT-006 / FR-005, NFR-001: exact migrated PostgreSQL 16 schema rolled back to version 88, seeded legacy source, then applied version 89 twice via disposable down/up. Recovered spelling matches fresh extraction and all preexisting row fields remain unchanged. The later recovery-versus-fresh-indexing test extends this to all supported source locations.
- Focused database command: `go test ./internal/db -run '^TestHashtagSpellings' -count=1`, database required; passed.
- Composer query now derives aggregate counts and exact-spelling frequencies from one eligible CTE. No spelling is guessed for an identity lacking recoverable candidates; historical integrity remains an explicit rollout check.

- IT-001/AT-001 composer red→green: `go test ./internal/api -run '^TestHashtagSuggestionsMostUsedSpelling$' -count=1` passed after projection and eligible CTE aggregation.
- IT-001 search red→green: extending the same scenario failed with `memademay 9`; minimum search query change passed.
- IT-001 top hashtags red→green: next extension failed with `memademay 9`; minimum craft query change passed. Existing adjacent ranking, visibility, suggestions and top-hashtag tests passed.
- UT-001/UT-002/AT-002/IT-007: SQL tables exercise lowercase/all-caps dominance, forward/reverse tied insertions, accented ties, uncased tags, duplicate facets and overlapping variants. All three discovery surfaces and query casing permutations passed without extra production changes.
- IT-005: full-schema real-indexer lifecycle matches all eight TD-009 states and update-before-create/replay. Initial suggestion was empty because the direct indexer fixture omitted image-safety state; seeding clear state for image-free synthetic records fixed setup, with no production permission change.
- Implementation seam: a shared SQL CTE fragment in `internal/api/hashtag_aggregation.go` prevents the three queries from drifting in voting/counting policy. Endpoint eligibility and normalized identity ordering stay in their existing query files.

- IT-002/AT-003/IT-008/REG-001: `TestHashtagSpellingRankingAndSavedCursor` passes for lower/mixed/upper queries, normalized relevance/alphabetical ordering, casing-only changes between pages, saved cursor continuation, and empty/no-match results. Existing limit/escape/cursor/request protections pass.
- IT-003/REG-002: `TestHashtagSpellingEligibilityAndCraft` passes with otherwise dominant excluded spellings on old/reply/quote/moderated/blocked/muted posts, craft-specific winners and aged-out populations. Composer intentionally retains its existing relationship-policy difference.
- UT-004: removed unused `RankHashtagResults`, `NormalizeHashtagSuggestionRows`, and their obsolete summation tests after nearby tests were green. SQL integration and existing request/response/cursor suites now protect the actual public path; no redundant winner algorithm added.
- IT-008 Flutter red→green: casing-only adjacent-page changes initially yielded five rows instead of two. Deduplication by lowercase identity, retaining the first loaded row, passes all five pagination-merge tests.
- IT-001/IT-004: real handler JSON retains mixed-case `tag` and existing camelCase count fields on all four discovery endpoints. Three tag-feed casing variants retrieve the same nine ordered post identities.
- UT-005/AT-004/AT-005/REG-004: 45 focused Flutter tests pass. API mapping retains exact response strings; composer inserts `Before #MeMadeMay ` and keeps `9 posts` without a window explanation; actual routed taps from suggestions/results/top chips retain the selected heading and repository tag argument. New fixture compilation/query mismatch errors were corrected before claiming behavioral evidence.
- IT-006 extended: `TestHashtagSpellingsRecoveryMatchesFreshIndexing` compares ten historical records (nine frequency-example posts plus all project metadata sources) to the real indexer in a separate exact-migration schema; all spelling arrays agree and both yield `MeMadeMay`/9. All four spelling migration tests pass.
- API fixture compatibility: two older identity-cache tests used minimal schemas without the new column. They now use the exact migration-backed shared test pool and valid public source facets; both pass. No normalized array is treated as original-spelling evidence.

- REG-002 completion: `TestHashtagSpellingExcludesTerminalOwners` passes on the full migration schema; three terminal-owner alternative spellings influence neither counts nor winner on any discovery surface. Time tests now include the exact cutoff, one microsecond inside, and one microsecond outside.
- Indexer fixture alignment: existing `TestCraftskyPost_*` suites now apply the exact spelling migration alongside their focused DDL, so future/malformed facets and lifecycle regressions exercise the generated projection too. Focused race check for indexer, cutoff and terminal-owner tests passed.
- REG-003: composer test now explicitly asserts the submitted author text retains `#SockKAL`, alongside its exact tag facet. Existing Go recent-search normalization tests and Dart typed recent-search payload/display tests remain protections.
- REG-004: composer zero/one/multiple count labels retain their existing wording (including the current composer `1 posts` label); search navigation tests verify existing localized `0 posts` / `1 post`. No product copy was changed.
- Broader Flutter command: `flutter test --no-pub test/search/providers/search_pagination_merge_test.dart test/shared/rich_text/facet_autocomplete_editor_test.dart test/feed/widgets/post_composer_sheet_facets_test.dart test/search/search_page_test.dart test/search/data/search_api_client_test.dart test/shared/rich_text/facet_suggestion_repository_test.dart test/search/models/recent_search_test.dart` passed, 58 tests. After explicit author-text and localized-count assertions were added, the two affected suites passed again (22 tests).
- Changed Dart files formatted. Targeted static analysis passed after correcting two test-only style findings (directive ordering and description length); final new composer assertion is formatted too.

## Query Cost And Rollout Limits

Measured locally against PostgreSQL 16 in a rolled-back disposable schema, using the actual composer SQL, exact migration 000089 and 20,000 synthetic rows: 100 matching identities across 10,000 matching posts plus 10,000 unrelated posts. Visibility helper functions returned clear/non-terminal for this cost fixture, and moderation was empty; eligibility semantics are verified separately by real-schema behavior tests.

- Generated-column `ALTER TABLE`: 306.003 ms, including deriving historical spelling arrays and rebuilding the fixture indexes.
- Composer execution: 57.160 ms; planning: 0.450 ms.
- Plan: one materialized eligible CTE (10,000 rows), distinct aggregate and spelling-frequency sorts, normalized identity result sorting with top-N limit. Shared-buffer hits: 966; no temporary-file spill. Main sorts used about 931 kB and 1,010 kB.
- Existing created-at and tag-array GIN indexes were present. The planner selected a sequential scan for this broad substring query; adding another array index would not directly accelerate the current unnest/substring predicate. No speculative index added.
- This is a sizing observation, not a latency-budget pass or production estimate. A stored generated column rewrites existing rows and takes the normal `ALTER TABLE` lock. Production row volume, historical source completeness and deployment lock duration remain rollout checks; none were inspected or changed here.
- No candidate is invented from normalized arrays when retained source spellings are absent. Deployment review must verify required historical source integrity before enabling these queries; an incomplete source population needs explicit requirement revision rather than treating this fixture evidence as a live-data guarantee.

## Initial Implementation Completion

Full `just test` Go race suite passed with local PostgreSQL and MinIO. No commits, pushes, production migrations or deployments are authorized by this implementation stage. Separate implementation review is the next user-selected stage.


Initial verification:

- `just test`: full Go race suite passed with required local PostgreSQL/MinIO services. No integration skip path was used.
- Later focused race verification after indexer fixture/cutoff additions: `go test -race ./internal/index ./internal/api -run 'TestCraftskyPost|TestHashtagSpellingEligibilityAndCraft|TestHashtagSpellingExcludesTerminalOwners' -count=1` passed.
- Seven focused Flutter suites passed (58 tests); the two suites with final author-text and localized-count assertions passed again (22 tests).
- `flutter analyze --no-pub` over all seven changed Dart files: no issues found.
- `gofmt`, `dart format`, and `git diff --check`: clean.
- Changes reviewed for scope against requirements: public records/normalized identities remain intact, API wire shape/routes and eligibility predicates preserved, no production credentials/infrastructure/lexicon changes.

Requirement completion mapping:

| Requirement | Executable evidence |
|---|---|
| BR-001, FR-002, RULE-001 | Core nine-post example, SQL winner tables, all discovery-handler JSON, Flutter mapping and discovery widgets |
| FR-001, NFR-002 | Query-casing/ranking/cursor SQL cases, existing request/escape/limit tests, Flutter identity-based pagination |
| FR-003 | Composer insertion and exact authored-text tests, actual routed discovery selections, equivalent mixed/lower/upper tag-feed requests |
| FR-004, NFR-001 | Exact eight-step real-indexer lifecycle, update-before-create, replay, migration down/up and fresh-indexing parity |
| FR-005 | Exact historical migration and every-source recovery versus real fresh indexing; live historical integrity remains a rollout dependency |
| RULE-002 | Exact source deduplication, repeated/overlapping variant SQL tables, distinct aggregate separate from spelling frequencies |
| RULE-003 | Forward/reverse insertion ties, accented case pair, explicit spelling `COLLATE "C"` ordering |
| RULE-004 | Time cutoff/craft/visibility/relationship/terminal-owner SQL tests, composer and localized count-label protections |

All 23 planned test IDs are covered by executed tests or existing regression protections at the documented seams. UT-001/UT-002 and extraction additions use SQL integration rather than an unused pure helper. No manual-only test or unresolved local verification gap remains. The implementation-review verdict is intentionally not claimed in this stage.

## Review Correction Pass

The user selected Address required changes after `06-implementation-review.md` reported IR-001 and IR-002. The earlier completion evidence above describes the first implementation, not resolution of these findings. Existing migration authoring/local-test authorization still applies; no production operation or commit is requested.

| Order | Test ID / Finding | Requirement IDs | Acceptance Criteria | Status |
|---|---|---|---|---|
| 1 | UT-006, IT-001, IT-002 / IR-001 | FR-001, FR-002, FR-004, RULE-001 | AC-003, AC-004, AC-006 | completed |
| 2 | UT-003, IT-005, IT-006 / IR-002 | FR-002, FR-004, FR-005 | AC-004, AC-006, AC-007 | completed |
| 3 | IT-008, REG-001 / IR-001 client continuation | FR-001, NFR-002 | AC-003, AC-011 | completed |
| 4 | Final service-backed verification and evidence update | All existing requirements | Existing acceptance criteria | completed |

Correction strategy: preserve the generated-column design and existing normalized arrays. Use Go's simple Unicode lowercase mapping for candidate identity matching, rather than platform-dependent database casing. Preserve accepted case-insensitive source-field matching. Add real-indexer-to-discovery tests first, then migration parity/recovery coverage. No unrelated UI or policy changes.

IR-001 red→green: `TestCraftskyPost_HashtagUnicodeSpelling` failed with no result for one uppercase post and the wrong lowercase winner for a 2–1 uppercase majority. After `craftsky_tag_lower` was added to the approved unshipped migration and used for spelling votes, the same real-indexer test passed on all three discovery stores for both query casings. Initial missing test import was corrected before interpreting the behavioral failure. The mapping is generated from the repository's Go Unicode 17 simple lowercase rules; `TestHashtagSpellingsUnicodeNormalization` checks every changed case mapping and lower counterpart against `strings.ToLower`. Adjacent migration/indexer/discovery suites passed.

IR-002 red→green: `TestCraftskyPost_HashtagFieldCasing` first verified source validation accepts each fixture and the real indexer retains `memademay`. All five cases then failed with an empty suggestion: facet/index fields, top-level facets, structured project tags, pattern fields, and material fields. `craftsky_tag_field` now matches the ASCII schema names with Go's case-folding aliases, including long s/Kelvin equivalents, at each recognized source boundary. The same test passed on all three discovery stores. Additional migration parity cases verify `$TYPE`, Unicode field-name folding, and invalid recognized case aliases still invalidating their sibling array. Recovery-versus-fresh-indexing now covers 12 records, including every source through field aliases and a Unicode normalization mismatch, with explicit expected spelling arrays and discovery results. Adjacent migration/lifecycle/discovery tests passed.

Prior-pass conclusion, superseded by the IR-003 correction below: the casing helper followed the serialization order of the retained JSONB object when multiple keys fold to one field. JSONB cannot retain original key order or overwritten duplicate exact keys. Consequently, historical conflicting aliases are a source-integrity limitation rather than proof of universal original-token decoding parity; deployment verification must check for them and return to the FR-005 recovery gate if they affect required contributions. No source record is rewritten, no alias spelling is fabricated, and no new record rejection is added. The normal single-field casing cases identified in IR-002 are verified through original raw-source indexing and historical recovery.

IR-001 client continuation: a direct runtime check showed Dart retains U+A7CE on lowercasing, unlike the AppView. A new `IT-008 / REG-001` pagination test failed with two rows after a casing-only change between pages. `hashtagIdentity` now applies the same simple lowercase mapping as Go while retaining the first displayed row. The same focused suite passed. This is a justified coding-plan implementation refinement: Dart's platform lowercase cannot serve as the AppView's stable identity for these supported characters. No wire field, provider, route, or casing policy was added. `scripts/generate-hashtag-case-map.go`, run from `appview`, produces both the SQL map and Dart map from Go's Unicode rules without a dependency; the SQL parity test guards toolchain drift. Generated-column ownership and the stored normalized arrays are unchanged.

Cost refinement: the first broad 20,000-row query probe after full Unicode mapping took 388.707 ms. An ASCII fast path reduced this to 76.082 ms while preserving focused Unicode parity tests. The first broader Go test run was deliberately stopped to make this refinement; its exit 143 is an interrupted run, not passing evidence. Repeated extractor field scans were then reduced while green; final broader verification follows the refinements.

Correction verification:

- Focused migration/source parity, historical recovery, real-indexer lifecycle, all three discovery stores and HTTP scenarios passed with required PostgreSQL 16.
- Seven relevant Flutter suites passed, 59 tests, including the new Unicode pagination regression. Targeted analysis of the new identity helper, pagination implementation and pagination tests reported no issues.
- Final rolled-back synthetic fixture with the same 20,000 rows: generated-column ALTER TABLE 3,553.458 ms; query execution 76.077 ms; planning 0.534 ms; no temporary-file spill. Caching repeated field lookups reduced migration time from the initial correction probe's 8,604.105 ms. Compared with the original fixture's 306.003 ms migration and 57.160 ms query, accepted-field compatibility increases migration work and the identity mapping adds some query work. These local measurements do not establish a production latency or lock-duration guarantee.
- Final `just test` exercised the full Go race suite with required PostgreSQL/MinIO. Every package except `internal/scheduledposts` passed; that package reported `TestS3ObjectStoreAgainstMinIO` failing on Put with `private object store unavailable`. The command therefore exited 1 and is not recorded as a single green full-suite run.
- Local MinIO remained healthy. The Docker data filesystem had 1.8 GB free (97% used), but a disposable SDK write/delete probe succeeded, so low space was not established as the cause. The single failing MinIO test then passed with race detection. The entire `internal/scheduledposts` package passed on rerun (17.670 s) with the same required PostgreSQL and MinIO configuration. All packages have passing race evidence across the full run and this rerun; the initial transient failure remains recorded. No object-store source, fixtures, service configuration, or user data was changed.
- Formatting and `git diff --check` are clean. Source records, normalized arrays, API routes/count fields, viewer eligibility, count copy and PDS data remain unchanged. Corrections and the generator are local and uncommitted.

Correction completion: IR-001 and IR-002's reproduced cases are corrected and covered by passing regressions, including the discovered Flutter identity mismatch. No planned test was skipped. Historical source integrity and production migration sizing remain rollout dependencies. `06-implementation-review.md` remains the first review snapshot; a fresh review of these corrections is the next user-selected stage. No commit, push, deployment, or production migration was performed.

## IR-003 Validation Correction Pass (2026-10-09)

User authorization: explicitly approved the validation fix and instructed implementation now, including migration authoring/testing and requirements revision. The contract change is recorded in 01/02/04; historical 03/06 remain reviews of their prior snapshots. No commit, push or production operation requested.

| Step | Test ID | Requirement IDs | Acceptance criteria | Status |
|---|---|---|---|---|
| 1 | UT-007 | FR-006, FR-004 | AC-012 | completed |
| 2 | IT-009 | FR-006, FR-005 | AC-007, AC-012 | completed |
| 3 | IT-010 | FR-006, FR-004, FR-002 | AC-004, AC-006, AC-012 | completed |
| 4 | Existing regression gates / final verification | Original Must requirements and FR-006 | AC-001–AC-012 | completed |

Progress: workflow and authorized contract revision loaded; UT-007, IT-009 and IT-010 completed; final verification completed. Fresh implementation review remains the next stage.

UT-007 red: all 24 conflicting-field cases were incorrectly accepted on create/update. Green: shared raw-token ambiguity validation rejects them; all source-validator tests pass, retaining single aliases, Unicode aliases, separate features, unknown payloads and deletes. No new diagnostic logging was added.

IT-009 red: the exact migration accepted an ambiguous historical source without error. Green: scoped JSONB ambiguity preflight rejected the initial nine historical fixtures; expanded final coverage rejects all 26 fixtures across the recognized source boundaries; failure rolls back schema changes and preserves source rows. Explicit fixture resolution enables correct recovery. All hashtag migration tests pass (4.390 s). A CHECK constraint also guards bypass writes; migration down removes its helper/constraint.

IT-010 red: direct indexing accepted duplicate exact `tag` keys, which JSONB later collapses. Green: the shared raw ambiguity guard runs before decoding/projection writes. Rejected direct create/update preserve the existing post; direct SQL with conflicting aliases fails the new constraint. Single aliases still return MeMadeMay/1 on every discovery store. The real ingestion store retains duplicate keys in its JSON (not JSONB) source column; worker revalidation already rejects them, retains the invalid source, and clears stale serving state under the existing invalid-update policy. No worker change is needed. The durable integration fixture was corrected after its initial JSONB assumption and lifecycle-helper setup error; these were setup failures, not red/green evidence. Focused direct/durable/alias/lifecycle tests pass (4.823 s).

Final verification completed:

- `just test` passed in one full service-backed Go race run (exit 0), including PostgreSQL, the API/indexer/ingestion regressions, and MinIO. The previously intermittent MinIO package passed in this run (14.892 s). No required correction test was skipped or failure suppressed.
- Final 26-fixture historical ambiguity coverage passed as part of the full database package (17.518 s). The full indexer package passed (17.521 s), including raw validation, direct-write rejection, durable-source handling, aliases, Unicode, lifecycle and discovery coverage. The API package passed (31.066 s).
- `gofmt -l` on all six correction Go files produced no output. `git diff --check` passed.
- The same rolled-back 20,000-row cost fixture, using the final exact migration, measured historical preflight 1,269.811 ms, constraint validation 1,219.252 ms and generated-column ALTER TABLE 2,929.160 ms (about 5.42 s combined). Query execution was 76.452 ms, planning 0.415 ms, without temporary-file spill. The prior generated-column-only measurement was 3,553.458 ms; the new safety checks add migration work. These local observations do not establish production lock duration or performance guarantees.
- Flutter was not rerun for this backend-only correction; its unchanged prior seven-suite/59-test and analysis evidence remains recorded above.

Correction completion: all three added Must test IDs are implemented and passing, alongside the original 23 IDs. FR-006/AC-012 explicitly authorize rejecting ambiguous recognized source fields; CPQ-001 applies to the revised unambiguous-source contract. The historical migration fails visibly before enabling spelling recovery; single aliases, unknown payloads and normal hashtag variants remain supported. Historical overwritten exact JSONB keys remain an explicitly documented information limit, not a claim of recovered original tokens. Where available, the durable Tap JSON source retains original field order/keys for explicit investigation. Production integrity/volume checks remain rollout work; no production source was inspected or changed.

- [x] Authorized requirements, acceptance tests and coding plan revised.
- [x] UT-007, IT-009 and IT-010 completed with meaningful red/green evidence and adjacent regressions.
- [x] Full required service-backed Go race verification passed.
- [x] Formatting, diff checks and migration cost observations recorded.
- [x] No unlinked behavior, PDS changes, commits, pushes or deployments.
- [ ] Fresh implementation review: pending user stage choice. Historical 06 remains the pre-correction review snapshot.

## Main Merge Verification (2026-10-09)

Merged freshly fetched `origin/main` at `3a2ef57f` in merge commit `24dbf82a`, preserving the uncommitted hashtag implementation. Resolved the search-store fixture conflict by retaining main's lazy-loading search helper migration and applying the hashtag migration through the shared hashtag fixture. Main introduced migration `000089_search_matching_helpers`, so the unshipped hashtag migration is now `000090_hashtag_spellings`; source, test, generator and plan references were updated.

Post-merge verification: `just test` passed the full service-backed Go race suite (exit 0); the seven affected Flutter suites passed all 59 tests. `git diff --check` passed, no unresolved index entries remain, and migration versions are unique. No push or production operation was performed.
