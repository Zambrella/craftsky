# Requirements: First Search Improvements

## 1. Initial Request

The user finds search insufficiently useful, especially for posts. They accepted the proposed first pass: better word matching, bounded typo matching, a small craft vocabulary, broader searchable fields, and a benchmark of realistic queries. They explicitly want Posts and Projects search results to remain separate. This document defines that pass; it does not authorize implementation or production changes.

## 2. Current Codebase Findings

- Relevant files: `appview/internal/api/search_post_store.go`, `search_project_store.go`, `search_store.go`, `search_request.go`, `search_cursor.go`, `search_store_test.go`; `appview/migrations/000019_search_foundation.up.sql`; `app/lib/search/models/search_queries.dart`, `app/lib/search/data/search_api_client.dart`, and `app/lib/search/pages/search_results_tabs.dart`.
- Existing patterns: authenticated AppView-mediated reads, PostgreSQL full-text search, opaque relevance cursors, shared visibility predicates, and separately hydrated post/project responses.
- Submitted post queries use `plainto_tsquery('simple', ...)` against post text and rank with `ts_rank_cd`, then creation time and URI. The `simple` configuration does not perform language stemming. Query normalization currently lowercases and trims input. Plain queries require all surviving terms.
- Posts search excludes project posts and replies. Projects search uses weighted title, pattern name, materials, project tags, design tags and caption. Those existing project fields must remain searchable.
- Ordinary post tags are materialized from authored inline hashtag facets; they are not a separate top-level author-entered tags field. Project tags also include structured project tags and metadata facets. Existing tags and optional authored image alt text are indexed but not included in the submitted post-search vector.
- Public records carry optional author-selected language metadata. The composer defaults to the account primary language; no automatic text-language detection is involved. Language visibility and language-aware matching are distinct.
- Relevance cursors bind lowercased/trimmed query text and result kind, but do not currently bind matching-configuration version. Reliable/typo tiers require an updated internal ordering contract.
- Keyword search is already relevance ordered; this change is not a switch from chronological ordering. Flutter's submitted post/project queries carry query text, not a user-selected sort.
- Visibility checks include viewer relationships, moderation and content-language preferences. Search result hydration includes viewer state and existing post embeds.
- Constraints: retain the `/v1/` JSON/HTTP contract, camelCase bodies, authentication, bounded requests and opaque pagination from `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`. Keep public search projections in AppView; no PDS or lexicon changes are necessary. The reference architecture favors PostgreSQL search without a separate service. Diagnostics must follow `docs/development/logging-error-reporting.md`.
- Test commands for later stages: `just test` against development compose PostgreSQL; focused host Go API tests where appropriate; `just app-test test/search` if Flutter behavior changes. AppView runs inside Docker in development. No tests or runtime changes are made in this requirements stage.

## 3. Clarifying Questions And Decisions

### Q1: Which direction did the user accept?

Answer: The first pass recommended in the preceding discussion.

Decision / implication: Improve lexical retrieval using the existing database, add bounded typo matching and curated craft equivalences, broaden searchable public fields, and verify usefulness with representative examples. Defer embeddings and a separate search service.

### Q2: Should Posts and Projects be combined?

Answer: The user is happy to keep them separate.

Decision / implication: Preserve disjoint submitted-search tabs, routes and response shapes. Do not introduce an All tab. The user confirmed during grilling that matching improvements apply to both submitted-search routes.

### Q3: Are new filters, sort controls or highlighted snippets included?

Answer: They were broader options, not part of the accepted first-pass recommendation.

Decision / implication: Retain the current UI and relevance ordering. New controls, query operators and snippets are outside this slice.

### Grilling review: confirmed decisions (2026-10-09)

The user accepted each recommendation in the interview and subsequently requested that the requirements be updated:

| Interview question | Confirmed decision | Requirement trace |
|---|---|---|
| Q1: Scope | Improve both tabs while keeping results separate | FR-001 through FR-004, RULE-001 |
| Q2: Query concepts | Require every meaningful concept | FR-002, FR-003 |
| Q3: Typo eligibility | Admit strong typo matches below reliable matches even when reliable results exist | FR-003, FR-005 |
| Q4: Languages | English first; retain other-language literal matching | FR-001 |
| Q5: Replies | Keep replies excluded; useful answers existing only in replies remain undiscoverable through this search | RULE-001 |
| Q6: Ordering | All reliable matches precede typo matches; relevance within tiers, recency for ties; no popularity boost | FR-005, FR-006 |
| Q7: Field contribution | Include authored alt text/tags; captions, titles and pattern names carry more weight | FR-004, FR-005 |
| Q8: Correction scope | Correct at most one word; protect short abbreviations/codes | FR-003, NFR-002 |
| Q9: Vocabulary | Small checked-in list: WIP/work in progress, jumper/sweater, stockinette/stocking stitch; exclude associations and ambiguous stitches | FR-002 |
| Q10: Intent | Keyword/topic search without instructional intent; the user clarified that search is not intended to work like Google | RULE-004 |
| Q11: Language selection | Use declared English metadata; no automatic detection; other/unspecified languages retain literal matching | FR-001 |
| Q12: Cross-field matching | All concepts may be satisfied collectively across the record's searchable fields | FR-002, FR-004 |
| Q13: Quotes | Search the record's own fields, not quoted-record bodies | RULE-003 |
| Q14: UI | Preserve entered query; no correction UI | RULE-004 |
| Q15: Evidence | Reviewable realistic positives/negatives in both tabs; top-five checks subject to reliable-first ordering. The performance-measurement portion was subsequently removed by the user during simplification. | BR-001 |

### Simplification review: user-authorized changes (2026-10-09)

- FR-005 / AC-009: Remove the additional guarantee that literal caption/title/name matches outrank approved synonym matches. Keep reliable-before-typo ordering, primary-field weighting, deterministic ties and no popularity/duplicate-tag boost.
- NFR-003 / AC-013: Retire performance measurement entirely, including latency/query-plan assessment, metrics and numerical release gates. This supersedes the performance portion of grilling Q15. Keep the relevance fixture benchmark and bounded matching. These IDs remain retired and shall not be reused.
- FR-006 / EC-007: Keep complete, duplicate-free pagination while data/rules are unchanged; permit restarting after a deployment changes matching rules. Defer matching-configuration versioning and cross-deployment continuity.

These scope changes were explicitly authorized after the simplification review; the earlier grilling approval alone does not cover them. Technical/test-design review must use this revised scope, not restore the retired obligations.

### Document-review follow-up: user-authorized revisions (2026-10-09)

The user chose to revise requirements and acceptance tests following `03-document-review.md`. DR-001 clarifies language-normalized versus typo eligibility; DR-002 makes scenario cases explicit; DR-003 records deterministic, visibility-aware proposed correction bounds; DR-004 makes benchmark cases and distractors concrete. These clarify the existing scope and preserve all stable IDs and retired performance obligations. Numerical defaults remain engineering proposals for coding-plan review.

### Implementation scope confirmation (2026-10-09)

The user approved removing older-post upgrade coverage and creating/testing the local SQL helper migration. No backfill, reindexing or pre-upgrade-record harness is required. Production operations remain outside this authorization.

## 4. Candidate Approaches

### Option A: Improve existing PostgreSQL lexical search

Summary: Normalize language forms, expand curated craft equivalences, search additional public fields and place bounded typo matches after reliable matches, including when reliable results exist.

Pros: Fits existing architecture; explainable behavior; no external model or search-service dependency.

Cons: Does not infer arbitrary meaning; vocabulary and fuzzy thresholds require tuning.

Risks: Over-expansion can reduce precision; added matching paths can increase database work. Keep candidate expansion bounded.

### Option B: Hybrid semantic and keyword search

Summary: Combine embeddings with lexical retrieval to find conceptually related content.

Pros: Can bridge wording differences beyond a curated vocabulary.

Cons: Adds embedding generation, storage, operational cost and relevance tuning.

Risks: Related content can be mistaken for useful answers; extra dependency and lifecycle handling.

Decision: Defer Option B until the first-pass benchmark demonstrates remaining gaps.

## 5. Recommended Direction

Use Option A for submitted Posts and Projects search. Require every meaningful query concept across the record's searchable fields, allowing word variants and approved equivalents to satisfy a concept. Use declared English metadata for English word-form matching; retain literal matching for other or unspecified languages and exact technical names. Rank reliable matches first and strong typo matches second, correcting at most one query word and protecting short abbreviations/codes. Search existing authored hashtags/project tags and image alt text alongside existing fields, weighting captions, titles and pattern names more strongly than supporting fields. Preserve entered queries and the existing UI. Verify positive retrieval, ordering and negative cases with a relevance fixture benchmark. Search remains craft keyword search; it does not infer a request for instructions.

## 6. Problem / Opportunity

Users should find a useful post without knowing its author's exact inflection, regional vocabulary, abbreviation or spelling. Information already published as tags or image descriptions should contribute to discovery. Retrieval must remain precise enough that a search for a particular material and object does not become a general craft feed.

## 7. Goals

- G-001: Improve retrieval for word variations, natural query wording, approved craft equivalents and minor typos.
- G-002: Discover relevant published content through existing public fields while retaining separate result types.
- G-003: Demonstrate useful first-page results with a repeatable relevance benchmark.
- G-004: Preserve visibility, pagination, record ownership and existing search integration.

## 8. Non-Goals

- NG-001: Semantic embeddings, generative query rewriting, AI answers or a separate search service.
- NG-002: Merged Posts/Projects results, an All tab, or searching replies.
- NG-003: New sort/filter controls, highlighted snippets, spelling-suggestion UI, phrase/operator syntax or typeahead changes.
- NG-004: Changes to profiles, hashtag discovery, exact hashtag feeds, project browsing, recent-search persistence or the chronological home feed.
- NG-005: Personalization, popularity-based boosts or inferred interests.
- NG-006: OCR, image recognition, linked-page fetching, video transcription or indexing private drafts/schedules.
- NG-007: PDS writes, lexicon/schema evolution, production deploys or infrastructure changes in this stage.
- NG-008: Google-style question answering, instructional-intent inference, automatic language detection or correcting several query words at once.
- NG-009: Performance metrics, latency or query-plan measurement, load testing and numerical performance release gates for this slice. Existing instrumentation requires no change.
- NG-010: A matching-configuration versioning system or preserving an in-progress search across deployments that change matching rules.
- NG-011: Older-post backfill, reindexing, compatibility layers and pre-upgrade-record test coverage; the app has no users.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Searcher | Authenticated crafter using Posts or Projects search | Useful matches despite ordinary wording variations or minor spelling errors |
| Author | Publishes captions, tags, project details or image descriptions | Existing authored public information discoverable without additional required fields |
| Maintainer | Maintains search vocabulary and relevance behavior | Repeatable fixtures and bounded, understandable matching rules |

## 10. Current Behavior

Literal normalized terms are required in searchable fields. A plural, inflection or abbreviation can fail to match an otherwise useful record. Natural questions retain filler terms under the current simple configuration. Posts only search captions; projects search more structured fields. Existing relevance scoring measures word frequency/proximity but does not repair a missing match. Visibility predicates and stable tie-breakers apply before returning results.

## 11. Desired Behavior

Users can search `sock` and find declared-English content using `socks`, or search `jumper` and find an approved `sweater` equivalent. A query such as `crochett blanket` can retrieve a strong `crochet blanket` correction after all reliable matches, even when reliable results exist. A project can match `blue wool socks` through blue in its caption, wool in its materials and socks in its title; all three concepts must be satisfied. Existing tags and authored image alt text can make a terse post discoverable. `How to knit socks` is a topic query and may return a showcase post; instructional content is not promised. Projects remain in Projects, ordinary top-level posts (including quoting posts matching their own fields) remain in Posts, and replies remain excluded. The entered query is never rewritten in the UI.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Searchers shall retrieve useful results for reviewable realistic benchmark cases within the first five results of the relevant tab, except corrected matches necessarily placed later by reliable-first ordering. Such cases shall verify expected retrieval and ordering separately. | Makes usefulness verifiable | User agreement; grilling Q15 | AC-001 |
| FR-001 | Functional | Must | Submitted search shall match documented English singular/plural and inflection variants for content declared English and ignore documented filler words without discarding meaningful craft terms. Exact technical names and literal matching for non-English or unspecified-language content shall remain available. This language boundary governs English word-form expansion; it does not prohibit independently qualifying typo matches. No automatic language detection shall be introduced. | Avoid wording failures with a clear language boundary | Grilling Q4, Q10, Q11 | AC-002, AC-003 |
| FR-002 | Functional | Must | Both submitted-search routes shall support a finite checked-in dictionary of clear craft equivalences, initially WIP/work in progress, jumper/sweater and stockinette/stocking stitch. Multiword equivalents shall be alternatives for one concept. Every meaningful query concept must match somewhere across the record's searchable fields; concepts shall not be dropped to fill pages. Broader associations and ambiguous stitch abbreviations shall not be treated as equivalents. | Bridge terminology while preserving intent | Grilling Q1, Q2, Q9, Q12 | AC-004 |
| FR-003 | Functional | Must | Search shall admit strong bounded typo matches after reliable literal, language-normalized or approved-vocabulary matches, including when reliable matches exist. Each corrected match shall require correction of at most one query word, preserve all other meaningful concepts and protect short abbreviations/codes such as DK and K2. Without a confident correction, fewer results or none shall be returned. | Recover minor mistakes without broad unrelated matching | Grilling Q3, Q6, Q8 | AC-005, AC-006 |
| FR-004 | Functional | Must | Posts search shall retrieve eligible records through caption, existing authored hashtags and image alt text. Projects search shall retain all existing searchable fields and also retrieve through existing materialized authored tags and image alt text. No new author-entered tag field or generated description shall be required. | Use available public information | Codebase; grilling Q7, Q12 | AC-007, AC-008 |
| FR-005 | Functional | Must | Reliable matches shall precede typo matches. Within each group, rank by relevance, weighting captions, project titles and pattern names above tags and alt text. Break ties by descending creation time then URI; do not boost popularity or duplicate tags. | Preserve agreed ordering and field priorities without requiring literal-versus-synonym ranking | Grilling Q6, Q7; user-approved simplification | AC-009 |
| FR-006 | Functional | Must | While data and matching rules are unchanged, pagination shall be complete and duplicate-free, including the reliable-to-typo transition. Cursors shall reject use with a different query or result type. A deployment that changes matching rules may require restarting the search. | Preserve correct pagination without requiring continuity across matching changes | Codebase; grilling Q3, Q6; user-approved simplification | AC-010 |
| NFR-001 | Non-functional | Must | Search shall use AppView's PostgreSQL projection without an external search/model service and without changing published records or requiring author edits. | Keep operational scope small | Accepted direction; architecture | AC-011 |
| NFR-002 | Non-functional | Must | Typo correction and vocabulary expansion shall have explicit finite bounds documented in test design, retain existing request/result limits, and avoid unbounded retries or candidate expansion. | Bound workload | Discovery; API contract | AC-012 |
| RULE-001 | Business rule | Must | Submitted Posts search shall return only ordinary top-level posts; submitted Projects search shall return only project posts. Replies shall remain excluded. Other search/feed surfaces shall retain their existing behavior. | Preserve user-approved separation | User answer; codebase | AC-014 |
| RULE-002 | Business rule | Must | All retrieval paths and ranking candidates shall apply existing authentication, moderation, account-lifecycle, relationship and content-language visibility rules before candidate eligibility and result selection. Hidden content shall not prevent eligible typo matches or displace visible results. | Protect visibility and useful pages | Codebase; AGENTS.md | AC-015 |
| RULE-003 | Business rule | Must | Search shall index only existing published fields in scope, shall not search private data or quoted-record bodies automatically, and shall not export raw queries or private data to diagnostic sinks or external services. | Preserve privacy boundaries | AGENTS.md; logging guide | AC-016 |
| RULE-004 | Business rule | Must | Existing search routes, authentication headers, request bounds, camelCase response/error contracts and result hydration shall remain intact. Search fetching shall not create new search-history writes. The UI shall preserve entered query text and shall not add correction explanations or new controls. Search shall remain topic/keyword retrieval without inferring instructional intent. | Preserve integration and agreed keyword-search scope | API spec; grilling Q10, Q14 | AC-017 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001 | Given a reviewable realistic fixture set covering exact names, word variants, keyword/natural wording, craft equivalences, typos, added fields and misleading near-matches in both tabs, each designated useful record appears in the first five results, and designated negatives are absent. For a corrected record necessarily displaced by reliable-first ordering, the fixture instead asserts its expected position after all reliable records and retrieval through pagination. Existing exact-match baseline retrieval is preserved. Record each query/tab case separately, with concrete authored fields, language metadata, useful and negative URIs, contrasting distractor content and usefulness rationale. Negatives are query-specific; knitting hats may qualify for knit while failing sock. |
| AC-002 | FR-001 | Given content declared English using socks and knitting, searches using sock and knit retrieve corresponding records; how to knit socks retrieves the designated topic match without requiring filler words. The topic match may be a showcase post and need not contain instructions. |
| AC-003 | FR-001 | Given an exact technical/pattern name and non-English or unspecified-language fixtures, each exact query retrieves its designated record. English word-form expansion applies only to declared-English fixtures, without automatic detection; literal retrieval and existing language visibility remain intact. Verify the reliable language-normalized path separately from typo eligibility: a non-English or unspecified-language record may independently qualify under FR-003. A query with no searchable meaningful terms returns an empty page rather than all records. |
| AC-004 | FR-002 | Given equivalent fixtures, jumper/sweater, WIP/work in progress and stockinette/stocking stitch retrieve their corresponding records in both directions. Given blue wool socks split across a project caption, materials and title, the project qualifies; a record missing any concept does not. Cover cross-field matching in Posts too. Wool shall not expand to yarn, ambiguous US/UK stitch abbreviations shall not be equated, and unlisted terms shall not expand arbitrarily. |
| AC-005 | FR-003 | Given crochett blanket, a strong crochet blanket match is returned, whether reliable matches exist or not, while crochet-only content lacking the blanket concept is excluded. With reliable results, the corrected record follows all of them. Exercise a project pattern-name typo and a record requiring two word corrections, which shall not qualify through typo matching. |
| AC-006 | FR-003 | Given an unrelated or excessively misspelled query without a confident correction, return only qualifying reliable results or an empty page. Short abbreviations/codes such as DK and K2 retain literal matching and do not undergo typo correction. Every accepted corrected match changes at most one query word. |
| AC-007 | FR-004 | Given ordinary posts with existing authored hashtag facets or image alt text contributing the required concepts, the query retrieves each post. Missing tags/images/alt text do not fail search, and no separate author-entered tags field or generated media description is needed. |
| AC-008 | FR-004 | Given projects matching separately by title, pattern name, material, project tag, design tag, caption, existing materialized authored tag or image alt text, each remains retrievable through its designated query. |
| AC-009 | FR-005 | Given an old weak reliable match and a newer stronger typo match, the reliable record appears first. Within each group, comparable caption/title/name matches outrank tag/alt-text-only matches. Equal relevance uses descending creation time then URI. Duplicate tags and increased likes/reposts do not independently boost rank. No separate ranking guarantee distinguishes literal matches from approved synonym matches. |
| AC-010 | FR-006 | Given an unchanged dataset/configuration and a page limit smaller than the result set, traversing reliable-only, synonym, typo-only and mixed-tier queries returns the expected ordered set once, including pages straddling the tier transition and records matching both tiers. Reusing a cursor for another query/tab produces the standard validation error. |
| AC-011 | NFR-001 | Search uses local PostgreSQL without changing published records, PDS writes, author edits or external model/search calls. |
| AC-012 | NFR-002 | Given maximum supported query input, repeated equivalents and adversarial typo input, documented expansion/correction bounds and existing result limits hold, and execution does not perform an unbounded correction loop. |
| AC-014 | RULE-001 | Given ordinary posts, projects and replies with matching content, Posts returns only ordinary top-level posts and Projects only projects. Exact hashtag feeds, project browse, profiles, typeahead and the home feed retain existing behavior. |
| AC-015 | RULE-002 | Given matching muted/blocked authors, moderated or terminal owners and language-excluded records, neither reliable nor typo paths expose those records. Hidden candidates cannot consume the visible page limit, displace visible records or prevent eligible corrected records being retrieved. |
| AC-016 | RULE-003 | Given protected-value canaries in private records and queries, no protected value appears in diagnostic sink output or external requests; private content and text existing only inside a quoted record do not create a match for the quoting post. Existing independently eligible quoted posts may still match their own fields. |
| AC-017 | RULE-004 | Given existing client requests, both APIs retain response/error shapes, auth behavior, limits and hydrated viewer/embed state. Searches cause no additional recent-search write. A typo query remains unchanged in the input and history payload where explicitly saved; no correction explanation/control is introduced. Question-shaped wording is treated as a topic query without an instructional-content guarantee. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Empty/whitespace or oversized query | Preserve current boundary validation; a valid query reduced to no meaningful terms yields an empty page | FR-001, RULE-004 |
| EC-002 | Mixed case, punctuation and repeated equivalent terms | Normalize consistently; no duplicate results or inflated tag contribution | FR-002, FR-005, FR-006 |
| EC-003 | Short craft terms such as `DK`, `KAL` or `K2` | Retain literal usefulness; do not guess arbitrary corrections/expansions | FR-001, FR-003 |
| EC-004 | Mixed/unknown language | Apply English word-form matching only when metadata declares English; preserve literal retrieval and existing visibility for other or unspecified languages | FR-001 |
| EC-005 | Hidden reliable matches and visible corrected matches | Hidden records neither consume the page nor prevent eligible typo results; visible reliable records still rank first | RULE-002, FR-005 |
| EC-006 | Updated/deleted caption, tag or alt text | Existing indexing convergence removes stale searchable content; no separate orphaned search copy | NFR-001, RULE-002 |
| EC-007 | Data or matching rules change during pagination | Completeness applies while data and rules are unchanged; a deployment changing the rules may require a fresh search. No snapshot or cross-deployment continuity guarantee is required | FR-006 |
| EC-008 | Sparse or irrelevant dataset | Return fewer results or none rather than relaxing all meaningful concepts | FR-002, FR-003 |

## 15. Data / Persistence Impact

- New public fields: None. Search uses already published captions, tags, structured project fields and image descriptions.
- Changed fields: Local derived search vectors/indexes may change; dictionary configuration may be added. The schema/index design belongs to the coding plan.
- Migration required: Local SQL helper functions only, as selected in the coding plan. No older-post backfill, reindexing or upgrade compatibility work is required.
- Backwards compatibility: No shipped-client behavior guarantee is required per AGENTS.md, but preserve the current API integration. Production data and migrations still require normal controls. Incompatible cursors may be rejected through existing validation; preserving searches across matching-rule deployments is not required. Cursor mechanics remain an implementation choice.

## 16. UI / API / CLI Impact

- UI: Existing Posts and Projects tabs, empty/error/loading states and cards remain. Preserve the entered query; add no automatic query replacement, spelling suggestion, correction explanation, controls or snippets.
- API: Improve retrieval behind `GET /v1/search/posts` and `GET /v1/search/projects`; keep existing contracts. Matching weights and internal cursor details are not new client fields.
- CLI: None identified.
- Background jobs: No new external processing job required; retain normal published-record indexing and account-removal convergence.

## 17. Security / Privacy / Permissions

- Authentication/authorization: Existing session and viewer visibility policies apply to every matching path.
- Sensitive data: Queries and search history are private user behavior. Do not log raw query/correction text or use it as metric labels. Drafts, schedules, moderation evidence and credentials are excluded.
- Abuse cases: Bound query/candidate expansion and correction attempts; preserve parameterized database access. Repeated tags must not manufacture rank. Broader media analysis is excluded.

## 18. Observability

- Events/logs: Retain existing search operation diagnostics and original typed errors. Empty results are normal, not reportable failures. Any diagnostic change follows the mandatory logging guide and serialized-sink canary checks.
- Metrics: No new metrics, performance measurement or metric-reporting deliverables are required for this slice. The relevance fixture benchmark verifies retrieval and ordering, not execution speed. Existing instrumentation requires no change.
- Alerts: None newly required.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | English stemming damages names or multilingual retrieval | Useful exact content disappears | Retain exact matching and add technical-name/non-English fixtures |
| RISK-002 | Synonyms or fuzzy matching broaden intent | Irrelevant first-page results | Finite approved vocabulary, require all concepts, at most one corrected word, reliable-first tiers and negative fixtures |
| RISK-003 | Added fields/corrections increase database work | Search latency and database load may grow | Retain bounded matching/candidate expansion under NFR-002. Performance measurement is explicitly deferred; reconsider it if operational evidence identifies search latency or database load as a problem |
| RISK-004 | New candidate path bypasses visibility or pagination rules | Content exposure or missing/duplicate results | Shared eligibility rules and route-level integration coverage |
| RISK-005 | Benchmark overfits synthetic examples | Tests pass without practical usefulness | Include realistic craft wording, contrasting negatives, technical names and both result types; expand with user examples later |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | Resolved by grilling Q4/Q11: English word-form matching uses declared language metadata; other/unspecified languages retain literal matching | Confirmed product decision, not an outstanding assumption |
| ASM-002 | Superseded by grilling Q3/Q6: strong typo matches follow reliable matches even when reliable results exist | The former zero-reliable-results condition must not be implemented |
| ASM-003 | Resolved by grilling Q1/Q7/Q12: both submitted tabs gain shared matching and additional authored fields; captions/titles/pattern names carry more weight | Confirmed product decision |
| ASM-004 | Resolved by grilling Q9: small checked-in dictionary, initially three clear equivalences; no admin UI or automatic learning | Confirmed product decision |
| ASM-005 | Resolved by grilling Q10/Q14: unchanged UI/query text and topic search without instructional intent | Confirmed product decision |

## 21. Open Questions

- Blocking product questions: None. The user settled the product decisions through grilling and confirmed the requirements update.
- Engineering follow-up: Finalize correction confidence, finite expansion/candidate bounds and relevance weights within the agreed one-word-correction and reliable-first rules. Test-design defaults are proposals; document deterministic candidate selection after visibility eligibility and ensure hidden candidates cannot consume its cap. These are engineering choices, not performance measurement obligations.
- Engineering follow-up: Choose mixed-result pagination mechanics. Searches may restart after matching-rule deployments; a matching-configuration versioning system is not required.
- Further dictionary entries require explicit review for equivalence; the three initial pairs are settled. No additional entries are required for this slice.

## 22. Review Status

Status: Reviewed

Risk level: Medium

Review recommended: Product decisions reviewed through grilling and simplification; `03-document-review.md` approved coding planning with notes. The user then authorized revisions to this document and the acceptance tests addressing DR-001 through DR-004. The review artifact records the earlier reviewed versions; these follow-up edits do not change agreed product scope.

Reviewer: User, with assistant-led grilling, simplification review and read-only codebase fact checking

Date: 2026-10-09

Notes: The user accepted the grilling recommendations, then explicitly authorized simplifying ranking and deployment continuity and removing performance measurement entirely. Retained IDs are preserved; NFR-003 and AC-013 are retired. The materially revised scope is recorded above and must be used in subsequent review. Requirements describe agreed behavior, not implemented or runtime-verified behavior. Matching thresholds, weights and pagination details remain engineering choices. No code changes, tests, migrations or deployments are authorized by this document update.

## 23. Handoff To Test Design

- Requirements file: `docs/changes/2026-10-08-search-improvements/01-requirements.md`.
- Next test specification: `02-acceptance-tests.md` using `write-acceptance-tests`.
- Must-cover requirement IDs: BR-001; FR-001 through FR-006; NFR-001 and NFR-002; RULE-001 through RULE-004. NFR-003 and AC-013 are retired and must not be restored as test obligations.
- Suggested test levels: normalization/dictionary/bounds unit tests; PostgreSQL-backed reliable/typo retrieval, ranking, lifecycle and cursor integration tests; authenticated handler/response/visibility regressions; existing Flutter search regression checks if the integration changes; an offline relevance fixture benchmark for retrieval and ordering only.
- Test design must enumerate reviewable realistic query/expected-record/negative fixtures, the three approved initial equivalences, deterministic correction thresholds and bounds. Cover mixed reliable/typo pages and tier transitions, cross-field concepts, declared-English versus unspecified/non-English metadata, protected short codes, and unchanged UI query text. Include normal ingestion updates/deletes. Older-post upgrade coverage is excluded because the app has no users.
- Blocking open questions: None. Matching thresholds, weights, query bounds and cursor mechanics remain engineering choices within the confirmed product rules. No performance measurement or release threshold is required.
