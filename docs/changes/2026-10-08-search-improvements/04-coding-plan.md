# Coding Plan: First Search Improvements

## 1. Inputs

- Requirements: `01-requirements.md`, including user-authorized simplifications and the document-review follow-up edits.
- Tests: `02-acceptance-tests.md`, including clarified language/typo eligibility, explicit equivalence cases, deterministic proposed bounds and concrete distractors.
- Document review: `03-document-review.md`; verdict Approved with notes, risk Medium. It records the earlier versions; the user subsequently authorized the revisions and progression to this stage.
- Repository context: `atproto-craft-social-app-reference.md`, the AppView API architecture spec, `docs/development/logging-error-reporting.md`, existing search capability owners, indexer materialization, Flutter search providers and test harnesses.
- This artifact is a design contract. No implementation, executable tests, dependencies, migrations or commits were created or run in this stage.

## 2. Implementation Strategy

Keep submitted keyword retrieval inside the existing PostgreSQL-backed SearchStore. Add shared concept planning and SQL matching helpers behind the two existing submitted-search methods. Preserve their public capability interfaces, handler hydration and Flutter request/state behavior.

Compute search documents from existing indexed rows at query time: caption, deduplicated authored tags, own image alt text and current project fields. Retain a literal document for every record and add English word-form matching only for records declaring English. Apply all meaningful concepts across fields, with the three approved equivalences. Build reliable and single-correction results in one database statement, deduplicate by URI, and paginate a deterministic tier/score/time/URI tuple.

Use a local migration for small deterministic SQL helper functions, with no new search table, persisted vectors, global word vocabulary, trigger, worker or external extension. Existing rows immediately obtain the new behavior after the helper migration and application update; existing indexing transactions continue to own field updates/deletes. This resolves the migration/backfill question without introducing another projection lifecycle.

No new extension is selected. Migration 000019 already creates pg_trgm, but trigram similarity alone does not enforce the agreed one-edit boundary. Implement a narrowly scoped one-edit predicate using core PostgreSQL/PLpgSQL instead of adding fuzzystrmatch or relying on a global vocabulary. Verify its behavior with real PostgreSQL tests. Keep existing indexes and other search surfaces intact; index tuning and performance assessment are outside this slice.

PostgreSQL 16 supports language-configured document/query parsing, weighted vectors and phrase queries. Preserve literal matching alongside English forms, and use phrase parsing only internally for curated multiword equivalents, without adding user query syntax. References: [PostgreSQL 16 text-search controls](https://www.postgresql.org/docs/16/textsearch-controls.html), [text-search functions](https://www.postgresql.org/docs/16/functions-textsearch.html).

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Acceptance Criteria | Test IDs |
|---|---|---|---|---|---|
| Concept planning | Lowercase/trim via searchTSQuery | Shared bounded concept representation; longest approved phrase recognition before filler removal | FR-001, FR-002, NFR-002 | AC-002, AC-003, AC-004, AC-012 | UT-001, UT-002, UT-003, UT-007, IT-001, IT-002, IT-009 |
| Submitted retrieval | Separate pgx SQL capability owners | Shared matching pipeline over eligible rows with reliable/typo tiers and scoped fields | FR-003, FR-004, RULE-001, RULE-002 | AC-005, AC-006, AC-007, AC-008, AC-014, AC-015 | AT-003, AT-004, AT-006, UT-004, IT-003, IT-004, IT-010 |
| Ordering and cursors | Score/time/URI keyset cursor | Explicit tier; deterministic selection and complete mixed pagination | FR-005, FR-006 | AC-009, AC-010 | AT-005, UT-005, UT-006, IT-005, IT-006 |
| Local database helpers | Numbered up/down migrations | Pure field/token/one-edit helpers; no data backfill or new projection | NFR-001, NFR-002 | AC-011, AC-012 | IT-007, IT-008, IT-014 |
| HTTP/diagnostics | Existing parsers, narrow interfaces and hydration | Preserve wire behavior and query privacy; add contract/sink regressions | RULE-003, RULE-004 | AC-016, AC-017 | IT-011, IT-012, IT-013, REG-004, REG-005 |
| Relevance fixtures | Go PostgreSQL tests and seed helpers | Explicit independent query/tab cases plus controlled rank/tier corpus | BR-001 | AC-001 | AT-001, AT-002 |
| Flutter | Riverpod repository/provider graph | Test extensions only; preserve source graph, tabs and original query | RULE-004 | AC-017 | AT-007, REG-004, REG-005 |
| Other surfaces | Existing browse/hashtag/profile/feed owners | Existing regression suites remain authoritative | RULE-001, RULE-002 | AC-014, AC-015 | REG-001, REG-002, REG-003 |

## 4. Files And Modules

Paths marked Create are implementation-stage targets only. Migration number 000089 is provisional: the current maximum is 000088; recheck before adding it.

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/internal/api/search_matching.go` | Create | Internal concept/dictionary/filler plan and immutable policy constants; parameter payload for SQL | FR-001, FR-002, FR-003, NFR-002 | UT-001, UT-002, UT-003, UT-004, UT-007 |
| `appview/internal/api/search_matching_store.go` | Create | Shared submitted-search SQL fragments and execution shape; preserve capability-specific filters and scanners | FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, RULE-002 | IT-001 through IT-006, IT-009, IT-010 |
| `appview/internal/api/search_post_store.go` | Change | Delegate nonempty submitted query to shared matching pipeline with Posts eligibility | FR-001, FR-003, FR-004, RULE-001 | AT-002, AT-004, AT-006, IT-001, IT-003, IT-004 |
| `appview/internal/api/search_project_store.go` | Change | Delegate submitted query with current project filters and Projects eligibility; preserve browse branch | FR-001, FR-003, FR-004, RULE-001 | AT-003, AT-004, AT-006, IT-002, IT-003, IT-004, REG-002 |
| `appview/internal/api/search_store.go` | Change | Private tier-bearing scan result beside existing SearchPostRow; reuse shared visibility predicates | FR-005, FR-006, RULE-002 | IT-005, IT-006, IT-010 |
| `appview/internal/api/search_cursor.go` | Change | Internal submitted cursor with tier validation and existing normalized-query/kind binding | FR-006, RULE-004 | UT-006, IT-006, IT-013 |
| `appview/migrations/000089_search_matching_helpers.up.sql` and `.down.sql` | Create | Scoped helper functions and exact down cleanup; no new extension/table/index or authored-data rewrite | NFR-001, NFR-002 | UT-004, IT-007, IT-009 |
| `appview/internal/api/search_matching_test.go` | Create | Concept/dictionary/filler/bounds table tests; SQL-owned normalizer/edit checks use real DB rather than a second Go algorithm | FR-001, FR-002, FR-003, NFR-002 | UT-001 through UT-004, UT-007 |
| `appview/internal/api/search_store_test.go`, `search_ranking_test.go`, `search_cursor_test.go`, `search_request_test.go` | Change | Focused retrieval, rank, cursor, parser, visibility and existing regressions | FR-001 through FR-006, NFR-002, RULE-001, RULE-002 | UT-005, UT-006, IT-001 through IT-006, IT-009, REG-001 through REG-003 |
| `appview/internal/api/search_improvements_acceptance_test.go` | Create | Migrated-schema corpus, actual handler workflow, external-call boundaries and private/quote isolation | BR-001, NFR-001, RULE-001 through RULE-004 | AT-001 through AT-006, IT-008, IT-010, IT-011, IT-013 |
| `appview/internal/api/search_improvements_privacy_test.go` | Create | Serialized log/SDK/outbound canaries and retained permitted error context | RULE-003 | IT-012 |
| `appview/internal/db/search_improvements_migration_test.go` | Create | Seed pre-upgrade schema, apply exact helper migration, retrieve old rows without mutation | NFR-001 | IT-007 |
| `appview/internal/index/search_improvements_projection_test.go` | Create | Real indexer update/replay/delete convergence with current search helpers | NFR-001, RULE-002 | IT-014 |
| `appview/internal/api/search_capability_test.go`, `search_response_test.go`, `search_recent_store_test.go` | Change tests if needed | Contract/hydration and fetch-versus-explicit-save assertions | RULE-004 | IT-013, REG-005 |
| `app/test/search/search_page_test.dart`, client/error suites, provider contract tests and recent-search suites | Change tests only | Original typo wording, separate tabs, opaque cursors, states and explicit history saves | RULE-004 | AT-007, REG-004, REG-005 |

Keep SQL in the existing pgx search ownership pattern: these stores currently compose SQL with shared visibility predicates. The queries directory is empty and no sqlc configuration was found in this checkout. Do not introduce an ORM or a new query-generation subsystem for this slice. Pure database helpers belong in migrations; shared retrieval SQL belongs to the search capability owner, with user input bound as parameters.

No production change is planned for `search.go` capability interfaces, route registration, Flutter source/generated providers, indexer write logic, lexicons or dependencies. If a failing approved test exposes a necessary boundary fix, keep it narrowly tied to the existing contract and document it in the later implementation evidence.

## 5. Services, Interfaces, And Data Flow

### Existing seams and internal types

Keep `SearchPostsWithLanguages` and `SearchProjectsWithLanguages` returning `[]SearchPostRow, cursor, error`. Keep SearchStore construction, engagement/quote/relationship readers and `buildSearchPostResponses` as they are. Internal identifiers should use `syntax.ATURI`/`syntax.DID` where new internal fields carry them; do not expand this into refactoring all existing string boundaries.

Illustrative private signatures only:

```text
buildSearchConceptPlan(query string) -> conceptPlan
searchSubmitted(ctx, scope, viewer, languages, request, conceptPlan) -> page
rankedSearchHit { row SearchPostRow; tier matchTier }
submittedCursor { kind; normalizedQuery; tier; score; createdAt; uri }
```

The concept plan is request-local and never written to history, logs or SDK attributes. Serialize a bounded internal JSON parameter if helpful for SQL iteration; this is an internal PostgreSQL parameter, not a new API body or serialization format.

### Query concepts and language policy

1. Retain existing lowercase/trim normalization for cursor query binding. Keep original request wording intact for client/history behavior.
2. Tokenize with Unicode-aware punctuation handling; use PostgreSQL's simple parser for actual field/query lexemes. The Go planner identifies approved phrases and filler words, not a competing stemmer. Test punctuation, hyphenated technical names and codes against actual PostgreSQL parsing so token splitting cannot silently discard a concept.
3. Recognize the longest approved sequences first: work in progress and stocking stitch are single concepts with WIP and stockinette alternatives. Jumper/sweater is the third group. Store original spelling plus the approved alternative; expand once without recursive substitution.
4. For this slice explicitly ignore filler `how` and `to` outside recognized phrases. Preserve meaningful terms, technical names and phrase components; do not blindly discard every PostgreSQL English stop word. An all-filler query yields no concepts and an empty page. Other words remain required unless they are part of an approved phrase. This deliberately small documented filler policy satisfies the approved natural-wording case without Google-style rewriting.
5. For each required concept, permit a literal/simple match on any scoped own field. Also permit an English-configured match only when record metadata declares base language en, including en regional tags and mixed lists containing en. Use existing language-tag conventions; never infer language from content. If English parsing erases a meaningful concept, retain its literal path rather than treating an empty tsquery as satisfied.
6. Approved multiword alternatives must match a phrase in an individual own field. Different concepts may match across fields; do not assemble a phrase from unrelated field endings. This uses phrase matching internally and introduces no user phrase/operator syntax.
7. Reliable eligibility requires every concept, independently satisfied by literal, applicable English forms or approved alternatives. Literal and English paths may jointly satisfy different concepts on the same record. No concept is dropped to fill results.

DR-001: reliable language normalization and typo eligibility remain separate. Test knitting/knit outside the chosen one-edit boundary for unknown-language exclusion from form matching, or inspect the reliable helper directly. Unknown-language socks may still match sock through the independent typo path.

### Scoped fields and relevance

Build per-field simple vectors and, where allowed, English vectors from current `craftsky_posts` and `craftsky_project_posts` columns. Null arrays/images/alt values contribute empty content. Read only own images' `alt` strings from the existing flattened JSON array; never traverse record JSON, quoted bodies, external links, captions/transcriptions or private tables.

Choose field weights A=1.0, B=0.4, C=0.2, D=0.1:

- A: caption, project title and pattern name.
- B: project materials.
- C: the canonical deduplicated union of materialized authored tags, project tags and design tags.
- D: own authored image alt text.

Normalize tag identity, deduplicate and order it deterministically before vector construction, including overlap between structured and materialized tags. Keep real caption/title/name word frequency; duplicate tag entries must not add relevance. Do not boost engagement.

Compute a concept score using the best matching alternative/path, rather than adding every equivalent or both English/literal matches. Use PostgreSQL `ts_rank_cd` on the applicable weighted field vectors, retaining positions. Sum concept contributions for the record. Preserve all-concept eligibility independently from the score. Controlled identical-content field comparisons must establish the primary-field priority in IT-005; no literal-versus-approved-synonym priority is required. Real repeated caption terms can contribute relevance, preserving the existing stronger-caption regression.

### Typo confidence and finite bounds

Confirm the test-design defaults for this implementation:

- Only query words of at least four Unicode characters are correctable; DK/K2/KAL are protected.
- Exactly one insertion, deletion, substitution or adjacent transposition is the correction boundary. Zero-edit candidates belong to reliable matching; more than one edit is rejected. Apply edits to words in original concepts, not to stems or recursively expanded alternatives.
- A small immutable SQL function compares two words, checking length difference and walking characters with one permitted edit. Do not build a full arbitrary-distance matrix or retry with a larger threshold. Bound comparisons by query length: immediately reject a candidate whose length differs by more than one.
- Derive distinct candidate words from simple lexemes of eligible records satisfying all other concepts. For approved phrases, replacing one original query word reconstructs the complete phrase/concept before normal matching; it does not discard the remaining phrase words. Keep this reconstruction testable even though the first required typo fixtures concern ordinary words/pattern names.
- Order qualifying candidate words by edit distance, then lexical word order under a deterministic C collation; take at most eight per original word. Apply visibility, tab eligibility and other-concept eligibility before this cap, and recompute from the same full eligible set on every page, independently of the cursor filter.
- Candidate selection is finite: at most 1 + 8n correction plans before the finite vocabulary alternatives for n meaningful words; only one original word is replaced in any plan. Use an OR of alternatives per concept rather than enumerating a Cartesian product of all vocabulary combinations. Expansion is one pass over three two-member groups.
- A corrected record must satisfy every concept through a selected one-word replacement. Use its best qualifying corrected score once. A record already reliable appears only in the reliable tier.

The candidate cap bounds matching expansion, not a claim about elapsed time or the number of rows PostgreSQL may inspect. No latency/query-plan/load evidence is required. SQL-owned edit/normalization checks run on PostgreSQL; do not create a duplicate Go edit algorithm solely to call a test “unit.”

### Retrieval statement and pagination

Use one statement with these conceptual stages:

```text
eligible published rows with existing scope/relationship/moderation/language predicates
  -> own field documents and required-concept match evidence
  -> reliable matches + viewer-eligible deterministic correction vocabulary
  -> corrected matches, excluding reliable URIs
  -> one best score per URI and explicit tier
  -> cursor predicate, ordered page of limit + 1
  -> existing PostRow scanner and hydration
```

Apply existing account-lifecycle predicates through the same visibility helpers. Preserve project-specific filter semantics for internal callers and the current quote/reply exclusions. Keep empty-query project browse and exact hashtag/feed queries on their existing code paths.

Use tier 1 for reliable and tier 0 for corrected so the entire ordering is descending `(tier, score, created_at, uri)`. The next-page predicate is strictly less than the cursor tuple. Generate the next cursor from the last returned row only if a lookahead row exists. Compute the candidate vocabulary before the cursor filter, preventing later pages from admitting a new vocabulary or missing a tier transition.

Add tier to the opaque submitted-search cursor and validate it as an allowed integer; validate score as finite, timestamp and URI as valid, and query/kind using the existing normalization. Reject a legacy submitted cursor lacking tier with the existing invalid_cursor response; a fresh search is permitted after deployment. No configuration version field, server snapshot, persisted search session or signature subsystem is introduced. Other chronological/popularity cursors remain unchanged.

### Migration and convergence

Add only helpers needed for own-image alt extraction, field/lexeme construction and the exact one-edit predicate. Use fixed function names prefixed `craftsky_search_`; explicit configuration/collation arguments and null-safe inputs make results deterministic. Grant no new privileges and read no private tables from helpers. The down migration drops only these new functions in dependency order; it must not drop existing shared array functions or indexes.

For IT-007, create an isolated production schema through migration 000088 (or the actual preceding cutoff if numbering changes), seed representative old public rows with unchanged record/CID values, apply the exact new helper migration and query through the new SearchStore. Use testdb migration readers/scoped test pools; do not introduce a production CLI or backfill command. Compare raw record/CID values before and after; no author/PDS action is involved.

Indexer production code needs no additional write step. IT-014 updates fields using the existing CraftskyPost Tap handler, replays the same URI/CID and delivers a delete. Query-time documents read current rows, so stale terms disappear without a separate orphaned vocabulary/vector. Validate this on migrated schemas, including real lifecycle predicates.

## 6. State, Providers, Controllers, Or DI

Preserve the current Riverpod graph:

```text
dioProvider
  -> searchApiClientProvider (keepAlive Provider)
  -> searchRepositoryProvider (keepAlive Provider, ApiSearchRepository)
  -> postSearchProvider(PostSearchQuery(q)) / projectSearchProvider(ProjectSearchQuery(q))
     (generated AsyncNotifier families, watching activeContentLanguagePolicyProvider)
  -> _SubmittedPostResults / _SubmittedProjectResults
```

Keep AsyncLoading/AsyncValue.guard handling, opaque cursor round-tripping, appendUniquePosts and active-account operation fencing. No new provider, matching-policy state, correction DTO or generated source is needed. Backend policy is immutable code/migration configuration, not an injected external service or runtime admin setting. Preserve current narrow SearchStore capability DI and construction.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

No widget tree or navigation changes. `_SearchResultsTabs` continues to host its existing tab views; submitted Posts and Projects remain independently backed by their respective providers, existing cards, loading skeletons, empty/error states and load-more behavior. Other tabs retain their existing behavior.

Keep GET `/v1/search/posts` and GET `/v1/search/projects` registration, parsers, authentication/device middleware, camelCase response bodies, error envelope and hydrated viewer/save/engagement/embed state. Tier/score are internal and never added to client DTOs. Keep `buildSearchPostResponses` and `attachQuoteViews` hydration; displaying a quoted post is distinct from searching its body.

Result fetching remains read-only. Preserve existing explicit recent-search save calls and normalization; the entered typo is never replaced by a candidate spelling in requests, inputs or saved payloads. No controls, correction explanation, instructional-answer UI, snippets or new CLI/background surface.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Missing/blank/oversized request, invalid limit | Preserve current parser statuses/envelopes, default 25 and maximum 100, query maximum 256 runes | RULE-004, NFR-002 | IT-009, IT-013, REG-004 |
| Valid all-filler query | Return empty items and no next cursor; never browse all | FR-001 | UT-001, IT-001 |
| No confident correction / missing concepts | Reliable results only or empty; no threshold escalation or dropped concepts | FR-002, FR-003 | AT-004, UT-004, IT-003 |
| Missing optional tags/images/alt | Null-safe empty field contribution, ordinary retrieval still works | FR-004 | IT-004 |
| Unknown/non-English/mixed languages | Literal always available; English forms only for declared en; existing viewer visibility independent | FR-001, RULE-002 | UT-002, IT-001, IT-010 |
| Hidden candidates exceed limits | Eligibility before correction cap and result selection; keep visible matches eligible | RULE-002 | AT-006, IT-010 |
| Malformed/mismatched/legacy submitted cursor | Existing invalid_cursor response, restart with no cursor; no cross-deployment promise | FR-006, RULE-004 | UT-006, IT-006, IT-013 |
| Database/query/scan/hydration failure | Existing search_unavailable handling and typed cause chain; no raw query/correction in error context | RULE-003, RULE-004 | IT-012, IT-013 |
| Client loading/empty/error/load-more/account change | Existing AsyncNotifier states and fencing; no source changes anticipated | RULE-004 | AT-007, REG-004 |
| Updated/replayed/deleted record | Existing transaction/idempotency convergence; current fields only, no extra projection | NFR-001, RULE-002 | IT-014 |

Keep existing ObserveDB and logging entry points without new metrics or spans. Empty results are normal. IT-012 captures serialized local/SDK output for success, validation and injected DB failure, asserting protected canaries absent and permitted operation/error context present. Exercise outbound search/model/PDS boundaries without forbidding the existing diagnostic SDK transport itself.

## 9. Test Implementation Plan

All references retain the local IDs in `02-acceptance-tests.md`. Fixtures TD-001 through TD-006 map respectively to realistic cases, fields/languages, corrections, ranking/pages, privacy/lifecycle/contracts and bounds. Use independent seed cases and the concrete distractors; review the full URI sets, not just counts.

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | IT-001, UT-001, UT-002, AT-002 | API matching/store suites | English socks/knitting showcase; literal names; unknown/non-English contrast outside correction distance; TD-001/TD-002 | Existing simple query fails English form/filler cases; literals should already pass |
| 2 | IT-007 | DB helper migration suite | Exact pre-upgrade schema and previously indexed records | Helper functions absent before migration; assert old rows receive behavior afterwards without record/CID writes |
| 3 | UT-003, IT-002, AT-003 | Matching/store/acceptance suites | All three groups in both directions/tabs; explicit missing-concept and cross-field records | Existing lexical search lacks synonyms/cross-field post coverage |
| 4 | IT-004 | Store suite | Exclusive marker per field; optional fields absent; TD-002 | Ordinary tags/alt and project authored tags/alt not searched yet |
| 5 | UT-004, UT-007, IT-003, IT-009, AT-004 | Matching helpers, PostgreSQL store and parsers | TD-003/TD-006, one-edit boundaries, protected codes, eight/nine candidates and shuffled seed order | Current search cannot retrieve correction; bounds helper absent; existing parser limits should pass |
| 6 | IT-010, AT-006, REG-003 | Migrated-schema acceptance/store suites | TD-005 with real terminal predicates, all matching paths, hidden candidates > cap | New paths must demonstrate eligible-before-cap behavior; existing visibility baselines should pass |
| 7 | UT-005, IT-005 | Ranking/store suites | TD-004 controlled fields/tiers, duplicate tags and changed engagement | Explicit tiers/primary field weights not present in current search |
| 8 | UT-006, IT-006, AT-005 | Cursor/store/handler suites | Reliable-only, synonym, typo-only, mixed and dual-qualified rows; limits 1/2/3 | Current cursor lacks tier and mixed-page behavior |
| 9 | IT-008, IT-011, IT-012, IT-013, REG-001, REG-005 | API capability/response/privacy/acceptance suites | TD-005, existing reader spies, actual DB state, supported Sentry mock transport and serialized logs | Baselines may already pass; new paths must preserve them, not manufacture a red failure |
| 10 | IT-014 | Indexer projection suite | Real Tap update/replay/delete, captions/tags/images/projects; migrated schema | Added field retrieval absent until matching changes; convergence should work without indexer production edits |
| 11 | AT-001 | Migrated-schema handler acceptance corpus | Independent TD-001 cases, six concrete distractors, explicit tier exception | Current search misses forms/equivalences/typos/alt tags; all intended cases must pass after increments |
| 12 | AT-007, REG-002, REG-004 | Flutter search/page/client/provider suites and existing API hashtag/profile/browse/timeline suites | FakeSearchRepository and mocked client responses; original typo; existing surfaces | Most regressions should already pass; add assertions without altering production UI behavior |

Unit test IDs describing SQL-owned matching remain the same, but use real PostgreSQL to execute those helpers. Do not duplicate their logic as a mock or use a unit-only run as evidence. Put tests under `api_test`/`index_test`/`db_test` conventions where possible; private pure planner tests may use package api if needed without exporting production helpers solely for tests.

Test fixtures that mirror schema manually must load the actual new helper migration SQL. Lifecycle and migration coverage must use exact production migrations; focused all-active fallback predicates are insufficient. The full TDD suite must report DB tests skipped without configured PostgreSQL as missing evidence.

## 10. Sequencing And Guardrails

- First TDD step: Add `TestSearchImprovementsEnglishWordForms` for IT-001 with Posts and Projects English sock/socks cases; run it against real development PostgreSQL and confirm the behavior fails before adding normalization. Add a separate knitting/knit unknown-language contrast to distinguish language forms from the later typo path.
- Dependencies: Helper migration and concept planning precede field/equivalence matching; visibility is present from the first query; correction candidate bounds precede typo retrieval; score/tier cursor design lands with mixed pagination; existing-row and indexer convergence tests follow actual production schema paths.
- Red-green-refactor: Implement one approved behavior at a time. Baseline regressions that already pass remain green; never deliberately break privacy or visibility to force a failing test.
- Keep raw query/word candidates in request-local memory and bound SQL parameters. Never interpolate them into SQL, function names, errors, logs or SDK attributes. No external search, model or PDS write clients are introduced.
- Preserve source record/CID values, current indexer transactions, account-removal fencing, hydration seams and client query spelling. No global vocabulary cache whose hidden content influences correction candidates.
- Do not apply cursor filtering before constructing capped vocabulary; do not apply page limits before visibility. Deduplicate reliable and corrected identities before lookahead/page selection.
- Run focused checks for each changed behavior, then required backend/Flutter regressions once the complete feature is green. Do not add unrelated tests or repeated broad runs without a new concern.
- Out of scope: Performance metrics, latency/query-plan/load measurements, numerical release gates, semantic search, new controls, replies, merged tabs, literal-before-synonym ranking, configuration versioning, cross-deployment continuity, PDS/lexicon changes, production infrastructure and deployment.
- This plan stage writes only this document. No stage commit was authorized.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking | SQL helper correctness and Unicode/tokenizer alignment | Wrong names/forms or edit boundaries | Selected approach: literal plus declared-English documents, PostgreSQL parser and one-edit helper. Verify real DB fixtures before expanding coverage; no extension dependency |
| CPQ-002 | Non-blocking | Confidence/candidate policy could admit weak or suppress useful corrections | Precision or completeness under the accepted finite cap | Confirmed one edit, minimum length four, at most eight, deterministic C lexical order after eligibility; use positive/negative boundary fixtures. Any revision must preserve product rules and update test expectations |
| CPQ-003 | Non-blocking | Migration number can move before implementation | Conflicting filename/schema cutoff | Recheck maximum; use next number and actual preceding production schema for IT-007. No backfill is required with computed documents |
| CPQ-004 | Non-blocking | Concept-wise ranking differs from current combined-vector score | Controlled priority or benchmark regression | Use best alternative/path per concept with primary A/support C/D weights; verify IT-005 and full benchmark, retain literal retrieval and real caption frequency. No promise of literal-before-synonym ordering |
| CPQ-005 | Non-blocking | Query-time matching may inspect substantial data | More database work | Bounds constrain expansion/correction. No performance measurement/tuning deliverable is included; record concrete operational issues if later encountered without silently expanding scope |
| CPQ-006 | Non-blocking | Lifecycle helpers in focused schemas may mask leaks | False confidence in authorization | Mandatory migrated-schema IT-010/IT-014; existing lifecycle predicates authoritative |
| CPQ-007 | Non-blocking | Test-design wording represents fixtures, not executable data yet | Accidental negative/phrase mismatch | Preserve independent cases and field assignments, bound Gherkin examples, realistic distractors and original URI prefixes; do not treat hats as negative for knit |

No blocking product or architecture question remains. Risk stays Medium. DR-001/DR-002/DR-004 are incorporated through the revised tests and execution details. DR-003/GAP-001 are resolved as explicit chosen bounds; GAP-002 is resolved by function-only migration and computed documents. GAP-003/004/005 remain verification responsibilities, not evidence already obtained.

## 12. Handoff To TDD Builder

- Coding plan: `docs/changes/2026-10-08-search-improvements/04-coding-plan.md`.
- TDD execution plan: `05-implementation-plan.md`, to be written using `implement-tdd` only after the user chooses that stage.
- Start with test: IT-001, `TestSearchImprovementsEnglishWordForms`; English sock/socks in each tab and a separate reliable-language contrast.
- Focused command after the existing development test database environment is configured: `cd appview && TEST_DATABASE_REQUIRED=true go test ./internal/api -run '^TestSearchImprovementsEnglishWordForms$' -count=1`.
- Test environment: `just dev-d` starts the compose stack when needed during implementation. `scripts/appview-test` discovers the development database and runs host Go tests; do not run the AppView outside Docker or print connection credentials. `just test` runs the full backend harness. Analogous focused package commands apply to `./internal/db` and `./internal/index`; `just app-test test/search` covers Flutter search regressions. Extend focused naming consistently for new tests; the existing suites still need their own selectors/full run.
- Notes: No runtime checks, migrations or extension probes were executed in this planning stage. First verify the intended failing behavior, then implement the smallest increment. Existing rows, visibility, cursor ordering and serialized privacy canaries are required evidence before calling the feature complete. No performance evidence is required.
- Open risks: Unicode/tokenization, custom one-edit predicate correctness, ranking comparisons and lifecycle-before-cap evidence. They have concrete tests and do not block starting TDD.
- Commit state: This document is uncommitted; no commit/push was requested.
- Next action requires the user's choice: continue to `implement-tdd`, add a manual note, or stop.
