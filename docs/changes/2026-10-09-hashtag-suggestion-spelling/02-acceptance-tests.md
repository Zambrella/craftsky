# Acceptance Test Specification: Hashtag Suggestion Spelling

## 1. Test Strategy

Design status: Revised draft following DR-001–DR-005 in `03-document-review.md`. This document specifies future tests; none have been implemented or executed for this change. The review artifact records the earlier snapshot and has not been rewritten.

Use Go unit tests for selection, extraction, and normalisation rules; real PostgreSQL integration tests for distinct-post aggregation, surface eligibility, lifecycle, recovery, and pagination; and Flutter widget/repository tests for visible spelling, insertion, navigation, and labels. Acceptance scenarios span these automated layers rather than requiring a new end-to-end harness. Regression checks protect existing behaviour. No manual-only checks are needed.

Carry forward requirements assumptions ASM-001–ASM-004: retain the 28-day window, count each post once per exact spelling, use case-sensitive Unicode code-point order for ties, and cover all named discovery surfaces with their existing eligible populations. These remain documented defaults, not newly confirmed decisions.

Existing automation context:

- Go uses `testing`, table-driven cases, and `httptest`. `appview/internal/api/search_store_test.go` uses `testdb.WithSchema`, `searchStoreDDL`, `seedMember`, and `seedPost`, including existing facet/search hashtag ranking and visibility tests.
- `appview/internal/index/craftsky_post_test.go` supplies real-database indexer tests for tag extraction, metadata, replay, updates, and deletion. Its local DDL and the API fixture DDL may need alignment with any future schema change.
- Pure helper coverage exists in `appview/internal/postutil/tags_test.go`, `appview/internal/api/facet_suggestion_test.go`, and `search_ranking_test.go`.
- Flutter uses `flutter_test`, Riverpod overrides, and `FakeSearchRepository` in `app/test/search/search_page_test.dart`; composer coverage exists in `app/test/shared/rich_text/facet_autocomplete_editor_test.dart` and `app/test/feed/widgets/post_composer_sheet_facets_test.dart`.
- Targets below are existing suites unless explicitly marked proposed. A winner helper is not mandated: assign its unit cases to the selected implementation seam; SQL selection remains integration-tested regardless.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-002 | AT-001, AT-002, IT-001 | Acceptance / Integration | Planned |
| FR-001 | AC-003 | AT-003, UT-004, IT-002, REG-001 | Acceptance / Unit / Integration / Regression | Planned |
| FR-002 | AC-001, AC-002, AC-004 | AT-001, AT-002, IT-001, IT-003 | Acceptance / Integration | Planned |
| FR-003 | AC-005 | AT-004, AT-005, IT-004, REG-003 | Acceptance / Integration / Regression | Planned |
| FR-004 | AC-006 | UT-003, IT-005 | Unit / Integration | Planned |
| FR-005 | AC-007 | IT-006 | Integration | Planned; recovery target pending |
| FR-006 | AC-012 | UT-007, IT-009, IT-010 | Unit / Integration | User-approved correction |
| RULE-001 | AC-001, AC-002 | AT-001, AT-002, UT-001, UT-006, IT-001 | Acceptance / Unit / Integration | Planned |
| RULE-002 | AC-008 | UT-003, IT-007 | Unit / Integration | Planned |
| RULE-003 | AC-009 | UT-002, IT-001 | Unit / Integration | Planned |
| RULE-004 | AC-010 | AT-004, IT-003, REG-002, REG-004 | Acceptance / Integration / Regression | Planned |
| NFR-001 | AC-006 | IT-005, IT-006 | Integration | Planned |
| NFR-002 | AC-003, AC-011 | IT-002, IT-008, REG-001 | Integration / Regression | Planned |

## 3. Acceptance Scenarios

### AT-001: Mixed-case spelling wins by usage

Requirement IDs: BR-001, FR-002, RULE-001

Acceptance Criteria: AC-001, AC-004

Priority: Must

Level: Acceptance

Automation Target: `appview/internal/api/search_store_test.go` plus `app/test/search/search_page_test.dart` and `app/test/shared/rich_text/facet_autocomplete_editor_test.dart`.

```gherkin
Scenario: One suggestion combines case variants
  Given four eligible posts use "memademay"
  And five different eligible posts use "MeMadeMay"
  When I request matching hashtag suggestions or results
  Then there is one item for the identity "memademay"
  And its displayed spelling is "MeMadeMay"
  And its combined post count is 9
```

Run API assertions for composer suggestions, search suggestions, hashtag results, and top hashtags. For top hashtags, place all nine posts in the same supported craft and make them valid project posts. Flutter fixtures consume the selected spelling and count rather than independently computing a winner.

### AT-002: Any casing style can win

Requirement IDs: BR-001, FR-002, RULE-001

Acceptance Criteria: AC-002

Priority: Must

Level: Acceptance

Automation Target: `appview/internal/api/search_store_test.go` and `app/test/search/search_page_test.dart`.

```gherkin
Scenario Outline: Display the most-used spelling without a casing preference
  Given <winningPosts> eligible posts use "<winningSpelling>"
  And <otherPosts> different eligible posts use "<otherSpelling>"
  When I view the matching hashtag item
  Then its displayed spelling is "<winningSpelling>"
  And its combined count is <total>

  Examples:
    | winningPosts | winningSpelling | otherPosts | otherSpelling | total |
    | 5            | memademay       | 4          | MeMadeMay     | 9     |
    | 6            | MEMADEMAY       | 4          | MeMadeMay     | 10    |
```

### AT-003: Query casing does not change results

Requirement IDs: FR-001, NFR-002

Acceptance Criteria: AC-003

Priority: Must

Level: Acceptance

Automation Target: `appview/internal/api/search_store_test.go` and `appview/internal/api/search_ranking_test.go`.

```gherkin
Scenario: Equivalent queries have equivalent results
  Given a fixed eligible post population, viewer, and clock
  When I search using "mema", "MeMa", and "MEMA"
  Then the ordered tag identities are identical
  And each identity has the same winning spelling and count
  And exact, prefix, and substring match precedence is unchanged
```

### AT-004: Composer selection preserves displayed spelling and labels

Requirement IDs: FR-003, RULE-004

Acceptance Criteria: AC-005, AC-010

Priority: Must

Level: Acceptance

Automation Target: `app/test/shared/rich_text/facet_autocomplete_editor_test.dart` and `app/test/feed/widgets/post_composer_sheet_facets_test.dart`.

```gherkin
Scenario: Insert the suggested spelling
  Given I type "#mema" in the composer
  And the repository returns "MeMadeMay" with count 9
  When the suggestion appears
  Then its label is "#MeMadeMay"
  And its count label is "9 posts"
  And no new time-window explanation is shown
  When I select it
  Then the editor replaces the active token with "#MeMadeMay "
  And surrounding text is preserved
```

### AT-005: Open the combined hashtag feed

Requirement IDs: FR-003

Acceptance Criteria: AC-005

Priority: Must

Level: Acceptance

Automation Target: `app/test/search/search_page_test.dart` plus `appview/internal/api/search_store_test.go`.

```gherkin
Scenario: Navigate using a mixed-case result
  Given search returns the hashtag "MeMadeMay"
  When I open it from a suggestion, result row, or top-hashtag chip
  Then the tag page heading retains the selected spelling
  And its request addresses the identity "memademay" case-insensitively
  And it returns the same post identities as the lowercase tag path
```

Assert selection/route behaviour with Flutter fakes and actual path normalisation/post membership with Go handler/store tests. Do not assert that suggestion counts equal the size of the tag feed: the existing populations can differ.

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | RULE-001 | AC-001, AC-002 | Winner selection where a pure selection seam exists | TD-001 and reversed/all-caps variants | Highest distinct-post frequency wins; no casing bonus; singleton spelling preserved | `appview/internal/api/facet_suggestion_test.go` or selected pure helper suite |
| UT-002 | RULE-003 | AC-009 | Stable tie-break | TD-002; permuted input order and query casing; tied non-ASCII candidates accepted by existing syntax | Code-point ascending winner; same result for all permutations, independent of locale/database default collation | Same selection seam as UT-001; SQL outcome also covered by IT-001 |
| UT-003 | FR-004, RULE-002 | AC-006, AC-008 | Extract exact spellings and deduplicate contributions | TD-003; post-text facets, structured tags, pattern/material facets; invalid byte ranges and non-tag features | Retain valid observed spelling candidates, deduplicate each exact spelling, retain one normalised identity; invalid/non-tag features do not contribute | `appview/internal/postutil/tags_test.go` |
| UT-004 | FR-001 | AC-003 | Normalised matching retains display spelling | Queries with case differences, leading hash and existing permitted whitespace; TD-005 | Existing identity normalisation/match ranks unchanged; result display spelling not lowercased by response/ranking helpers | `appview/internal/api/search_ranking_test.go`, `search_request_test.go`, `search_response_test.go`, `facet_response_test.go` |
| UT-005 | FR-002 | AC-004 | Flutter deserialisation retains case | Mock response with mixed-case tags on each relevant response shape | Exact returned spelling and count survive mapping; no client winner calculation | `app/test/search/data/search_api_client_test.dart`, `app/test/shared/rich_text/facet_suggestion_repository_test.dart` |
| UT-006 | RULE-001, FR-001 | AC-002, AC-003 | Accepted non-ASCII and uncased tags | TD-008 | Preserve observed spelling and existing identity equivalence; no transliteration or inferred word boundaries | `appview/internal/postutil/tags_test.go` and selected winner seam |

UT-001/UT-002 do not require introducing a helper solely for tests. If selection is entirely SQL, implement their input/output cases through IT-001 and record unit cases as covered at the integration level.

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | BR-001, FR-002, RULE-001, RULE-003 | AC-001, AC-002, AC-004, AC-009 | Real aggregation and JSON responses | TD-001/TD-002; identical eligible project-post populations across surfaces | Query stores and corresponding HTTP handlers at fixed time; vary insertion order | One case-insensitive item; correct winning exact spelling and distinct aggregate count; deterministic ties; camelCase JSON retains spelling | `appview/internal/api/search_store_test.go`, `facet_test.go`, `search_capability_test.go` |
| IT-002 | FR-001, NFR-002 | AC-003 | Ranking unaffected by query/display casing | TD-005 with exact, prefix, substring candidates and counts deliberately opposing relevance | Request lower/mixed/upper queries; vary winner casing without changing counts | Same normalised ordering, counts, and spellings per data state; relevance first, then count, then normalised tag | `appview/internal/api/search_store_test.go` |
| IT-003 | FR-002, RULE-004 | AC-004, AC-010 | Surface eligibility affects both count and winner | TD-004/TD-006; included posts plus excluded posts carrying an otherwise dominant spelling | Query every surface using its own existing eligibility predicates; advance fixed clock past cutoff | Excluded posts influence neither count nor winner; same population yields same spelling; craft groups can have different winners; winner updates as eligible posts age out | `appview/internal/api/search_store_test.go` |
| IT-004 | FR-003 | AC-005 | Mixed-case path retrieves combined feed | TD-001 plus posts covering existing feed filters; same viewer/language/sort | Call tag-post handler with `MeMadeMay`, `memademay`, and `MEMADEMAY` | Equivalent ordered post identities and pagination for equal parameters; existing feed visibility preserved | `appview/internal/api/search_capability_test.go`, `search_store_test.go` |
| IT-005 | FR-004, NFR-001 | AC-006 | Indexer lifecycle and replay | TD-009; canonical records with valid facet byte ranges | Execute the lifecycle sequence below, observing the suggestion after every step; separately process update-before-create on another URI | Exact frequencies, aggregate count, and winner match the lifecycle table; update-before-create contributes one post, not zero or two | `appview/internal/index/craftsky_post_test.go`; use API reads or integration assertions of the chosen projection |
| IT-006 | FR-005, NFR-001 | AC-007, AC-006 | Historical spelling recovery | TD-007 representing legacy normalised projection with verified public source fields; a separately fresh-indexed copy of the same source state | If a migration is selected, apply the real migration to pre-change fixtures; run the selected recovery twice, including a supported interrupted/resumed run if recovery is batched | Recovered and fresh-indexed results agree for every supported source: TD-001 gives MeMadeMay and nine; repetition preserves counts; source records are unchanged; every eligible recoverable post is accounted for | Proposed recovery integration suite beside the selected migration/reindex mechanism; target unresolved (GAP-001) |
| IT-007 | RULE-002 | AC-008 | Distinct counts with overlapping variants | TD-003 plus a second lowercase-only post | Index and query; repeat facets and metadata occurrences | First post alone: aggregate 1, each variant frequency 1. With second post: aggregate 2, lowercase frequency 2, mixed-case frequency 1; lowercase wins | `appview/internal/index/craftsky_post_test.go`, `appview/internal/api/search_store_test.go` |
| IT-008 | NFR-002 | AC-011 | Cursor and limits remain stable | TD-005 with more identities than page size; fixed clock/viewer | Fetch first page; change only source casing, preserving identities/counts/eligibility; resume saved cursor and traverse all pages | Same normalised identity order, no duplicates/omissions, valid saved cursor, unchanged limits; display can change | `appview/internal/api/search_store_test.go`, `search_cursor_test.go` |

### IT-005 lifecycle oracle

Requirement IDs: FR-004, NFR-001. Acceptance Criteria: AC-006. Fixture: TD-009. Hold time, visibility, and other filters fixed; each post has one spelling for this identity.

| Step | Action | Lowercase frequency | Mixed-case frequency | Aggregate | Winner |
|---|---|---|---|---|---|
| 1 | Index four lowercase and five mixed-case posts | 4 | 5 | 9 | `MeMadeMay` |
| 2 | Replay the same URI/CID events | 4 | 5 | 9 | `MeMadeMay` |
| 3 | Update one mixed-case post to lowercase using a new CID | 5 | 4 | 9 | `memademay` |
| 4 | Replay that update | 5 | 4 | 9 | `memademay` |
| 5 | Remove the tag from that updated post | 4 | 4 | 8 | `MeMadeMay` under ASM-003 |
| 6 | Delete the already untagged post | 4 | 4 | 8 | `MeMadeMay` under ASM-003 |
| 7 | Delete another contributing mixed-case post | 4 | 3 | 7 | `memademay` |
| 8 | Repeat that delete | 4 | 3 | 7 | `memademay` |

Assert frequencies at the chosen persistence/selection seam and aggregate/winner through suggestion reads. The fixture must not carry additional occurrences of this identity in structured metadata after the tag-removal step.

### IT-006 recovery verification gate

Requirement IDs: FR-005, NFR-001. Acceptance Criteria: AC-007, AC-006.

Before implementing this case, coding planning must identify recoverable fields for each currently indexed public tag source and bind the mechanism to an executable test target. Populate TD-007 from those verified fields, including post text, structured project tags, and faceted pattern/material metadata. Compare recovery with fresh ingestion of the same source state; do not seed invented mixed-case values into legacy normalised arrays as the only oracle.

If the design uses a schema migration, apply the actual migration to a pre-change schema/data fixture before recovery assertions. If the migration performs recovery itself, run its supported repeat/recovery path rather than pretending that a versioned migration can be applied twice normally. For a separate recovery job, exercise that job twice. Verify completion against all fixture post identities and source locations. If required casing cannot be recovered, retain GAP-001 and revise the requirement with the user; a guessed lowercase fallback is not proof that FR-005 passed.

## 6. Regression Tests

All cases below are planned automated additions or extensions to existing suites; they are not assertions of already passing coverage.

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test / Automation Target |
|---|---|---|---|---|
| REG-001 | Normalisation, relevance, limits, empty results, wildcard escaping | FR-001, NFR-002 | AC-003, AC-011 | Retain focused existing request/ranking/cursor/escape tests; add empty-query and zero-eligible-population assertions with mixed-case data. Targets: `appview/internal/api/search_request_test.go`, `search_ranking_test.go`, `search_cursor_test.go`, `facet_suggestion_test.go`, `search_store_test.go`. |
| REG-002 | Endpoint-specific visibility and top-level content eligibility | RULE-004 | AC-010 | Extend existing facet/search visible-count tests so excluded rows have a dominant alternative casing; include replies, quotes, moderation states, terminal owners, and relationship exclusions where each endpoint already applies them. Target: `appview/internal/api/search_store_test.go`. |
| REG-003 | Author text and normalised recent-search identity | FR-003 | AC-005 | Enter mixed-case text without choosing a suggestion and verify it remains unchanged; verify recent-search payload identity still normalises while selected display label retains case. Targets: `app/test/feed/widgets/post_composer_sheet_facets_test.dart`, `app/test/search/models/recent_search_test.dart`, `appview/internal/api/search_request_test.go`, `search_recent_store_test.go`. |
| REG-004 | Count label wording | RULE-004 | AC-010 | Assert existing localised count labels for zero/one/multiple counts where shown, with no added window explanation. Targets: `app/test/search/search_page_test.dart`, `app/test/shared/rich_text/facet_autocomplete_editor_test.dart`. |

## 7. Test Data

All fixtures use synthetic public content and typed valid identifiers. Use a fixed UTC instant T; regular eligible posts are T minus one day. Build valid source facet byte offsets against UTF-8 text. Do not seed arbitrary unvalidated source spellings that real ingestion could not produce.

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Core frequency examples | Nine distinct URIs: four lowercase and five mixed-case. Separate reversed and all-caps-dominant datasets. Use eligible project posts in one craft for cross-surface comparisons. | AT-001, AT-002, AT-005, UT-001, IT-001, IT-004, IT-005 |
| TD-002 | Ties | Two distinct posts per spelling for `MeMadeMay`, `memademay`, and another valid casing; repeat with non-ASCII candidates and permuted insertions. | UT-002, IT-001 |
| TD-003 | Duplicate and overlapping sources | One post with repeated mixed-case and lowercase facets, plus valid pattern/material facets and lowercase structured project tags; then a second lowercase-only post. | UT-003, IT-007 |
| TD-004 | Eligibility and time | Baseline eligible lowercase posts plus otherwise dominant mixed-case posts excluded one condition at a time. Include T minus 28 days, one instant inside/outside cutoff, replies, quotes, existing moderation/terminal-owner/relationship conditions. | IT-003, REG-002 |
| TD-005 | Ranking and pagination | Exact `sock`, prefix `sockkal`, substring `woolsock`; more than one page of identities, equal-count normalised alphabetical ties, unequal counts opposing match relevance, and casing-only updates. | AT-003, UT-004, IT-002, IT-008, REG-001 |
| TD-006 | Craft grouping | Supported craft A: five mixed-case/two lowercase; craft B: two mixed-case/five lowercase for the same identity; ordinary posts and nonmatching crafts as exclusions from craft-specific results. | IT-003 |
| TD-007 | Legacy recovery | Pre-change normalised tag arrays paired with recoverable public source text/facets and project metadata containing TD-001 spellings; already-recovered and unprocessed subsets. | IT-006 |
| TD-008 | Non-ASCII / uncased | Tags already accepted by current syntax, including accented casing pairs whose current normalisation agrees and an uncased script tag. Do not introduce new equivalence rules. | UT-006 |
| TD-009 | Exact lifecycle outcomes | TD-001 baseline with individually addressable post URIs/CIDs; one mixed-case post is updated then untagged/deleted, and a different mixed-case contributor is deleted. No hidden duplicate tags in project metadata. | IT-005 |

## 8. Manual Checks

None identified. The label, insertion, heading, and navigation behaviours can be covered with existing widget tests. Production rollout verification is a later operational task, not a substitute for these automated cases.

## 9. Test Gaps And Risks

Risk level: Medium. Review is recommended; no blocking product question prevents completing this specification.

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Recovery mechanism and available historical source data are not verified | FR-005, NFR-001 | Storage/recovery design is deferred; IT-006 has an explicit expected result but no executable target yet. | Inventory each indexed public source and verified recoverable fields during coding planning; select mechanism, integration target, and completion evidence using the IT-006 gate. Resolve before recovery implementation/rollout; missing source data requires an explicit requirement revision, not guessed casing. |
| GAP-002 | API field design and winner seam remain undecided | FR-002, RULE-001, RULE-003 | Requirements specify behaviour rather than storage/response implementation. | Bind display-spelling/normalised-identity fields and response/ranking/client mapping stages to concrete targets in coding planning. Preserve IDs and use IT-001/IT-008 as external oracles even if pure winner unit tests move to SQL integration. |
| GAP-003 | Simplified test schemas can conceal migration defects | FR-004, FR-005, NFR-001 | Existing suites frequently construct local DDL. | Align fixture DDL with implementation and exercise the real migration on pre-change data as part of IT-006 if selected. Updated fixture DDL alone is not migration evidence. |
| GAP-004 | Query cost has no agreed performance threshold | FR-002, NFR-002 | Requirements identify latency risk without a measurable budget. | Assess representative query plans in coding planning; do not invent a latency pass/fail threshold in this specification. Functional boundedness remains covered by IT-008. |
| GAP-005 | Requirements assumptions may be revised | RULE-001, RULE-002, RULE-003, RULE-004, FR-002 | Window, overlap voting, tie-break, and surface population defaults remain explicit assumptions. | Review ASM-001–ASM-004; update affected scenarios if the user revises them. |

No implemented coverage is claimed. All 12 Must requirements and 11 acceptance criteria have planned tests; GAP-001 remains a dependency before recovery verification can be executed.

## 10. Out Of Scope

- Forced CamelCase, inferred segmentation, curated spelling, or anti-spam policy.
- Changing the 28-day window, count label, visibility policy, ranking, or structured project-tag syntax.
- Rewriting PDS records, lexicon changes, infrastructure changes, or live production mutations.
- Global spelling resolution for inline post text and direct tag-page headings.
- New broad end-to-end infrastructure, implementation code, executable test files, migrations, commits, or test runs in this stage.

## 11. Handoff To Document Review

- Requirements file: `01-requirements.md`.
- Test specification: `02-acceptance-tests.md`.
- Next review artifact: `03-document-review.md`.
- External Plannotator review, if user-initiated outside this skill: `docs/changes/2026-10-09-hashtag-suggestion-spelling/`.
- Recommended first failing test: IT-001's four-lowercase/five-mixed-case composer suggestion case. Seed real source records through ingestion or populate the chosen source-spelling projection, not just mixed-case legacy tag arrays. Expect `MeMadeMay` and count nine.
- Suggested order: core source spelling preservation and winner rules; duplicate/overlap and tie cases; lifecycle/replay; all API surfaces and eligibility; ranking/pagination; recovery; Flutter mapping/insertion/navigation and regression labels.
- Commands discovered for later execution: `just appview-test-unit` for the explicitly incomplete Go unit path; `just test` for full Go validation with development PostgreSQL/MinIO available; `just app-test test/shared/rich_text/facet_autocomplete_editor_test.dart test/feed/widgets/post_composer_sheet_facets_test.dart test/search/search_page_test.dart` for focused Flutter widget coverage; `just app-test` for the Flutter suite. Focused host Go commands can use `go test ./internal/api -run '<selected-test-name>'` from `appview/`, with the documented local test database configured and required so integration suites cannot silently skip. No commands are run in this stage.
- Blocking product questions: None. Before implementation/rollout, coding planning must bind API/storage seams and resolve recovery feasibility, completion evidence, and any migration-backed verification in GAP-001–GAP-003. Carry ASM-001–ASM-004 visibly into that plan. Query-plan assessment is required follow-up, but no numeric latency threshold or performance pass is claimed.
- Revision disposition: DR-002 is addressed by the exact IT-005 oracle; DR-001/DR-003 are captured as explicit recovery/migration/API gates; DR-004 assumptions remain visible; DR-005 query-cost work remains a coding-plan follow-up. Technical feasibility has not been verified by document edits.
- Next action: review the revised documents or proceed to coding planning at the user's direction; risk remains medium.

## Approved Correction Tests (2026-10-09, IR-003)

The user explicitly revised the source-parity contract to reject ambiguous recognized fields. These cases supplement the original 23 test IDs.

### UT-007: Reject ambiguous raw source fields

Requirement IDs: FR-006, FR-004. Acceptance criteria: AC-012. Priority: Must. Target: `internal/index/source_validation_test.go` through the shared source validator.

Use original raw JSON literals, not map-marshaled conflicting keys. For create and update, reject both orders of `tag/Tag`, same identity with different spelling, different identities and identical values; reject duplicate exact keys and Unicode field-fold aliases. Cover each recognized nested source boundary. A single alias, multiple independent hashtag features, unknown fields and unknown feature payloads remain valid. Delete processing is unchanged.

### IT-009: Block ambiguous historical recovery

Requirement IDs: FR-006, FR-005. Acceptance criteria: AC-007, AC-012. Priority: Must. Target: `internal/db/hashtag_spellings_migration_test.go`.

Apply the exact migration to pre-update records with detectable case-folded field collisions. Expect a clear error and rollback with source rows and pre-migration schema intact. Cover recognized nested boundaries, including project metadata. Explicitly resolve only the disposable fixture, then verify successful recovery of original spelling. No automatic repair/deletion is permitted. JSONB's lost exact duplicate keys are not claimed recoverable.

### IT-010: Prevent ambiguous fresh projections

Requirement IDs: FR-006, FR-004, FR-002. Acceptance criteria: AC-004, AC-006, AC-012. Priority: Must. Target: real PostgreSQL/indexer ingestion tests.

Reject ambiguous create/update before mutating the post projection. Verify a previously valid contribution remains intact on a direct indexer failure, and that the existing validated pipeline treats invalid sources according to its established behavior. Test the database safeguard against writes that bypass application validation. Keep single-alias real-indexer-to-all-discovery-store coverage green.

Correction order: UT-007 → IT-009 → IT-010 → existing source-parity/lifecycle/discovery regressions → full service-backed Go suite.
