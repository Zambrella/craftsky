# Coding Plan: Hashtag Suggestion Spelling

## 1. Inputs

- Requirements: `01-requirements.md`, revised after document review.
- Tests: `02-acceptance-tests.md`, revised with explicit lifecycle and recovery oracles.
- Document review: `03-document-review.md`, Approved with notes. It records the pre-revision snapshot; this plan carries DR-001–DR-005 forward.
- Confirmed direction: most-used observed spelling overall, case-insensitive search/ranking/counting, existing count labels.
- Defaults retained: ASM-001–ASM-004 (28-day window, distinct-post votes per exact spelling, case-sensitive code-point tie-break, per-surface populations). These are not new user confirmations.
- Repository constraints: public records remain PDS-owned; AppView is a projection; no lexicon, OAuth, infrastructure, or PDS-write change. Reference architecture and API architecture were inspected alongside current implementation; current repository instructions govern development commands where the older reference differs.

## 2. Implementation Strategy

Keep `craftsky_posts.tags` and existing normalised lookup semantics. Add a **stored generated spelling array** derived from the same row's retained public `record` and `is_project` flag. Queries choose the most-used spelling from eligible rows, while aggregate counts, relevance, and pagination continue to use normalised identities.

Proposed database contract:

- Pure immutable SQL function `craftsky_post_tag_spellings(record JSONB, is_project BOOLEAN) -> TEXT[]`.
- Stored generated column `craftsky_posts.tag_spellings`, with an empty array for a valid record without recognised tags. Exact trimmed spellings are deduplicated per post using explicit case-sensitive comparison.
- A new up/down migration adds/removes the function and column. Allocate the next migration number at implementation time; current highest checked-in version is 000088.

This choice recovers existing records when the column is added and automatically derives spellings on later inserts/updates, including writes from an older binary during deployment. Deletes naturally remove their contributions. It avoids a separate backfill CLI, recurring worker, readiness flag, and same-CID replay exception. The generation expression uses only row inputs and an immutable function; PostgreSQL 16 supports stored generated columns with these restrictions. [PostgreSQL 16 documentation](https://www.postgresql.org/docs/16/ddl-generated-columns.html)

The extractor must mirror the existing recognised-source and best-effort facet semantics for records accepted under FR-006. The user-approved validation exception rejects ambiguous recognized field aliases before indexing; it does not reject single aliases or change ordinary tag syntax. It must not be a second broad lexicon validator. SQL/Go parity and exact migration tests are implementation gates; a generated-column design that changes accepted tag contributions or fails on tolerated future facet shapes must be corrected before proceeding, rather than silently weakening the contract.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Acceptance Criteria | Test IDs |
|---|---|---|---|---|---|
| Public source projection | Indexer stores complete `ev.Record` in JSONB | Derive exact spellings from retained source with a generated column | FR-004, FR-005, NFR-001, RULE-002 | AC-006, AC-007, AC-008 | UT-003, IT-005, IT-006, IT-007 |
| Discovery SQL | Inline pgx SQL in FacetStore/SearchStore | Select winner after eligible-post filtering; preserve identity counts and order | BR-001, FR-001, FR-002, RULE-001, RULE-003, RULE-004, NFR-002 | AC-001–AC-004, AC-009–AC-011 | AT-001–AT-003, UT-001, UT-002, IT-001–IT-003, IT-008, REG-001, REG-002 |
| Response/ranking helpers | `tag` string plus count | Treat `tag` as display spelling; normalise only local comparison keys | FR-001, FR-002, NFR-002 | AC-003, AC-004, AC-011 | UT-004, UT-005, IT-001, IT-002, IT-008 |
| Flutter pagination | Exact `tag` string deduplication | Deduplicate by existing lowercase identity, preserving first-seen result/display label | FR-001, NFR-002 | AC-003, AC-011 | IT-008, REG-001 |
| Flutter UI/navigation | Existing repository/provider/editor widgets | Preserve selected spelling through rendering/insertion/routes; no layout or label change | FR-003, RULE-004 | AC-005, AC-010 | AT-004, AT-005, IT-004, REG-003, REG-004 |

## 4. Files And Modules

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/migrations/<next>_hashtag_spellings.up.sql` and `.down.sql` | Create later | Immutable extraction function and generated column; down drops column before function | FR-004, FR-005, RULE-002, NFR-001 | UT-003, UT-006, IT-005–IT-007 |
| `appview/internal/db/hashtag_spellings_migration_test.go` | Create later | Actual migration, recovery, and source-extraction parity verification | FR-004, FR-005, RULE-002, NFR-001 | UT-003, UT-006, IT-006, IT-007 |
| `appview/internal/api/facet_store.go` | Change later | Composer suggestion winner selection with current eligibility | FR-001, FR-002, RULE-001–RULE-004 | AT-001–AT-003, IT-001–IT-003, IT-007 |
| `appview/internal/api/search_hashtag_store.go` | Change later | Search and craft-specific top-hashtag winners, counts and stable normalised sorting | BR-001, FR-001, FR-002, NFR-002, RULE-001–RULE-004 | AT-001–AT-003, IT-001–IT-003, IT-008 |
| `appview/internal/api/search_ranking.go`, hashtag helpers in `facet_store.go` | Change later | Preserve supplied winner and use normalised comparison keys; do not sum overlapping variant frequencies into totals | FR-001, FR-002, RULE-002, NFR-002 | UT-004, IT-002, IT-007, REG-001 |
| `appview/internal/api/search_store_test.go`, `search_capability_test.go`, `facet_test.go` | Change later | Store/handler and serialised response coverage | BR-001, FR-001–FR-003, RULE-001–RULE-004, NFR-002 | AT-001–AT-003, AT-005, IT-001–IT-004, IT-007, IT-008, REG-001, REG-002 |
| `appview/internal/api/search_ranking_test.go`, `facet_suggestion_test.go`, request/response/cursor suites | Change later | Casing preservation, normalised relevance, limits and cursors | FR-001, FR-002, NFR-002 | UT-004, REG-001 |
| `appview/internal/index/craftsky_post_test.go` | Change later | Observe generated projection through source-record create/update/delete/replay | FR-004, RULE-002, NFR-001 | IT-005, IT-007 |
| `appview/internal/postutil/tags_test.go` | Retain/extend later | Protect normalised extraction and byte-range semantics as the parity reference | FR-001, FR-004, RULE-002 | UT-003, UT-006, REG-001 |
| `app/lib/search/providers/search_pagination.dart` | Change later | Compare lowercase identities rather than display strings | FR-001, NFR-002 | REG-001, IT-008 |
| `app/test/search/providers/search_pagination_merge_test.dart` | Change later | Case-only winner changes cannot duplicate an existing result | FR-001, NFR-002 | REG-001, IT-008 |
| Existing Flutter API/repository, search-page, rich-text editor, composer and recent-search test suites named in `02-acceptance-tests.md` | Change later | API mapping, rendering, insertion, navigation, author spelling and labels | FR-002, FR-003, RULE-004 | UT-005, AT-004, AT-005, REG-003, REG-004 |

No new Go indexer record structs, Flutter models, providers, routes, or generic services are required. Production indexer insert/update SQL does not write the generated column directly. Check all explicit column lists and test fixtures for compatibility; change a caller only when a failing test demonstrates the need.

## 5. Services, Interfaces, And Data Flow

### Source inventory and recovery binding (DR-001)

Verified from code: `appview/internal/index/craftsky_post.go` stores `ev.Record` in `craftsky_posts.record` on create and new-CID update. The retained JSONB contains the source fields below; the normalised arrays and flattened metadata are not the recovery authority.

| Source | Retained record fields | Existing eligibility/extraction rule |
|---|---|---|
| Post hashtags | `text`, `facets[].features[].tag` for recognised tag features | Use facet ranges scoped to `text`; ignore non-tag features/invalid ranges |
| Structured project tags | `project.common.tags` | Include only when the indexer materialises a valid project (`is_project` true); lowercase is a legitimate observed spelling |
| Pattern hashtags | `project.common.pattern.name/nameFacets`, `designer/designerFacets`, `publisher/publisherFacets` | Validate each facet range against its sibling string |
| Material hashtags | `project.common.materials[].text/facets` | Validate each facet range against that entry's text |

These exhaust the sources currently merged by `projectSearchTags` and top-level `ExtractTagsForText`. URL fields, colors, design tags, and arbitrary un-faceted text are not new hashtag sources. Replies/quotes do not gain project metadata merely because a raw `project` field exists; `is_project` preserves the indexer's decision.

The function returns observed spellings from recognised features, not guessed capitalisation of visible text. Validate safe numeric byte boundaries against UTF-8 octet lengths and mirror `DecodeFacets`' best-effort behaviour for malformed/unknown arrays. Match `strings.TrimSpace` behaviour for observed tag trimming; default PostgreSQL space-only trimming is insufficient for parity. Arrays and JSON types must be checked before expansion/casts so tolerated source shapes do not turn ordinary writes into failures.

Completion is the successful actual migration plus fixture verification that every recoverable source contributes exactly as fresh ingestion. IT-006 applies prior migrations, seeds legacy record data, applies this exact migration, and compares generated spellings/results with newly inserted identical records. Verify `record`, `cid`, normalised `tags`, and timestamps remain unchanged by recovery. For repeatability, re-read/replay and perform down/up on disposable fixtures; do not reapply an already-versioned up migration as if normal migration tooling permits it. No live production data inventory has been performed: source retention is code-verified, while historical data integrity is a rollout verification dependency.

### Query and interface contract (DR-003)

Keep existing wire shapes: `tag` carries winning observed spelling without `#`; `postsLast28Days` retains the aggregate count; top-hashtag items retain their existing `count` field. Do not add a second normalised field to the API. Inside SQL, name the grouping/ranking key `tag_key`, separate from `display_tag`; handlers scan `display_tag` into the existing `Tag` field. Existing request/path normalisation and recent-search payload identity remain unchanged.

Partial flow sketch:

```text
retained record + is_project -> generated exact spelling set per post
eligible posts -> distinct (post URI, tag_key) -> aggregate post count
same eligible posts + spelling set -> distinct (post URI, tag_key, spelling)
per-spelling distinct-post counts -> highest frequency, spelling COLLATE "C" ASC
tag_key match relevance + aggregate count + existing tag_key sort -> page
winner spelling + aggregate count -> existing DTO tag/count -> Flutter
```

Build all contributions from the same eligible CTE within one query/snapshot. Keep the existing normalised `p.tags` as the tag-identity population and match spellings to those identities using existing normalisation semantics. This prevents a spelling extractor from silently adding search identities. Do not use the number of unnest rows or sum spelling frequencies as the aggregate. Apply limit/offset only after one row per identity exists. Use `COLLATE "C"` for the exact spelling tie-break; retain existing identity collation/order for result ranking.

Scope of eligibility is fixed:

- Composer facets: existing last-28-days, no reply/quote, and `postVisibleModerationPredicate`. Do not add viewer relationship filters absent from this query.
- Hashtag search/search suggestions: same existing time/content/moderation conditions plus `relationshipTopLevelPredicate` for the viewer.
- Top hashtags: existing project and matching-craft conditions plus its time/content/moderation/relationship predicates.
- Tag feed: unchanged post-store query and language/filter logic; it need not share suggestion-count population.

Retain query escaping, count-first secondary ranking, normalised alphabetical tie-break, offset cursor/query binding, and one extra row for `hasMore` where already used. `SearchSuggestionsHandler` already delegates to `SearchHashtags`; no new orchestration service is needed.

`RankHashtagResults` and `NormalizeHashtagSuggestionRows` currently reconstruct lowercase outputs from count maps. Audit their callers: actual stores return aggregate winners, so helpers must preserve a supplied winner and use lowercase keys only for matching/sorting. Remove unsupported variant-count summation semantics or replace unused helpers rather than treating aggregate rows as occurrence votes. Their unit fixtures must reflect the new aggregate contract. Winner selection stays SQL; UT-001/UT-002 bind to IT-001's real SQL inputs/outputs instead of introducing a redundant Go winner implementation.

## 6. State, Providers, Controllers, Or DI

Retain existing Riverpod dependency graphs and asynchronous state:

```text
searchApiClientProvider -> searchRepositoryProvider (keepAlive Provider)
  -> searchSuggestionsProvider (generated async provider)
  -> hashtagResultSearchProvider (paginated async notifier)
  -> topHashtagsProvider / blankSearchProvider
  -> hashtagSearchProvider (tag feed async notifier)

hashtagSuggestionRepositoryProvider -> FacetAutocompleteEditor query/selection
```

No additional provider or DI registration. Repository/model mapping keeps the returned spelling; Flutter does not choose a winner. `appendUniqueHashtags` currently deduplicates exact `tag` strings: use the existing lowercase identity as the deduplication key while retaining the first loaded row/display value. A casing-only change between pages must not create a second identity. Do not introduce a new global spelling cache or alter pagination state/error recovery.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

Preserve existing widget trees. `FacetAutocompleteEditor` renders `#<tag>` and inserts that spelling with current trailing-space behaviour. Search suggestion rows, result tiles, top-hashtag chips, and tag-page headings use their existing returned/navigation tag. Current count localisation and author-entered text remain unchanged.

Keep `/v1/facets/hashtags`, `/v1/search/suggestions`, `/v1/search/hashtags`, `/v1/search/hashtags/top`, and `/v1/search/hashtags/{tag}/posts`; no route registration or error-envelope changes. Existing Go DTOs and Dart mapper fields already carry strings. Test their casing rather than regenerate models without a schema change. Keep recent-search canonical payloads lowercase and display labels as selected.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Empty query / no eligible tags | Existing empty response/state | FR-001, RULE-004 | REG-001 |
| Database query failure | Existing store wrapping, handler envelope and reporting; no payload logging | FR-001, RULE-004 | Existing handler failure suites alongside IT-001 |
| Lowercase/all-caps dominance | Any observed style can win | RULE-001 | AT-002, UT-001, IT-001 |
| Duplicate occurrences/overlapping spellings | Per-post exact-spelling dedup; independent distinct aggregate | RULE-002 | UT-003, IT-007 |
| Tied frequencies | Explicit case-sensitive code-point-compatible comparator | RULE-003 | UT-002/IT-001 |
| Unknown/invalid facets | Match existing best-effort recognised-source behaviour; no new write failures except the explicitly approved FR-006 ambiguity rejection | FR-004, RULE-002 | UT-003, migration parity tests |
| Post edits/deletes/replay | Generated data follows source row; existing same-CID no-op remains | FR-004, NFR-001 | IT-005 |
| Historical source integrity failure | Migration/verification fails visibly; do not guess casing or silently omit required recovery | FR-005 | IT-006 |
| Case-only changes across pages | Stable identity ordering/cursor; client dedup by lowercase identity | NFR-002 | IT-008, REG-001 |

No new logging feature is needed. Any added migration/parse diagnostics must obey `docs/development/logging-error-reporting.md`, preserve typed causes, and exclude raw record bodies, search queries, credentials, and private data.

## 9. Test Implementation Plan

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | IT-001, AT-001 | `internal/api/search_store_test.go` | TD-001 valid retained records and normalised tag arrays; new test `TestHashtagSuggestionsMostUsedSpelling` | Current response is lowercase instead of `MeMadeMay`; total remains nine |
| 2 | UT-003, UT-006, IT-006 | Proposed `internal/db/hashtag_spellings_migration_test.go` | Exact pre-change migrations/legacy rows TD-007; postutil facet fixtures and TD-008 | No generated column/function; no recovered exact spellings |
| 3 | UT-001, UT-002, AT-002 | IT-001 SQL cases in `search_store_test.go` | TD-001 reversed/all-caps, TD-002 ties/permutations | Winner selection/tie comparator missing |
| 4 | IT-005, IT-007 | Indexer suite with migration-backed projection, plus search store suite | TD-009's eight-step lifecycle oracle and TD-003 overlap fixture | Missing contribution changes or inflated aggregate |
| 5 | IT-002, IT-003, AT-003, REG-002 | Search/facet store suite | TD-004–TD-006 with fixed clocks and existing predicates | Winner influenced by exclusions or casing-dependent ranking |
| 6 | UT-004, IT-008, REG-001 | Ranking/request/response/cursor suites; Flutter pagination merge suite | Returned mixed-case winner, identity-order ties, saved cursor and casing-only updates | Helpers recase output or client creates duplicate row |
| 7 | IT-004, AT-005, REG-003 | Capability/request/recent-search suites and Flutter search page | Mixed-case selected tag; compare normalised paths and stable feed filters | Selected spelling/path identity mishandled |
| 8 | UT-005, AT-004, REG-004 | Flutter API/repository/editor/composer/search-page suites | Mixed-case response and count fixtures, existing fakes/localisations | Mapping/insertion recases text or count label changes |

UT-001/UT-002 execute at SQL integration level. UT-003's new spelling extraction also executes at migration/SQL level, with existing Go postutil tests retained as the behavioural reference; there is no unused production Go helper just to satisfy a nominal unit classification. Existing helper regression tests may already pass and should remain protections rather than be forced to fail artificially.

Use `testdb.ApplyMigrations`/`ReadMigration` for exact historical boundary tests and `WithMigratedSchema` for full PostgreSQL 16 schema evidence. Existing simplified `searchStoreDDL` and `craftskyPostsDDL` must include the exact extractor/column definition through the production migration, not a handwritten substitute. New core fixtures require complete public `record` JSON; synthetic lowercase-array-only rows must not be interpreted as evidence of original spelling.

## 10. Sequencing And Guardrails

- First TDD step: add IT-001's composer case against current behaviour; seed source JSON and normalised tags. Confirm the lowercase result is the intended failure.
- Then implement/test source projection before changing query selection. Apply the migration on isolated fixtures only after implementation is authorised. Verify extraction parity, malformed/future facets, structured/project scopes, whitespace and accepted non-ASCII cases before relying on the new column.
- Implement each discovery query with its existing filters and shared aggregation semantics; keep each change paired with its focused failing test.
- Add lifecycle, recovery, ranking/pagination and Flutter verification, then refactor helpers only where their tests establish the required aggregate contract.
- Recovery completion is transactional migration completion with source-parity verification. No CLI job or Render pre-deploy command change is planned: the existing migration path remains responsible for schema application.
- Assess representative query plans and generated-column migration write/lock cost. Adding a stored projection processes existing rows; production rollout requires normal migration review and timing assessment. No live database scan, deployment, or migration is performed in planning.
- Retain existing normalised tag indexes initially. Add an index only if measured plans justify it; do not pre-emptively build a global popularity cache/table that loses per-viewer eligibility.
- Do not modify PDS records, lexicons, credentials, source `record` JSON, normalised tag contents, ranking policy, count window/labels, or visibility rules. No new infrastructure/dependencies.
- Keep one query snapshot for counts/winner/page. Do not fetch unrestricted spellings from excluded posts after calculating eligible counts.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking design; implementation gate | SQL extraction must match Go best-effort facets, Unicode trimming, and source eligibility for unambiguous records; FR-006 rejects duplicate recognized fields | Incorrect contributions or failed ingestion | Exact migration/source-parity tests before query rollout; revise the implementation design if parity cannot be achieved without changing semantics |
| CPQ-002 | Non-blocking rollout dependency | Code retains full records, but live historical integrity/size has not been inspected | Recovery correctness and migration duration | IT-006 proves repository mechanism; deployment review verifies source integrity and table/migration cost. If recovery is infeasible, return to FR-005 revision |
| CPQ-003 | Non-blocking | Database lowercasing/collation and Go/Dart lowercase behaviour can differ for unusual Unicode | Identity/spelling matching drift | Preserve existing identities, test accepted non-ASCII cases across actual PostgreSQL configuration and helpers; do not add new case-folding rules in this change |
| CPQ-004 | Non-blocking | No agreed latency budget | No numerical performance verdict possible | Assess plans with representative tags and sizes; seek a target only if a meaningful trade-off emerges |
| CPQ-005 | Non-blocking defaults | ASM-001–ASM-004 remain assumptions | Tests change if policy changes | Carry unchanged unless user revises; no new question blocks this plan |

DR-001/GAP-001: retained-source inventory and recovery mechanism are bound above. DR-002: use the exact revised IT-005 oracle. DR-003/GAP-002/GAP-003: wire contract, SQL comparator, mapper checks, and exact migration target are specified. DR-004/DR-005: assumptions and query-cost limits remain visible. These are design resolutions, not completed implementation or production verification.

## 12. Handoff To TDD Builder

- Coding plan: `04-coding-plan.md`.
- TDD execution plan: `05-implementation-plan.md`, created only in the authorised implementation stage.
- Start with test: IT-001, proposed Go name `TestHashtagSuggestionsMostUsedSpelling`.
- Focused command: from `appview/`, `go test ./internal/api -run '^TestHashtagSuggestionsMostUsedSpelling$' -count=1`, with the documented local `TEST_DATABASE_URL` configured and `TEST_DATABASE_REQUIRED=true`. Do not mistake skipped PostgreSQL tests for evidence.
- Required service-backed checks: `just test` (development PostgreSQL/MinIO available); focused exact-migration tests in `./internal/db`; `just app-test test/search/providers/search_pagination_merge_test.dart test/shared/rich_text/facet_autocomplete_editor_test.dart test/feed/widgets/post_composer_sheet_facets_test.dart test/search/search_page_test.dart`; expand to the appropriate remaining changed suites. `just appview-test-unit` is explicitly incomplete database evidence.
- No tests, code, migrations, dependencies, commits, or production mutations were created/executed in this planning stage. Only this plan artifact is added.
- Next choice: proceed to `implement-tdd`, add a manual note, or stop.

## Approved IR-003 Correction Design (2026-10-09)

The user approved rejecting ambiguous recognized source fields, including the historical migration check. This explicitly revises CPQ-001's preservation contract only for duplicate recognized fields. Keep the generated spelling projection for unambiguous sources.

- Add a shared raw-JSON ambiguity validator for recognized hashtag-source boundaries. Preserve raw field order/duplicates while examining names with Go case-fold equivalence. Unknown fields/features keep best-effort semantics. Invoke it from shared source validation and guard direct post-indexer writes.
- Add equivalent JSONB ambiguity checking to migration 90 (still unshipped; renumbered after merging main), a clear historical preflight failure and a database constraint to prevent ambiguity during writes by older binaries or direct callers. The migration must leave source data intact and fail transactionally. Down migration removes the constraint and helper.
- Historical conflicting aliases require explicit resolution before migration retry. Never delete/rewrite PDS records automatically. JSONB cannot expose exact duplicate keys already overwritten; document this information limit.
- Test order and mapping: UT-007 / FR-006, FR-004; IT-009 / FR-006, FR-005; IT-010 / FR-006, FR-004, FR-002; then existing regression gates.
- No route, wire, lexicon, auth, permission, UI or production-infrastructure change. Normal source aliases, hashtag spelling variants and independent facet features remain accepted.
